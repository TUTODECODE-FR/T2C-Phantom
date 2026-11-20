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
	"strings"
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
	url    string // adresse publique vérifiée (relais d'entrée possible), vide sinon
}

// isOfficial : clé d'un relais de la liste signée par la racine.
func isOfficial(k ed25519.PublicKey, relays []RelayKey) bool {
	return slices.ContainsFunc(relays, func(r RelayKey) bool { return r.Key.Equal(k) })
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
	entryShare := c.policy.EntryShare()
	banned := c.policy.Banned(hex.EncodeToString(entry))
	c.mu.Unlock()
	mode := c.anonMode()
	if !isOfficial(entry, c.opt.Relays) {
		// consignes (urgence, exclu, plus d'entrée bénévole) ou mode Maximal
		// (entrée toujours officielle) : on quitte cette entrée bénévole
		if share == 0 || entryShare == 0 || banned || mode == ModeMaximal {
			go c.Close()
			return nil, ErrNoCircuit
		}
		return c.buildFromVolunteerEntry(ctx, official[:2])
	}
	if share < 100 && mrand.IntN(100) >= share { // consignes : ce circuit reste sur les officiels
		return c.officialPath(ctx, official)
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
	if mode != ModeMaximal {
		if v2, ok := pickWeighted(vs, now, func(v volRelay) bool { return v.exit }); ok {
			// un bénévole au milieu ne joint qu'un bénévole PUBLIC (connexion à la
			// demande) : une sortie cachée n'est reliée qu'aux relais officiels
			v1, ok := pickWeighted(vs, now, func(v volRelay) bool { return !v.key.Equal(v2.key) && v.family != v2.family })
			if ok && v2.url != "" {
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
	return c.officialPath(ctx, official)
}

// buildFromVolunteerEntry : entrée par un relais bénévole PUBLIC E (il voit
// l'adresse du membre, rien d'autre). Ensuite, de préférence :
//   - mode Partagé : [E, X, Y] bénévoles publics de familles distinctes, Y sortie ;
//     sinon [E, O, Y] (O officiel au milieu, relié à tous les bénévoles) ;
//   - à défaut, et toujours en mode Maximal : [E, O], sortie officielle.
func (c *Client) buildFromVolunteerEntry(ctx context.Context, official []ed25519.PublicKey) (*circuit, error) {
	entry, o := official[0], official[1]
	vs := c.usableVolunteers(ctx, official)
	fam := ""
	for _, v := range c.volunteers(ctx) {
		if v.key.Equal(entry) {
			fam = v.family
		}
	}
	mode := c.anonMode()
	now := time.Now()
	other := func(v volRelay) bool { return !v.key.Equal(entry) && (fam == "" || v.family != fam) }
	if mode != ModeMaximal {
		y, ok := pickWeighted(vs, now, func(v volRelay) bool { return v.exit && other(v) })
		if ok {
			if x, ok := pickWeighted(vs, now, func(v volRelay) bool {
				return v.url != "" && other(v) && !v.key.Equal(y.key) && v.family != y.family
			}); ok && y.url != "" {
				if ci, err := c.buildPath(ctx, []ed25519.PublicKey{entry, x.key, y.key}); err == nil {
					return ci, nil
				}
				c.badVolunteer(x.key)
				c.badVolunteer(y.key)
			} else if ci, err := c.buildPath(ctx, []ed25519.PublicKey{entry, o, y.key}); err == nil {
				return ci, nil
			} else {
				c.badVolunteer(y.key)
			}
		}
	}
	return c.officialPath(ctx, []ed25519.PublicKey{entry, o})
}

// anonMode : mode d'anonymat en vigueur (réglage modifiable à chaud).
func (c *Client) anonMode() string {
	if c.opt.AnonModeFunc != nil {
		return c.opt.AnonModeFunc()
	}
	return c.opt.AnonMode
}

// officialPath : circuit par les relais officiels keys ; si l'un d'eux est en
// panne (circuit refusé), on se replie sur [entrée, autre officiel], chacun
// des autres essayé à son tour : un officiel arrêté ne doit pas priver les
// membres de service (le tirage de path le place sinon dans presque tous les
// circuits quand la liste ne compte que trois relais).
func (c *Client) officialPath(ctx context.Context, keys []ed25519.PublicKey) (*circuit, error) {
	ci, err := c.buildPath(ctx, keys)
	if err == nil || ctx.Err() != nil {
		return ci, err
	}
	o := others(keys[0], c.opt.Relays)
	mrand.Shuffle(len(o), func(i, j int) { o[i], o[j] = o[j], o[i] })
	for _, k := range o {
		if len(keys) == 2 && k.Equal(keys[1]) {
			continue // déjà essayé
		}
		if ci, e := c.buildPath(ctx, []ed25519.PublicKey{keys[0], k}); e == nil {
			return ci, nil
		}
		if ctx.Err() != nil {
			break
		}
	}
	return nil, err
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

// volunteers : annuaire des relais bénévoles (liste signée par un relais
// officiel de la liste racine, revérifiée ; mise en cache 2 minutes).
//
// Il n'est PAS demandé au relais d'entrée (qui choisirait alors quelle liste
// signée, encore valable, nous montrer) : en mode anonyme, il est demandé par
// le circuit à une SORTIE OFFICIELLE. Demande directe seulement pour amorcer,
// sans circuit encore, et seulement à un relais d'entrée officiel ; une entrée
// bénévole sans circuit n'en sert aucun (premier circuit : officiels seulement).
func (c *Client) volunteers(ctx context.Context) []volRelay {
	c.mu.Lock()
	if time.Since(c.volAt) < 2*time.Minute {
		v := c.vols
		c.mu.Unlock()
		return v
	}
	cached := c.vols
	c.mu.Unlock()
	var out []volRelay
	vctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req := &wire.Frame{T: wire.TGet, H: wire.Header{Kind: wire.KindVolunteers}}
	var r *wire.Frame
	var err error
	ci := c.currentCircuit()
	switch {
	case c.opt.Anonymous && ci != nil:
		if !isOfficial(ci.keys[len(ci.keys)-1], c.opt.Relays) {
			return cached // sortie bénévole : on attend un circuit à sortie officielle
		}
		r, err = c.requestVia(vctx, req, true)
	case len(c.opt.EntryKey) > 0 && !isOfficial(c.opt.EntryKey, c.opt.Relays):
		return nil // entrée bénévole, pas encore de circuit : jamais sa liste à elle
	default:
		r, err = c.requestVia(vctx, req, false)
	}
	if err == nil {
		var vl trust.VolunteerList
		var signer ed25519.PublicKey
		if json.Unmarshal(r.Data, &vl) == nil {
			signer, _ = trust.ParsePublicKey(vl.Relay)
		}
		if signer != nil && isOfficial(signer, c.opt.Relays) && vl.Verify(signer, time.Now()) == nil {
			for _, e := range vl.Volunteers {
				if pk, err := trust.ParsePublicKey(e.Key); err == nil {
					v := volRelay{key: pk, family: e.Family, since: time.Unix(e.Since, 0), exit: e.Exit}
					if strings.HasPrefix(e.URL, "ws://") || strings.HasPrefix(e.URL, "wss://") {
						v.url = e.URL
					}
					out = append(out, v)
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
		if v.key.Equal(c.opt.AvoidRelay) || v.key.Equal(c.opt.EntryKey) || c.volBad[k].After(time.Now()) || c.policy.Banned(k) ||
			slices.ContainsFunc(official, func(k ed25519.PublicKey) bool { return k.Equal(v.key) }) {
			continue
		}
		out = append(out, v)
	}
	return out
}

// PublicVolunteer : relais bénévole qui accepte d'être relais d'entrée
// (adresse publique vérifiée par un relais officiel).
type PublicVolunteer struct {
	Key    ed25519.PublicKey
	URL    string
	Family string
	Since  time.Time
}

// PublicVolunteers : relais d'entrée bénévoles de l'annuaire signé (hors exclus).
func (c *Client) PublicVolunteers(ctx context.Context) []PublicVolunteer {
	var out []PublicVolunteer
	vs := c.volunteers(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, v := range vs {
		if v.url != "" && !c.policy.Banned(hex.EncodeToString(v.key)) && c.policy.VolunteerShare() > 0 && c.policy.EntryShare() > 0 {
			out = append(out, PublicVolunteer{Key: v.key, URL: v.url, Family: v.family, Since: v.since})
		}
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
			if c.currentCircuit() != nil {
				continue // déjà remplacé (RenewCircuit)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		if nc, err := c.buildCircuit(ctx); err == nil {
			c.installCircuit(nc)
		}
		cancel()
	}
}

// RenewCircuit : bâtit tout de suite un nouveau circuit (nouveau pseudonyme)
// et remplace l'actuel. Le premier circuit d'une entrée bénévole ne passe que
// par des officiels (son annuaire n'est jamais demandé à l'entrée) ; les
// suivants peuvent prendre des bénévoles appris par le circuit.
func (c *Client) RenewCircuit(ctx context.Context) error {
	if !c.opt.Anonymous {
		return ErrNoCircuit
	}
	ci, err := c.buildCircuit(ctx)
	if err != nil {
		return err
	}
	c.installCircuit(ci)
	return nil
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
				c.handle(inner, true)
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
