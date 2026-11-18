// Package node : cœur du daemon client T2C Phantom.
//
// Il garde une connexion au relais (reconnexion automatique), tient le
// catalogue signé à jour, résout un alias .t2c en dossier local prêt à servir
// (téléchargé chez les bénévoles, vérifié morceau par morceau) et, si le
// bénévolat est activé, héberge une copie des sites visités.
package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"t2c/pkg/peerclient"
	"t2c/pkg/peerid"
	"t2c/pkg/trust"
	"t2c/pkg/wire"
)

// DefaultRelay : point d'entrée public (Cloudflare → VPS).
const DefaultRelay = "wss://t2cphantom.eu.org/v1/ws"

// DefaultRelays : relais essayés dans l'ordre. La liste signée (hosts.json,
// champ bootstrap) reçue à chaque connexion s'y ajoute : plusieurs relais
// peuvent se partager le réseau.
var DefaultRelays = []string{DefaultRelay, "wss://185-233-100-85.sslip.io/v1/ws"}

type Config struct {
	RelayURL  string // relais préféré (essayé en premier)
	DataDir   string
	MintBits  int
	RootKey   ed25519.PublicKey
	Volunteer bool     // compatibilité : avec HostSites, choix initial des sites hébergés
	HostSites []string // sites à héberger si aucun choix n'est encore enregistré
	Carnet    map[string]string
	// NoAnonymity : désactive les circuits en oignon (tests, dépannage).
	NoAnonymity bool
	// NoHiddenRelay : jamais de relais caché, quels que soient les réglages.
	NoHiddenRelay bool
	// HiddenRelayEvery : période de reconnexion du relais caché (tests).
	HiddenRelayEvery time.Duration
	// RelayMintBits : preuve de travail de la clé du relais caché (défaut broker.DefaultVolunteerBits).
	RelayMintBits int
}

type HostedSite struct {
	Site    string `json:"site"`
	Root    string `json:"root"`
	Expires int64  `json:"expires"`
}

type Status struct {
	Connected   bool         `json:"connected"`
	ID          string       `json:"id"`
	Relay       string       `json:"relay"`
	Volunteer   bool         `json:"volunteer"`
	Hosting     Hosting      `json:"hosting"`
	Anonymous   bool         `json:"anonymous"`    // demandes par circuit en oignon
	Hops        int          `json:"circuit_hops"` // relais du circuit
	Relays      int          `json:"relays"`       // relais connus dans la liste signée
	HiddenRelay bool         `json:"hidden_relay"` // relais caché bénévole actif
	RelayLinks  int          `json:"relay_links"`  // relais officiels auxquels il est relié
	StoreBytes  int64        `json:"store_bytes"`
	Served      int64        `json:"served_bytes"`
	Hosted      []HostedSite `json:"hosted"`
	CatalogVer  uint64       `json:"catalog_version"`
	LastError   string       `json:"last_error,omitempty"`
	FromPeers   int64        `json:"chunks_from_peers"`
	FromOrigin  int64        `json:"chunks_from_origin"`
	BadChunks   int64        `json:"bad_chunks_rejected"`
	Peers       []PeerInfo   `json:"peers"`
}

// PeerInfo : un bénévole rencontré (clé pseudonyme, jamais d'IP).
type PeerInfo struct {
	Peer        string `json:"peer"`
	Root        string `json:"root"`
	OriginScore int    `json:"origin_score"`
	LocalScore  int    `json:"local_score"`
	Banned      bool   `json:"banned"`
}

type Node struct {
	cfg   Config
	id    *peerid.Identity
	store *peerclient.ContentStore
	guard *trust.VersionGuard

	mu          sync.Mutex
	client      *peerclient.Client
	connected   chan struct{} // fermé quand une connexion est prête
	lastErr     string
	tickets     []trust.Ticket
	catalog     *trust.Catalog
	catalogAt   time.Time
	peers       map[string]PeerInfo
	fetching    map[string]*sync.Mutex // racine → verrou (pas deux téléchargements du même site)
	lazy        map[string]*lazySite   // racine → site lu à la demande
	hidden      *hiddenRelay           // relais caché bénévole (nil = arrêté)
	relayPolicy *trust.NetworkPolicy   // dernières consignes réseau vues par le relais caché
	relayID     *peerid.Identity       // identité du relais caché (distincte de id)
	relayIDMu   sync.Mutex
	stats       struct{ peers, origin, bad int64 }
	hosting     Hosting
	relay       string // relais actuellement utilisé
	recMu       sync.Mutex

	cancel context.CancelFunc
	done   chan struct{}
}

// Start charge l'identité et lance la boucle de connexion en arrière-plan.
func Start(ctx context.Context, cfg Config) (*Node, error) {
	if cfg.RelayURL == "" {
		cfg.RelayURL = DefaultRelay
	}
	if cfg.MintBits == 0 {
		cfg.MintBits = peerid.DefaultMintBits
	}
	if len(cfg.RootKey) == 0 {
		k, err := trust.RootKey()
		if err != nil {
			return nil, err
		}
		cfg.RootKey = k
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}
	id, err := peerid.LoadOrCreate(filepath.Join(cfg.DataDir, "identity.json"), cfg.MintBits)
	if err != nil {
		return nil, err
	}
	store, err := peerclient.NewContentStore(filepath.Join(cfg.DataDir, "hosted"))
	if err != nil {
		return nil, err
	}
	n := &Node{
		cfg: cfg, id: id, store: store,
		guard:     trust.NewVersionGuard(filepath.Join(cfg.DataDir, "versions.json")),
		connected: make(chan struct{}),
		peers:     map[string]PeerInfo{},
		fetching:  map[string]*sync.Mutex{},
		lazy:      map[string]*lazySite{},
		done:      make(chan struct{}),
	}
	n.loadTickets()
	n.hosting = n.loadHosting()
	n.loadStore()
	ctx, n.cancel = context.WithCancel(ctx)
	go n.run(ctx)
	go n.hiddenRelayLoop(ctx)
	return n, nil
}

func (n *Node) ID() string { return n.id.ID() }

// Stop ferme la connexion et attend la fin de la boucle.
func (n *Node) Stop() {
	n.cancel()
	n.mu.Lock()
	if n.client != nil {
		_ = n.client.Close()
	}
	n.mu.Unlock()
	<-n.done
}

func (n *Node) run(ctx context.Context) {
	defer close(n.done)
	backoff := time.Second
	attempt := 0
	for ctx.Err() == nil {
		relays := n.relayCandidates()
		url := relays[attempt%len(relays)]
		attempt++
		// anonymat automatique dès que la liste signée compte au moins deux relais
		keys := n.relayKeys()
		entry := entryKey(url, keys)
		anon := !n.cfg.NoAnonymity && peerclient.AnonymityPossible(entry, keys)
		dctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		c, err := peerclient.Dial(dctx, url, n.id, peerclient.Options{
			RootKey: n.cfg.RootKey, Store: n.store, Guard: n.guard,
			UploadBytesPerSec: n.hostingPrefs().MaxUploadKBs * 1024,
			Anonymous:         anon, EntryKey: entry, Relays: keys, AvoidRelay: n.hiddenRelayKey(),
			AnonMode: n.hostingPrefs().AnonMode,
		})
		cancel()
		if err != nil {
			n.setErr(fmt.Errorf("relais %s: %w", url, err))
			if attempt%len(relays) == 0 { // tous essayés : on patiente
				sleep(ctx, backoff)
				backoff = min(backoff*2, time.Minute)
			}
			continue
		}
		backoff, attempt = time.Second, 0
		n.mu.Lock()
		n.relay = url
		n.client = c
		n.lastErr = ""
		c.AdoptTickets(n.tickets)
		close(n.connected)
		n.mu.Unlock()

		n.onConnect(ctx, c)

		select {
		case <-ctx.Done():
		case <-c.Done():
		}
		n.mu.Lock()
		n.tickets = c.Tickets()
		n.client = nil
		n.connected = make(chan struct{})
		n.mu.Unlock()
		_ = c.Close()
		if ctx.Err() == nil {
			n.setErr(errors.New("connexion au relais perdue, reconnexion…"))
			sleep(ctx, 2*time.Second)
		}
	}
}

func (n *Node) onConnect(ctx context.Context, c *peerclient.Client) {
	if _, err := n.refreshCatalog(ctx, c); err != nil {
		n.setErr(err)
	}
	go n.learnRelaysRetry(ctx, c)
	actx, cancel := context.WithTimeout(ctx, 30*time.Second)
	_ = c.Announce(actx)
	cancel()
	go n.hostingLoop(ctx, c)
}

// waitClient renvoie la connexion courante (attend jusqu'à ctx).
func (n *Node) waitClient(ctx context.Context) (*peerclient.Client, error) {
	for {
		n.mu.Lock()
		c, ready := n.client, n.connected
		n.mu.Unlock()
		if c != nil {
			return c, nil
		}
		select {
		case <-ready:
		case <-ctx.Done():
			return nil, fmt.Errorf("relais injoignable: %s", n.lastError())
		}
	}
}

func (n *Node) setErr(err error) {
	n.mu.Lock()
	n.lastErr = err.Error()
	n.mu.Unlock()
}

func (n *Node) lastError() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastErr == "" {
		return "connexion en cours"
	}
	return n.lastErr
}

// --- catalogue -------------------------------------------------------------------

const catalogTTL = time.Minute

func (n *Node) refreshCatalog(ctx context.Context, c *peerclient.Client) (*trust.Catalog, error) {
	cat, err := c.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	n.mu.Lock()
	n.catalog, n.catalogAt = cat, time.Now()
	n.mu.Unlock()
	return cat, nil
}

// Catalog : catalogue signé (mis en cache 1 min).
func (n *Node) Catalog(ctx context.Context) (*trust.Catalog, error) {
	n.mu.Lock()
	cat, at := n.catalog, n.catalogAt
	n.mu.Unlock()
	if cat != nil && time.Since(at) < catalogTTL {
		return cat, nil
	}
	c, err := n.waitClient(ctx)
	if err != nil {
		if cat != nil {
			return cat, nil
		}
		return nil, err
	}
	fresh, err := n.refreshCatalog(ctx, c)
	if err != nil && cat != nil {
		return cat, nil
	}
	return fresh, err
}

// CachedCatalog : dernier catalogue connu, sans réseau (nil si aucun).
func (n *Node) CachedCatalog() *trust.Catalog {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.catalog
}

// Resolve : alias .t2c (ou entrée du carnet local) → entrée du catalogue signé.
func (n *Node) Resolve(ctx context.Context, domain string) (*trust.SiteEntry, error) {
	alias := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".t2c")
	cat, err := n.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	// le carnet local ne s'applique que s'il pointe vers un site du catalogue
	// (un ancien carnet libp2p ne doit pas masquer l'alias officiel)
	for _, k := range []string{alias + ".t2c", alias} {
		if mapped, ok := n.cfg.Carnet[k]; ok {
			if e, ok := cat.Find(strings.TrimSuffix(strings.ToLower(mapped), ".t2c")); ok {
				return e, nil
			}
		}
	}
	e, ok := cat.Find(alias)
	if !ok {
		return nil, fmt.Errorf("%s.t2c n'est pas dans le catalogue signé", alias)
	}
	return e, nil
}

// --- sites -------------------------------------------------------------------------

func (n *Node) siteDir(root string) string {
	return filepath.Join(n.cfg.DataDir, "sites", root)
}

// SiteDir renvoie le dossier local du site (le télécharge et le vérifie au besoin).
// Un contenu est adressé par sa racine : une nouvelle version = un nouveau dossier.
func (n *Node) SiteDir(ctx context.Context, domain string) (string, *trust.SiteEntry, error) {
	e, err := n.Resolve(ctx, domain)
	if err != nil {
		return "", nil, err
	}
	dir := n.siteDir(e.Root)
	ready := filepath.Join(dir, ".t2c-ok")
	if _, err := os.Stat(ready); err == nil {
		return dir, e, nil
	}

	n.mu.Lock()
	lk := n.fetching[e.Root]
	if lk == nil {
		lk = &sync.Mutex{}
		n.fetching[e.Root] = lk
	}
	n.mu.Unlock()
	lk.Lock()
	defer lk.Unlock()
	if _, err := os.Stat(ready); err == nil {
		return dir, e, nil
	}

	c, err := n.waitClient(ctx)
	if err != nil {
		return "", nil, err
	}
	// copie déjà hébergée (et revérifiée à la lecture) : rien à télécharger
	data, _ := n.store.Read(e.Root)
	if data == nil && n.hostingPrefs().wants(e.ID) {
		if _, err := c.Host(ctx, e.ID); err != nil {
			n.setErr(fmt.Errorf("bénévolat %s: %w", e.ID, err))
		} else {
			n.saveTickets(c.Tickets())
			data, _ = n.store.Read(e.Root)
		}
	}
	if data == nil {
		blob, _, st, err := c.Fetch(ctx, e.Root)
		n.addStats(st)
		if err != nil {
			return "", nil, err
		}
		data = blob
	}
	n.notePeers(ctx, c, e.Root)

	tmp := dir + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := Unzip(data, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return "", nil, err
	}
	if err := os.WriteFile(filepath.Join(tmp, ".t2c-ok"), []byte(e.Root+"\n"), 0o644); err != nil {
		return "", nil, err
	}
	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		return "", nil, err
	}
	return dir, e, nil
}

func (n *Node) addStats(st *peerclient.FetchStats) {
	if st == nil {
		return
	}
	n.mu.Lock()
	n.stats.peers += st.FromPeers.Load()
	n.stats.origin += st.FromOrigin.Load()
	n.stats.bad += st.BadChunks.Load()
	n.mu.Unlock()
}

func (n *Node) notePeers(ctx context.Context, c *peerclient.Client, root string) {
	fctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	hosts, err := c.Find(fctx, root)
	if err != nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, h := range hosts {
		n.peers[h.Peer] = peerInfo(c, h, root)
	}
}

func peerInfo(c *peerclient.Client, h wire.HostInfo, root string) PeerInfo {
	return PeerInfo{Peer: h.Peer, Root: root, OriginScore: h.Score, LocalScore: c.Reputation().Score(h.Peer), Banned: c.Reputation().Banned(h.Peer)}
}

// --- état ----------------------------------------------------------------------------

func (n *Node) Status() Status {
	links := n.hiddenRelayLinks() // prend n.mu : avant le verrou ci-dessous
	n.mu.Lock()
	defer n.mu.Unlock()
	st := Status{
		Connected: n.client != nil, ID: n.id.ID(), Relay: n.relay, Volunteer: n.hosting.active(),
		Hosting: n.hosting, StoreBytes: n.store.Size(),
		LastError: n.lastErr, FromPeers: n.stats.peers, FromOrigin: n.stats.origin, BadChunks: n.stats.bad,
	}
	if n.client != nil {
		st.Served = n.client.Served()
		st.Anonymous = n.client.Anonymous()
		st.Hops = n.client.CircuitHops()
	}
	st.Relays = len(n.relayKeys())
	st.RelayLinks = links
	st.HiddenRelay = n.hidden != nil
	if n.catalog != nil {
		st.CatalogVer = n.catalog.Version
	}
	tickets := n.tickets
	if n.client != nil {
		tickets = n.client.Tickets()
	}
	now := time.Now()
	for _, t := range tickets {
		if t.Valid(now) {
			st.Hosted = append(st.Hosted, HostedSite{Site: t.Site, Root: t.Root, Expires: t.Expires})
		}
	}
	sort.Slice(st.Hosted, func(i, j int) bool { return st.Hosted[i].Site < st.Hosted[j].Site })
	for _, p := range n.peers {
		if n.client != nil {
			p.LocalScore = n.client.Reputation().Score(p.Peer)
			p.Banned = n.client.Reputation().Banned(p.Peer)
		}
		st.Peers = append(st.Peers, p)
	}
	sort.Slice(st.Peers, func(i, j int) bool { return st.Peers[i].Peer < st.Peers[j].Peer })
	return st
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
