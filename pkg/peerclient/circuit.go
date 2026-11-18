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
	"encoding/hex"
	"encoding/json"
	"errors"
	mrand "math/rand/v2"
	"slices"
	"sync"
	"time"

	"t2c/pkg/onion"
	"t2c/pkg/peerid"
	"t2c/pkg/trust"
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
	keys []ed25519.PublicKey // relais du circuit (entrée → sortie)
	anon *peerid.Identity    // pseudonyme jetable du circuit
	ctl  chan []byte         // réponses de construction (créé / étendu)
	dead chan struct{}
	once sync.Once
	// sendMu : chiffrer puis envoyer d'un seul tenant (ordre des nonces).
	sendMu sync.Mutex
}

// send enveloppe p à une taille fixe, le chiffre pour le dernier saut et
// l'envoie, dans l'ordre de chiffrement.
func (ci *circuit) send(conn *wire.Conn, p []byte) error {
	padded, ok := onion.Pad(p)
	if !ok {
		return errors.New("anonymat : commande trop grande")
	}
	ci.sendMu.Lock()
	defer ci.sendMu.Unlock()
	return conn.Send(&wire.Frame{T: wire.TCircCell, H: wire.Header{Circ: ci.id}, Data: ci.seal(padded)})
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

// Modes d'anonymat.
const (
	// ModePartage (défaut) : la sortie peut être le relais d'un membre ; plus
	// aucun relais de l'association ne voit alors ce qui est demandé.
	ModePartage = "partage"
	// ModeMaximal : la sortie est toujours un relais de l'association.
	ModeMaximal = "maximal"
)

// RampVolunteer : ancienneté au bout de laquelle un relais bénévole est choisi
// à plein régime ; avant, il l'est proportionnellement moins (une armée de
// relais créés d'un coup ne pèse presque rien pendant des semaines).
const RampVolunteer = 14 * 24 * time.Hour

type volRelay struct {
	key    ed25519.PublicKey
	family string
	since  time.Time
	exit   bool
}

func (v volRelay) weight(now time.Time) float64 {
	w := float64(now.Sub(v.since)) / float64(RampVolunteer)
	return min(1, max(0.05, w))
}

// buildCircuit : l'entrée est toujours un relais officiel (le seul à voir
// l'adresse du membre). Ensuite, de préférence :
//   - mode Partagé : bénévole au milieu → bénévole d'une AUTRE famille en sortie ;
//   - sinon : bénévole au milieu → relais officiel en sortie ;
//   - à défaut : relais officiels seulement.
//
// Un bénévole qui ne répond pas est évité 30 minutes et l'on passe au choix suivant.
func (c *Client) buildCircuit(ctx context.Context) (*circuit, error) {
	official := path(c.opt.EntryKey, c.opt.Relays)
	if len(official) < 2 {
		return nil, ErrNoCircuit
	}
	entry, exitOfficial := official[0], official[len(official)-1]
	c.mu.Lock()
	share := c.policy.VolunteerShare()
	c.mu.Unlock()
	if share < 100 && mrand.IntN(100) >= share { // consignes : ce circuit reste sur les officiels
		return c.buildPath(ctx, official)
	}
	vs := c.usableVolunteers(ctx, official)
	now := time.Now()
	try := func(keys []ed25519.PublicKey, vols ...volRelay) (*circuit, bool) {
		ci, err := c.buildPath(ctx, keys)
		if err == nil {
			return ci, true
		}
		for _, v := range vols {
			c.badVolunteer(v.key)
		}
		return nil, false
	}
	mode := c.opt.AnonMode
	if c.opt.AnonModeFunc != nil {
		mode = c.opt.AnonModeFunc()
	}
	if mode != ModeMaximal {
		if v2, ok := pickWeighted(vs, now, func(v volRelay) bool { return v.exit }); ok {
			if v1, ok := pickWeighted(vs, now, func(v volRelay) bool { return !v.key.Equal(v2.key) && v.family != v2.family }); ok {
				if ci, ok := try([]ed25519.PublicKey{entry, v1.key, v2.key}, v1, v2); ok {
					return ci, nil
				}
			} else if ci, ok := try([]ed25519.PublicKey{entry, v2.key}, v2); ok {
				return ci, nil
			}
		}
	}
	if v1, ok := pickWeighted(c.usableVolunteers(ctx, official), now, func(volRelay) bool { return true }); ok {
		if ci, ok := try([]ed25519.PublicKey{entry, v1.key, exitOfficial}, v1); ok {
			return ci, nil
		}
	}
	return c.buildPath(ctx, official)
}

// pickWeighted : tirage au hasard pondéré par l'ancienneté.
func pickWeighted(vs []volRelay, now time.Time, ok func(volRelay) bool) (volRelay, bool) {
	var cand []volRelay
	total := 0.0
	for _, v := range vs {
		if ok(v) {
			cand = append(cand, v)
			total += v.weight(now)
		}
	}
	if len(cand) == 0 {
		return volRelay{}, false
	}
	r := mrand.Float64() * total
	for _, v := range cand {
		if r -= v.weight(now); r <= 0 {
			return v, true
		}
	}
	return cand[len(cand)-1], true
}

// volunteers : relais bénévoles annoncés par le relais d'entrée (liste signée
// par lui, revérifiée ; mise en cache 2 minutes).
func (c *Client) volunteers(ctx context.Context) []volRelay {
	c.mu.Lock()
	if time.Since(c.volAt) < 2*time.Minute {
		v := c.vols
		c.mu.Unlock()
		return v
	}
	c.mu.Unlock()
	var out []volRelay
	vctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if b, err := c.get(vctx, wire.KindVolunteers, "", 0); err == nil {
		var vl trust.VolunteerList
		if json.Unmarshal(b, &vl) == nil && vl.Relay == hex.EncodeToString(c.opt.EntryKey) &&
			vl.Verify(c.opt.EntryKey, time.Now()) == nil {
			for _, e := range vl.Volunteers {
				if pk, err := trust.ParsePublicKey(e.Key); err == nil {
					out = append(out, volRelay{key: pk, family: e.Family, since: time.Unix(e.Since, 0), exit: e.Exit})
				}
			}
		}
	}
	c.mu.Lock()
	c.vols, c.volAt = out, time.Now()
	c.mu.Unlock()
	return out
}

// usableVolunteers : sans les relais officiels, notre propre relais caché et
// ceux qui n'ont pas répondu récemment.
func (c *Client) usableVolunteers(ctx context.Context, official []ed25519.PublicKey) []volRelay {
	vs := c.volunteers(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []volRelay
	for _, v := range vs {
		k := hex.EncodeToString(v.key)
		if v.key.Equal(c.opt.AvoidRelay) || c.volBad[k].After(time.Now()) || c.policy.Banned(k) ||
			slices.ContainsFunc(official, func(k ed25519.PublicKey) bool { return k.Equal(v.key) }) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (c *Client) badVolunteer(v ed25519.PublicKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.volBad == nil {
		c.volBad = map[string]time.Time{}
	}
	c.volBad[hex.EncodeToString(v)] = time.Now().Add(30 * time.Minute)
}

// buildPath bâtit un circuit saut par saut (chaque saut négocie ses clés),
// puis l'ouvre sous un pseudonyme jetable.
func (c *Client) buildPath(ctx context.Context, hopsKeys []ed25519.PublicKey) (*circuit, error) {
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
	ci.keys = hopsKeys

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

// coverLoop : cellules leurres à intervalles aléatoires (2 à 8 s), indiscernables
// des vraies pour les relais d'entrée et intermédiaires : le rythme du trafic
// ne trahit plus l'activité réelle.
func (c *Client) coverLoop() {
	for {
		wait := 2*time.Second + time.Duration(mrand.Int64N(int64(6*time.Second)))
		select {
		case <-c.done:
			return
		case <-time.After(wait):
		}
		if ci := c.currentCircuit(); ci != nil {
			_ = ci.send(c.conn, []byte{onion.CmdDrop})
		}
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
		if inner, ok := onion.Unpad(p); ok {
			p = inner
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

// CircuitKeys : clés des relais du circuit actif, de l'entrée à la sortie.
func (c *Client) CircuitKeys() []ed25519.PublicKey {
	if ci := c.currentCircuit(); ci != nil {
		return ci.keys
	}
	return nil
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
