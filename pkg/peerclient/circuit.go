package peerclient

// Mode anonyme : toutes les demandes (annuaire, documents, morceaux chez les
// bénévoles) passent par un circuit en oignon à travers 2 ou 3 relais de la
// fédération. Le relais d'entrée connaît l'IP mais ne lit rien ; le relais de
// sortie lit les demandes mais ne connaît ni l'IP ni l'identité, seulement un
// pseudonyme jetable propre au circuit (renouvelé régulièrement).
//
// Échec fermé : sans circuit, aucune demande n'est envoyée en clair à la place.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	mrand "math/rand/v2"
	"sync"
	"time"

	"t2c/pkg/onion"
	"t2c/pkg/peerid"
	"t2c/pkg/wire"
)

// RelayKey : un relais de la fédération (liste signée hosts.json).
type RelayKey struct {
	URL string
	Key ed25519.PublicKey
}

// CircuitLife : durée d'un circuit avant d'en bâtir un nouveau (nouveau pseudonyme).
const CircuitLife = 10 * time.Minute

var ErrNoCircuit = errors.New("anonymat : circuit indisponible")

type circuit struct {
	id   uint64
	mu   sync.Mutex
	hops []*onion.Hop
	anon *peerid.Identity // pseudonyme jetable du circuit
	ctl  chan []byte      // réponses de construction (créé / étendu)
	dead chan struct{}
	once sync.Once
	// sendMu : chiffrer puis envoyer d'un seul tenant (ordre des nonces).
	sendMu sync.Mutex
}

// send chiffre p pour le dernier saut et l'envoie, dans le même ordre.
func (ci *circuit) send(conn *wire.Conn, p []byte) error {
	ci.sendMu.Lock()
	defer ci.sendMu.Unlock()
	return conn.Send(&wire.Frame{T: wire.TCircCell, H: wire.Header{Circ: ci.id}, Data: ci.seal(p)})
}

func (ci *circuit) kill() { ci.once.Do(func() { close(ci.dead) }) }

func (ci *circuit) alive() bool {
	select {
	case <-ci.dead:
		return false
	default:
		return true
	}
}

// seal chiffre une commande pour le dernier saut actuel du circuit.
func (ci *circuit) seal(p []byte) []byte {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	for i := len(ci.hops) - 1; i >= 0; i-- {
		p = ci.hops[i].SealForward(p)
	}
	return p
}

func (ci *circuit) open(c []byte) ([]byte, error) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	var err error
	for _, h := range ci.hops {
		if c, err = h.OpenBackward(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// AnonymityPossible : il faut au moins un autre relais que celui d'entrée.
func AnonymityPossible(entry ed25519.PublicKey, relays []RelayKey) bool {
	return len(entry) == ed25519.PublicKeySize && len(others(entry, relays)) > 0
}

func others(entry ed25519.PublicKey, relays []RelayKey) []ed25519.PublicKey {
	var out []ed25519.PublicKey
	seen := map[string]bool{string(entry): true}
	for _, r := range relays {
		if len(r.Key) == ed25519.PublicKeySize && !seen[string(r.Key)] {
			seen[string(r.Key)] = true
			out = append(out, r.Key)
		}
	}
	return out
}

// path : entrée, puis (milieu) et sortie tirés au hasard parmi les autres relais.
func path(entry ed25519.PublicKey, relays []RelayKey) []ed25519.PublicKey {
	o := others(entry, relays)
	mrand.Shuffle(len(o), func(i, j int) { o[i], o[j] = o[j], o[i] })
	if len(o) > 2 {
		o = o[:2]
	}
	return append([]ed25519.PublicKey{entry}, o...)
}

func (c *Client) currentCircuit() *circuit {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.circ != nil && c.circ.alive() {
		return c.circ
	}
	return nil
}

// buildCircuit bâtit un circuit saut par saut (chaque saut négocie ses clés),
// puis l'ouvre sous un pseudonyme jetable.
func (c *Client) buildCircuit(ctx context.Context) (*circuit, error) {
	hopsKeys := path(c.opt.EntryKey, c.opt.Relays)
	if len(hopsKeys) < 2 {
		return nil, ErrNoCircuit
	}
	var idb [8]byte
	_, _ = rand.Read(idb[:])
	ci := &circuit{id: binary.BigEndian.Uint64(idb[:]) | 1, ctl: make(chan []byte, 1), dead: make(chan struct{})}
	c.mu.Lock()
	c.building[ci.id] = ci
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.building, ci.id)
		c.mu.Unlock()
	}()
	wait := func() ([]byte, error) {
		select {
		case r := <-ci.ctl:
			return r, nil
		case <-ci.dead:
			return nil, ErrNoCircuit
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(15 * time.Second):
			return nil, ErrNoCircuit
		}
	}
	fail := func(err error) (*circuit, error) {
		ci.kill()
		_ = c.conn.Send(&wire.Frame{T: wire.TCircDestroy, H: wire.Header{Circ: ci.id}})
		return nil, err
	}

	hs, err := onion.NewClient(hopsKeys[0])
	if err != nil {
		return nil, err
	}
	if err := c.conn.Send(&wire.Frame{T: wire.TCircCreate, H: wire.Header{Circ: ci.id}, Data: hs.Message()}); err != nil {
		return fail(err)
	}
	r, err := wait()
	if err != nil {
		return fail(err)
	}
	hop, err := hs.Finish(r)
	if err != nil {
		return fail(err)
	}
	ci.hops = []*onion.Hop{hop}

	for _, k := range hopsKeys[1:] {
		hs, err := onion.NewClient(k)
		if err != nil {
			return fail(err)
		}
		cmd := append(append([]byte{onion.CmdExtend}, k...), hs.Message()...)
		if err := ci.send(c.conn, cmd); err != nil {
			return fail(err)
		}
		r, err := wait()
		if err != nil {
			return fail(err)
		}
		hop, err := hs.Finish(r)
		if err != nil {
			return fail(err)
		}
		ci.mu.Lock()
		ci.hops = append(ci.hops, hop)
		ci.mu.Unlock()
	}

	anon, err := peerid.New(0) // jetable : jamais réutilisé, aucune preuve de travail utile
	if err != nil {
		return fail(err)
	}
	ci.anon = anon
	begin := append([]byte{onion.CmdBegin}, anon.ID()...)
	if err := ci.send(c.conn, begin); err != nil {
		return fail(err)
	}
	return ci, nil
}

// installCircuit remplace le circuit courant (l'ancien est fermé).
func (c *Client) installCircuit(ci *circuit) {
	c.mu.Lock()
	old := c.circ
	c.circ = ci
	c.mu.Unlock()
	if old != nil {
		old.kill()
		_ = c.conn.Send(&wire.Frame{T: wire.TCircDestroy, H: wire.Header{Circ: old.id}})
	}
}

// circuitLoop : renouvelle le circuit (et donc le pseudonyme), et le rebâtit s'il tombe.
func (c *Client) circuitLoop() {
	t := time.NewTicker(CircuitLife)
	defer t.Stop()
	for {
		ci := c.currentCircuit()
		var dead <-chan struct{}
		if ci != nil {
			dead = ci.dead
		}
		select {
		case <-c.done:
			return
		case <-t.C:
		case <-dead:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		if nc, err := c.buildCircuit(ctx); err == nil {
			c.installCircuit(nc)
		}
		cancel()
	}
}

// onCircuitFrame : trames de circuit reçues du relais d'entrée.
func (c *Client) onCircuitFrame(f *wire.Frame) {
	c.mu.Lock()
	ci := c.building[f.H.Circ]
	if ci == nil && c.circ != nil && c.circ.id == f.H.Circ {
		ci = c.circ
	}
	c.mu.Unlock()
	if ci == nil {
		return
	}
	switch f.T {
	case wire.TCircCreated:
		select {
		case ci.ctl <- f.Data:
		default:
		}
	case wire.TCircDestroy:
		ci.kill()
	case wire.TCircCell:
		p, err := ci.open(f.Data)
		if err != nil || len(p) == 0 {
			ci.kill()
			return
		}
		switch p[0] {
		case onion.CmdExtended:
			select {
			case ci.ctl <- p[1:]:
			default:
			}
		case onion.CmdFrame:
			if inner, err := wire.Decode(p[1:]); err == nil {
				c.handle(inner)
			}
		}
	}
}

// sendData : requêtes et échanges avec les bénévoles. En mode anonyme,
// uniquement par le circuit (jamais en clair à la place).
func (c *Client) sendData(f *wire.Frame) error {
	if !c.opt.Anonymous {
		return c.conn.Send(f)
	}
	ci := c.currentCircuit()
	if ci == nil {
		return ErrNoCircuit
	}
	enc, err := wire.Encode(f)
	if err != nil {
		return err
	}
	return ci.send(c.conn, append([]byte{onion.CmdFrame}, enc...))
}

// e2eIdentity : identité présentée aux bénévoles (pseudonyme jetable en mode anonyme).
func (c *Client) e2eIdentity() (*peerid.Identity, error) {
	if !c.opt.Anonymous {
		return c.id, nil
	}
	ci := c.currentCircuit()
	if ci == nil || ci.anon == nil {
		return nil, ErrNoCircuit
	}
	return ci.anon, nil
}

// Anonymous : vrai si les demandes passent par un circuit actif.
func (c *Client) Anonymous() bool { return c.opt.Anonymous && c.currentCircuit() != nil }

// CircuitHops : nombre de relais du circuit actif (0 sans circuit).
func (c *Client) CircuitHops() int {
	ci := c.currentCircuit()
	if ci == nil {
		return 0
	}
	ci.mu.Lock()
	defer ci.mu.Unlock()
	return len(ci.hops)
}
