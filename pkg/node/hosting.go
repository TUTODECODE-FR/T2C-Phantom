package node

import (
	"context"
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"t2c/pkg/peerclient"
	"t2c/pkg/trust"
)

// Hosting : choix du bénévole, enregistré dans <data>/hosting.json.
// Personne n'héberge sans l'avoir choisi.
type Hosting struct {
	All          bool     `json:"all"`            // tous les sites du catalogue
	Sites        []string `json:"sites"`          // sinon : ces sites (id)
	MaxDiskMB    int64    `json:"max_disk_mb"`    // place maximale des copies
	MaxUploadKBs int64    `json:"max_upload_kbs"` // débit maximal envoyé aux autres pairs
}

const (
	defaultDiskMB    = 2048
	defaultUploadKBs = 1024
	hostingEvery     = 3 * time.Minute
	ticketMargin     = 2 * time.Hour // renouveler un ticket avant son expiration
)

func (h Hosting) active() bool         { return h.All || len(h.Sites) > 0 }
func (h Hosting) wants(id string) bool { return h.All || slices.Contains(h.Sites, id) }

func (h *Hosting) normalize() {
	if h.MaxDiskMB <= 0 {
		h.MaxDiskMB = defaultDiskMB
	}
	if h.MaxUploadKBs <= 0 {
		h.MaxUploadKBs = defaultUploadKBs
	}
	seen := map[string]bool{}
	var sites []string
	for _, s := range h.Sites {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			sites = append(sites, s)
		}
	}
	sort.Strings(sites)
	h.Sites = sites
}

func (n *Node) hostingPath() string { return filepath.Join(n.cfg.DataDir, "hosting.json") }

func (n *Node) loadHosting() Hosting {
	var h Hosting
	if b, err := os.ReadFile(n.hostingPath()); err == nil && json.Unmarshal(b, &h) == nil {
		h.normalize()
		return h
	}
	if n.cfg.Volunteer {
		h.Sites = n.cfg.HostSites
	}
	h.normalize()
	return h
}

func (n *Node) hostingPrefs() Hosting {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.hosting
}

// SetHosting enregistre le choix du bénévole et l'applique tout de suite.
func (n *Node) SetHosting(h Hosting) error {
	h.normalize()
	b, _ := json.MarshalIndent(h, "", "  ")
	if err := os.WriteFile(n.hostingPath(), b, 0o600); err != nil {
		return err
	}
	n.mu.Lock()
	n.hosting = h
	c := n.client
	n.mu.Unlock()
	if c != nil {
		go n.reconcile(context.Background(), c)
	}
	return nil
}

// loadStore recharge les copies déjà présentes (signature revérifiée).
func (n *Node) loadStore() {
	matches, _ := filepath.Glob(filepath.Join(n.store.Dir, "*.json"))
	for _, m := range matches {
		root := strings.TrimSuffix(filepath.Base(m), ".json")
		if _, err := n.store.Load(root, n.cfg.RootKey); err != nil {
			n.store.Remove(root)
		}
	}
}

// hostingLoop : applique le choix à la connexion puis régulièrement
// (nouvelle version d'un site, ticket à renouveler, choix modifié).
func (n *Node) hostingLoop(ctx context.Context, c *peerclient.Client) {
	t := time.NewTicker(hostingEvery)
	defer t.Stop()
	for {
		n.reconcile(ctx, c)
		select {
		case <-ctx.Done():
			return
		case <-c.Done():
			return
		case <-t.C:
			if _, err := n.refreshCatalog(ctx, c); err != nil {
				n.setErr(err)
			}
		}
	}
}

// reconcile aligne les copies hébergées sur le choix du bénévole et le
// catalogue signé : télécharge ce qui manque (dans la limite de place),
// renouvelle les tickets, et supprime les copies périmées ou non voulues.
func (n *Node) reconcile(ctx context.Context, c *peerclient.Client) {
	n.recMu.Lock()
	defer n.recMu.Unlock()
	cat := n.CachedCatalog()
	if cat == nil {
		return
	}
	h := n.hostingPrefs()
	want := map[string]trust.SiteEntry{}
	for _, e := range cat.Sites {
		if h.wants(e.ID) {
			want[e.Root] = e
		}
	}
	changed := false
	for _, root := range n.store.Roots() {
		if _, ok := want[root]; !ok {
			uctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_ = c.Unhost(uctx, root)
			cancel()
			n.store.Remove(root)
			changed = true
		}
	}
	valid := map[string]bool{}
	soon := time.Now().Add(ticketMargin)
	for _, t := range c.Tickets() {
		if t.Valid(soon) {
			valid[t.Root] = true
		}
	}
	entries := make([]trust.SiteEntry, 0, len(want))
	for _, e := range want {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	for _, e := range entries {
		if valid[e.Root] && n.store.Manifest(e.Root) != nil {
			continue
		}
		if n.store.Manifest(e.Root) == nil && n.store.Size()+e.Size > h.MaxDiskMB<<20 {
			n.setErr(fmt.Errorf("bénévolat %s : place insuffisante (limite %d Mo)", e.ID, h.MaxDiskMB))
			continue
		}
		hctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		_, err := c.Host(hctx, e.ID)
		cancel()
		if err != nil {
			n.setErr(fmt.Errorf("bénévolat %s: %w", e.ID, err))
			continue
		}
		changed = true
	}
	if changed {
		n.saveTickets(c.Tickets())
	}
}

// --- relais ---------------------------------------------------------------

func (n *Node) relaysPath() string { return filepath.Join(n.cfg.DataDir, "relays.json") }

// relayCandidates : relais choisi par l'utilisateur, sinon les relais de la
// liste signée dans un ordre aléatoire (les utilisateurs se répartissent entre
// les relais de la fédération), puis les relais par défaut.
func (n *Node) relayCandidates() []string {
	var learned []string
	if b, err := os.ReadFile(n.relaysPath()); err == nil {
		_ = json.Unmarshal(b, &learned)
	}
	mrand.Shuffle(len(learned), func(i, j int) { learned[i], learned[j] = learned[j], learned[i] })
	first := []string{}
	if n.cfg.RelayURL != "" && n.cfg.RelayURL != DefaultRelay {
		first = append(first, n.cfg.RelayURL)
	}
	var out []string
	for _, u := range append(append(first, learned...), DefaultRelays...) {
		u, _, _ = strings.Cut(u, "#")
		if validRelay(u) && !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	return out
}

// relayKeys : relais de la liste signée avec leur clé (pour bâtir des circuits).
func (n *Node) relayKeys() []peerclient.RelayKey {
	var learned []string
	if b, err := os.ReadFile(n.relaysPath()); err == nil {
		_ = json.Unmarshal(b, &learned)
	}
	var out []peerclient.RelayKey
	for _, e := range learned {
		u, frag, ok := strings.Cut(e, "#")
		if !ok || !strings.HasPrefix(frag, "k=") {
			continue
		}
		k, err := trust.ParsePublicKey(strings.TrimPrefix(frag, "k="))
		if err != nil {
			continue
		}
		out = append(out, peerclient.RelayKey{URL: u, Key: k})
	}
	return out
}

// entryKey : clé du relais d'URL url, si la liste signée la donne.
func entryKey(url string, relays []peerclient.RelayKey) []byte {
	for _, r := range relays {
		if r.URL == url {
			return r.Key
		}
	}
	return nil
}

func validRelay(u string) bool {
	return strings.HasPrefix(u, "wss://") || strings.HasPrefix(u, "ws://127.0.0.1") || strings.HasPrefix(u, "ws://localhost")
}

// learnRelaysRetry : la liste peut être momentanément indisponible (relais qui
// rejoint juste son amont) : on réessaie quelques fois.
func (n *Node) learnRelaysRetry(ctx context.Context, c *peerclient.Client) {
	for i := 0; i < 10; i++ {
		if n.learnRelays(ctx, c) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-c.Done():
			return
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}
}

// learnRelays retient les relais publiés dans la liste signée hosts.json.
func (n *Node) learnRelays(ctx context.Context, c *peerclient.Client) bool {
	hctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	hl, err := c.Hosts(hctx)
	if err != nil {
		return false
	}
	if len(hl.Bootstrap) == 0 {
		return true
	}
	var urls []string
	for _, u := range hl.Bootstrap { // « url#k=clé » : la clé sert aux circuits anonymes
		base, _, _ := strings.Cut(u, "#")
		// relais caché (« hidden:nom#k=clé ») : jamais contacté, seulement utilisé dans les circuits
		usable := validRelay(base) || (strings.HasPrefix(base, "hidden:") && strings.Contains(u, "#k="))
		if usable && len(urls) < 16 && !slices.Contains(urls, u) {
			urls = append(urls, u)
		}
	}
	b, _ := json.Marshal(urls)
	_ = os.WriteFile(n.relaysPath(), b, 0o600)
	return true
}
