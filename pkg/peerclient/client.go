// Package peerclient : côté pair. Connexion au VPS (via Cloudflare) avec une
// identité pseudonyme, récupération des documents signés, téléchargement des
// sites en parallèle chez les bénévoles (chiffré de bout en bout, chaque
// morceau vérifié), repli sur l'origine, et mode bénévole (héberger une copie).
package peerclient

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"

	"t2c/pkg/peerid"
	"t2c/pkg/trust"
	"t2c/pkg/wire"
)

type Options struct {
	RootKey ed25519.PublicKey
	Store   *ContentStore       // nil = ne pas héberger
	Guard   *trust.VersionGuard // nil = garde en mémoire
	Workers int                 // téléchargements parallèles (défaut 6)
	Header  http.Header         // headers de la requête WebSocket (tests : simuler Cloudflare)
	// UploadBytesPerSec : débit maximal envoyé aux autres pairs (0 = illimité).
	UploadBytesPerSec int64
	// Anonymous : demandes par circuit en oignon (EntryKey = clé du relais
	// auquel on se connecte, Relays = relais de la liste signée).
	Anonymous bool
	EntryKey  ed25519.PublicKey
	Relays    []RelayKey
	// OnServe (bénévole) : appelé avec l'identité présentée par chaque demandeur
	// servi (vérification de l'anonymat, statistiques). Facultatif.
	OnServe func(from string)
}

type Client struct {
	id    *peerid.Identity
	opt   Options
	conn  *wire.Conn
	rep   *Reputation
	guard *trust.VersionGuard

	mu        sync.Mutex
	nextQ     uint64
	pending   map[uint64]chan *wire.Frame
	outs      map[uint64]*outSession
	tickets   map[string]trust.Ticket // racine → ticket
	closed    bool
	done      chan struct{}
	serveSem  chan struct{}
	upload    *rateLimiter
	served    atomic.Int64 // octets servis à d'autres pairs
	circ      *circuit     // circuit en oignon actif (mode anonyme)
	originSem chan struct{}
	building  map[uint64]*circuit
}

type outSession struct {
	peer string
	hs   *noise.HandshakeState
	cs   *noise.CipherState // réception (répondeur → initiateur)
	asm  responseAssembler
	max  int
	ch   chan outResult
}

type outResult struct {
	data []byte
	err  error
}

// Dial ouvre la WebSocket et réalise la poignée de main pseudonyme.
func Dial(ctx context.Context, url string, id *peerid.Identity, opt Options) (*Client, error) {
	if len(opt.RootKey) != ed25519.PublicKeySize {
		return nil, errors.New("peerclient: clé racine manquante")
	}
	if opt.Workers == 0 {
		opt.Workers = 6
	}
	d := *websocket.DefaultDialer
	d.Subprotocols = []string{wire.Subprotocol}
	ws, _, err := d.DialContext(ctx, url, opt.Header)
	if err != nil {
		return nil, err
	}
	c := &Client{
		id: id, opt: opt, conn: wire.NewConn(ws), rep: NewReputation(),
		pending: map[uint64]chan *wire.Frame{}, outs: map[uint64]*outSession{},
		tickets: map[string]trust.Ticket{}, done: make(chan struct{}), serveSem: make(chan struct{}, 8),
		upload: newRateLimiter(opt.UploadBytesPerSec), building: map[uint64]*circuit{},
		originSem: make(chan struct{}, originSlots),
	}
	c.guard = opt.Guard
	if c.guard == nil {
		c.guard = trust.NewVersionGuard("")
	}
	if err := c.handshake(); err != nil {
		_ = c.conn.Close()
		return nil, err
	}
	go c.readLoop()
	if opt.Anonymous {
		// échec fermé : sans circuit, pas de connexion utilisable en mode anonyme
		ci, err := c.buildCircuit(ctx)
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		c.installCircuit(ci)
		go c.circuitLoop()
	}
	return c, nil
}

func (c *Client) handshake() error {
	f, err := c.conn.Recv()
	if err != nil {
		return err
	}
	if f.T != wire.TChallenge || len(f.H.Nonce) != 32 {
		return errors.New("défi attendu")
	}
	if c.id.MintBits < f.H.MintBits {
		return fmt.Errorf("le relais exige une PoW d'identité de %d bits", f.H.MintBits)
	}
	err = c.conn.Send(&wire.Frame{T: wire.THello, H: wire.Header{
		Pub: c.id.Pub, Mint: c.id.MintNonce, MintBits: c.id.MintBits,
		NoisePub: c.id.Noise.PublicKey().Bytes(), NoiseSig: c.id.NoiseSig,
		Sig: ed25519.Sign(c.id.Priv, wire.HelloPayload(f.H.Nonce)),
	}})
	if err != nil {
		return err
	}
	f, err = c.conn.Recv()
	if err != nil {
		return err
	}
	if f.T == wire.TError {
		return errors.New("relais: " + f.H.Err)
	}
	if f.T != wire.TWelcome || f.H.ID != c.id.ID() {
		return errors.New("bienvenue attendue")
	}
	return nil
}

func (c *Client) ID() string              { return c.id.ID() }
func (c *Client) Reputation() *Reputation { return c.rep }
func (c *Client) Done() <-chan struct{}   { return c.done }

func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return c.conn.Close()
}

func (c *Client) readLoop() {
	defer func() {
		c.mu.Lock()
		for q, ch := range c.pending {
			close(ch)
			delete(c.pending, q)
		}
		for sid, o := range c.outs {
			o.ch <- outResult{err: errors.New("déconnecté")}
			delete(c.outs, sid)
		}
		c.mu.Unlock()
		close(c.done)
	}()
	for {
		f, err := c.conn.Recv()
		if err != nil {
			if ci := c.currentCircuit(); ci != nil {
				ci.kill()
			}
			return
		}
		switch f.T {
		case wire.TCircCreated, wire.TCircCell, wire.TCircDestroy:
			c.onCircuitFrame(f)
		case wire.TAudit:
			go c.onAudit(f)
		default:
			c.handle(f)
		}
	}
}

// handle : trame reçue directement ou extraite d'un circuit.
func (c *Client) handle(f *wire.Frame) {
	switch f.T {
	case wire.TFwd:
		c.onFwd(f)
	case wire.TError:
		if f.H.SID != 0 {
			c.failOut(f.H.SID, errors.New(f.H.Err))
			return
		}
		c.deliver(f)
	default:
		c.deliver(f)
	}
}

func (c *Client) deliver(f *wire.Frame) {
	c.mu.Lock()
	ch := c.pending[f.H.Q]
	delete(c.pending, f.H.Q)
	c.mu.Unlock()
	if ch != nil {
		ch <- f
	}
}

// request : envoie une trame et attend la réponse au même Q.
func (c *Client) request(ctx context.Context, f *wire.Frame) (*wire.Frame, error) {
	ch := make(chan *wire.Frame, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errors.New("fermé")
	}
	c.nextQ++
	q := c.nextQ
	c.pending[q] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, q)
		c.mu.Unlock()
	}()
	f.H.Q = q
	send := c.conn.Send
	if f.T == wire.TFind || f.T == wire.TGet {
		send = c.sendData // demandes de consultation : par le circuit en mode anonyme
	}
	if err := send(f); err != nil {
		return nil, err
	}
	select {
	case r, ok := <-ch:
		if !ok {
			return nil, errors.New("déconnecté")
		}
		if r.T == wire.TError {
			return nil, errors.New("relais: " + r.H.Err)
		}
		return r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// --- documents signés ----------------------------------------------------------

func (c *Client) get(ctx context.Context, kind, root string, index int) ([]byte, error) {
	r, err := c.request(ctx, &wire.Frame{T: wire.TGet, H: wire.Header{Kind: kind, Root: root, Index: index}})
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// Catalog : catalogue signé par l'origine, refusé s'il est plus ancien que le dernier vu.
func (c *Client) Catalog(ctx context.Context) (*trust.Catalog, error) {
	b, err := c.get(ctx, wire.KindCatalog, "", 0)
	if err != nil {
		return nil, err
	}
	var cat trust.Catalog
	if err := json.Unmarshal(b, &cat); err != nil {
		return nil, err
	}
	if err := cat.Verify(c.opt.RootKey); err != nil {
		return nil, err
	}
	return &cat, c.guard.Accept("catalog", cat.Version)
}

// Hosts : liste signée des bénévoles / bannis (nil si l'origine n'en publie pas encore).
func (c *Client) Hosts(ctx context.Context) (*trust.HostsList, error) {
	b, err := c.get(ctx, wire.KindHosts, "", 0)
	if err != nil {
		return nil, err
	}
	var h trust.HostsList
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, err
	}
	if err := h.Verify(c.opt.RootKey); err != nil {
		return nil, err
	}
	return &h, c.guard.Accept("hosts", h.Version)
}

func (c *Client) Manifest(ctx context.Context, root string) (*trust.Manifest, error) {
	b, err := c.get(ctx, wire.KindManifest, root, 0)
	if err != nil {
		return nil, err
	}
	var m trust.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if err := m.Verify(c.opt.RootKey); err != nil {
		return nil, err
	}
	if m.Root != root {
		return nil, errors.New("manifeste d'une autre racine")
	}
	return &m, nil
}

// OriginChunk : morceau via VPS → origine (repli quand aucun bénévole ne l'a).
// originSlots : demandes simultanées à l'origine par connexion (le relais en
// accepte 4 ; au-delà il répond « trop de requêtes simultanées »).
const originSlots = 3

// OriginChunk : morceau servi par l'origine (via le relais). Si le relais est
// momentanément saturé, on patiente et on réessaie au lieu d'échouer.
func (c *Client) OriginChunk(ctx context.Context, root string, i int) ([]byte, error) {
	select {
	case c.originSem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-c.originSem }()
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		var data []byte
		if data, err = c.get(ctx, wire.KindChunk, root, i); err == nil || !strings.Contains(err.Error(), "simultanées") {
			return data, err
		}
		select {
		case <-time.After(time.Duration(200*(attempt+1)) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, err
}

// Find : bénévoles d'une racine, chaque fiche vérifiée (ID ↔ clé ↔ clé Noise).
func (c *Client) Find(ctx context.Context, root string) ([]wire.HostInfo, error) {
	r, err := c.request(ctx, &wire.Frame{T: wire.TFind, H: wire.Header{Root: root}})
	if err != nil {
		return nil, err
	}
	var out []wire.HostInfo
	for _, h := range r.H.Hosts {
		if len(h.Pub) == ed25519.PublicKeySize && peerid.IDFromPub(h.Pub) == h.Peer &&
			peerid.VerifyNoise(h.Pub, h.NoisePub, h.NoiseSig) && h.Peer != c.ID() {
			out = append(out, h)
		}
	}
	return out, nil
}

// Report : signale au VPS un morceau faux reçu d'un pair (il sera audité).
func (c *Client) Report(peer, root string, i int) {
	if c.opt.Anonymous {
		return // signaler révélerait ce qu'on consulte ; le bannissement local suffit
	}
	_ = c.conn.Send(&wire.Frame{T: wire.TReport, H: wire.Header{Peer: peer, Root: root, Index: i}})
}

// --- échanges chiffrés entre pairs ---------------------------------------------

type chunkReq struct {
	Root  string `json:"root"`
	Index int    `json:"index"`
}

// FetchFromPeer : demande le morceau i à un bénévole (Noise IK via le VPS).
// Le résultat n'est PAS vérifié ici : l'appelant le compare au manifeste.
func (c *Client) FetchFromPeer(ctx context.Context, h wire.HostInfo, root string, i int) ([]byte, error) {
	me, err := c.e2eIdentity()
	if err != nil {
		return nil, err
	}
	hs, err := newInitiator(me, h.Peer, h.NoisePub)
	if err != nil {
		return nil, err
	}
	req, _ := json.Marshal(chunkReq{Root: root, Index: i})
	msg1, _, _, err := hs.WriteMessage(nil, req)
	if err != nil {
		return nil, err
	}
	var sidb [8]byte
	_, _ = rand.Read(sidb[:])
	sid := binary.BigEndian.Uint64(sidb[:]) | 1
	o := &outSession{peer: h.Peer, hs: hs, max: trust.ChunkSize, ch: make(chan outResult, 1)}
	c.mu.Lock()
	c.outs[sid] = o
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.outs, sid)
		c.mu.Unlock()
	}()
	if err := c.sendData(&wire.Frame{T: wire.TFwd, H: wire.Header{To: h.Peer, SID: sid}, Data: msg1}); err != nil {
		return nil, err
	}
	select {
	case r := <-o.ch:
		return r.data, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) failOut(sid uint64, err error) {
	c.mu.Lock()
	o := c.outs[sid]
	delete(c.outs, sid)
	c.mu.Unlock()
	if o != nil {
		o.ch <- outResult{err: err}
	}
}

func (c *Client) onFwd(f *wire.Frame) {
	c.mu.Lock()
	o := c.outs[f.H.SID]
	c.mu.Unlock()
	if o != nil && o.peer == f.H.From {
		c.onResponsePart(f.H.SID, o, f.Data)
		return
	}
	// sinon : nouvelle demande entrante (nous sommes bénévole)
	if c.opt.Store == nil {
		return
	}
	select {
	case c.serveSem <- struct{}{}:
		go func() {
			defer func() { <-c.serveSem }()
			c.serve(f.H.From, f.H.SID, f.Data)
		}()
	default: // surchargé : on ignore, le demandeur passera à un autre bénévole
	}
}

func (c *Client) onResponsePart(sid uint64, o *outSession, data []byte) {
	var plain []byte
	var err error
	if o.cs == nil {
		var cs2 *noise.CipherState
		plain, _, cs2, err = o.hs.ReadMessage(nil, data)
		o.cs = cs2
	} else {
		plain, err = o.cs.Decrypt(nil, nil, data)
	}
	if err == nil {
		var done bool
		done, err = o.asm.add(plain, o.max)
		if err == nil && done {
			st, body := o.asm.result()
			if st != stOK {
				err = errors.New("pair: " + string(body))
			}
			c.mu.Lock()
			delete(c.outs, sid)
			c.mu.Unlock()
			o.ch <- outResult{data: body, err: err}
			return
		}
	}
	if err != nil {
		c.failOut(sid, err)
	}
}

// serve : répond à une demande chiffrée de morceau.
func (c *Client) serve(from string, sid uint64, msg1 []byte) {
	hs, err := newResponder(c.id, from)
	if err != nil {
		return
	}
	payload, _, _, err := hs.ReadMessage(nil, msg1)
	if err != nil {
		return // message non déchiffrable : on ignore
	}
	if c.opt.OnServe != nil {
		c.opt.OnServe(from)
	}
	var req chunkReq
	resp := encodeResponse(stErr, []byte("requête invalide"))
	if json.Unmarshal(payload, &req) == nil {
		if data, err := c.opt.Store.Chunk(req.Root, req.Index); err == nil {
			resp = encodeResponse(stOK, data)
		} else {
			resp = encodeResponse(stErr, []byte("non hébergé"))
			if errors.Is(err, ErrCorrupt) {
				go c.dropCorrupt(req.Root)
			}
		}
	}
	c.upload.wait(len(resp))
	c.served.Add(int64(len(resp)))
	first := resp
	if len(first) > partSize {
		first = resp[:partSize]
	}
	msg2, _, cs2, err := hs.WriteMessage(nil, first)
	if err != nil {
		return
	}
	if c.conn.Send(&wire.Frame{T: wire.TFwd, H: wire.Header{To: from, SID: sid}, Data: msg2}) != nil {
		return
	}
	for off := len(first); off < len(resp); off += partSize {
		end := min(off+partSize, len(resp))
		ct, err := cs2.Encrypt(nil, nil, resp[off:end])
		if err != nil {
			return
		}
		if c.conn.Send(&wire.Frame{T: wire.TFwd, H: wire.Header{To: from, SID: sid}, Data: ct}) != nil {
			return
		}
	}
}

func (c *Client) onAudit(f *wire.Frame) {
	var hash []byte
	if c.opt.Store != nil {
		if data, err := c.opt.Store.Chunk(f.H.Root, f.H.Index); err == nil {
			hash = trust.AuditHash(f.H.Nonce, f.H.Index, data)
		}
	}
	_ = c.conn.Send(&wire.Frame{T: wire.TAuditResp, H: wire.Header{Q: f.H.Q, Hash: hash}})
}

// --- téléchargement d'un site --------------------------------------------------

// FetchStats : d'où viennent les morceaux.
type FetchStats struct {
	FromPeers, FromOrigin, BadChunks atomic.Int64
}

// Fetch télécharge le contenu root : bénévoles d'abord (en parallèle), origine en repli.
func (c *Client) Fetch(ctx context.Context, root string) ([]byte, *trust.Manifest, *FetchStats, error) {
	st := &FetchStats{}
	m, err := c.Manifest(ctx, root)
	if err != nil {
		return nil, nil, st, err
	}
	if c.opt.Store != nil && c.opt.Store.Manifest(root) != nil {
		data, err := c.opt.Store.Read(root)
		return data, m, st, err
	}
	hosts, _ := c.Find(ctx, root)
	if hl, err := c.Hosts(ctx); err == nil && hl != nil {
		kept := hosts[:0]
		for _, h := range hosts {
			if !hl.IsBanned(h.Peer) {
				kept = append(kept, h)
			}
		}
		hosts = kept
	}

	out := make([]byte, m.Size)
	idx := make(chan int)
	errs := make(chan error, len(m.Chunks))
	var wg sync.WaitGroup
	for w := 0; w < min(c.opt.Workers, len(m.Chunks)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				data, err := c.fetchChunk(ctx, m, i, hosts, st)
				if err != nil {
					errs <- fmt.Errorf("morceau %d: %w", i, err)
					continue
				}
				copy(out[i*m.ChunkSize:], data)
			}
		}()
	}
	for i := range m.Chunks {
		select {
		case idx <- i:
		case <-ctx.Done():
		}
	}
	close(idx)
	wg.Wait()
	close(errs)
	if err := <-errs; err != nil {
		return nil, nil, st, err
	}
	return out, m, st, ctx.Err()
}

func (c *Client) fetchChunk(ctx context.Context, m *trust.Manifest, i int, hosts []wire.HostInfo, st *FetchStats) ([]byte, error) {
	ranked := c.rep.rank(hosts)
	if n := len(ranked); n > 1 {
		// répartit la charge entre les 3 meilleurs
		k := i % min(n, 3)
		ranked = append(ranked[k:], ranked[:k]...)
	}
	for tries, h := range ranked {
		if tries >= 3 {
			break
		}
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		data, err := c.FetchFromPeer(cctx, h, m.Root, i)
		cancel()
		if err != nil {
			c.rep.Bad(h.Peer)
			continue
		}
		if !m.VerifyChunk(i, data) {
			// données modifiées : ban local immédiat + signalement (audit côté VPS)
			c.rep.Ban(h.Peer)
			c.Report(h.Peer, m.Root, i)
			st.BadChunks.Add(1)
			continue
		}
		c.rep.Good(h.Peer)
		st.FromPeers.Add(1)
		return data, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	data, err := c.OriginChunk(cctx, m.Root, i)
	if err != nil {
		return nil, err
	}
	if !m.VerifyChunk(i, data) {
		return nil, errors.New("morceau d'origine invalide")
	}
	st.FromOrigin.Add(1)
	return data, nil
}

// --- bénévolat -------------------------------------------------------------------

// Host : demande un ticket pour le site, en récupère une copie vérifiée et l'annonce.
func (c *Client) Host(ctx context.Context, site string) (*trust.Ticket, error) {
	if c.opt.Store == nil {
		return nil, errors.New("pas de stockage configuré")
	}
	r, err := c.request(ctx, &wire.Frame{T: wire.TVolunteer, H: wire.Header{Site: site}})
	if err != nil {
		return nil, err
	}
	t := r.H.Ticket
	if t == nil || t.Verify(c.opt.RootKey) != nil || t.Peer != c.ID() || !t.Valid(time.Now()) {
		return nil, errors.New("ticket invalide")
	}
	if c.opt.Store.Manifest(t.Root) == nil {
		data, m, _, err := c.Fetch(ctx, t.Root)
		if err != nil {
			return nil, err
		}
		if err := c.opt.Store.Put(m, data); err != nil {
			return nil, err
		}
	}
	c.mu.Lock()
	c.tickets[t.Root] = *t
	c.mu.Unlock()
	if err := c.Announce(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

// Announce (ré)annonce tous les tickets valides (après une reconnexion par exemple).
func (c *Client) Announce(ctx context.Context) error {
	c.mu.Lock()
	var ts []trust.Ticket
	now := time.Now()
	for root, t := range c.tickets {
		if t.Valid(now) {
			ts = append(ts, t)
		} else {
			delete(c.tickets, root)
		}
	}
	c.mu.Unlock()
	// l'annonce remplace l'ensemble annoncé : une liste vide retire tout
	r, err := c.request(ctx, &wire.Frame{T: wire.TAnnounce, H: wire.Header{Tickets: ts}})
	if err != nil {
		return err
	}
	if len(ts) > 0 && r.H.Index == 0 {
		return errors.New("aucun ticket accepté")
	}
	return nil
}

// Unhost : arrête d'héberger root (ticket abandonné, annonce mise à jour).
// La copie locale est à supprimer par l'appelant (ContentStore.Remove).
func (c *Client) Unhost(ctx context.Context, root string) error {
	c.mu.Lock()
	delete(c.tickets, root)
	c.mu.Unlock()
	return c.Announce(ctx)
}

// dropCorrupt : copie altérée sur le disque → supprimée et retirée de
// l'annuaire ; elle sera retéléchargée (vérifiée) au prochain alignement.
func (c *Client) dropCorrupt(root string) {
	c.opt.Store.Remove(root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = c.Unhost(ctx, root)
}

// Served : octets envoyés à d'autres pairs depuis la connexion.
func (c *Client) Served() int64 { return c.served.Load() }

// Tickets : copie des tickets détenus (pour les transférer à une nouvelle connexion).
func (c *Client) Tickets() []trust.Ticket {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []trust.Ticket
	for _, t := range c.tickets {
		out = append(out, t)
	}
	return out
}

// AdoptTickets reprend des tickets d'une connexion précédente.
func (c *Client) AdoptTickets(ts []trust.Ticket) {
	c.mu.Lock()
	for _, t := range ts {
		if t.Peer == c.ID() {
			c.tickets[t.Root] = t
		}
	}
	c.mu.Unlock()
}

// StoreDir : dossier de stockage du bénévole ("" s'il n'héberge pas).
func StoreDir(c *Client) string {
	if c.opt.Store == nil {
		return ""
	}
	return c.opt.Store.Dir
}
