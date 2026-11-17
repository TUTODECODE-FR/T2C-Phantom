// Package api : tableau de bord local (127.0.0.1:9090) du client T2C Phantom.
package api

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"t2c/pkg/node"
	"t2c/pkg/proxy"
	"t2c/pkg/proxy/assohttps"
	"t2c/pkg/sysint"
	"t2c/pkg/utils"
)

const gatewayHost = "t2cphantom.eu.org"

// proxyPort : proxy .t2c local (T2C_PROXY_PORT pour en changer, défaut 8080).
var proxyPort = func() int {
	if p, err := strconv.Atoi(os.Getenv("T2C_PROXY_PORT")); err == nil && p > 0 && p < 65536 {
		return p
	}
	return 8080
}()

type Dashboard struct {
	mu        sync.Mutex
	node      *node.Node
	proxyOn   bool
	volunteer bool
}

func NewDashboard() *Dashboard { return &Dashboard{} }

// Assets : interface web intégrée au binaire (aucun fichier externe, aucune ressource distante).
func Assets() fs.FS {
	sub, _ := fs.Sub(webFS, "web")
	return sub
}

// APIHandler : routes /api/* (réutilisées par l'application de bureau).
func (d *Dashboard) APIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", d.handleStatus)
	mux.HandleFunc("/api/config", d.handleConfig)
	mux.HandleFunc("/api/client/start", d.handleStartClient)
	mux.HandleFunc("/api/stop", d.handleStop)
	mux.HandleFunc("/api/proxy-test", d.handleProxyTest)
	mux.HandleFunc("/api/network", d.handleNetwork)
	mux.HandleFunc("/api/cert-validate", d.handleCertValidate)
	mux.HandleFunc("/api/asso-ca.crt", d.handleAssoCAFile)
	mux.HandleFunc("/api/asso-sites", d.handleAssoSites)
	mux.HandleFunc("/api/setup", d.handleSetup)
	mux.HandleFunc("/api/onboarding", d.handleOnboarding)
	mux.HandleFunc("/api/setup/ca", d.handleSetupCA)
	mux.HandleFunc("/api/setup/proxy", d.handleSetupProxy)
	mux.HandleFunc("/api/browsers", d.handleBrowsers)
	mux.HandleFunc("/api/open", d.handleOpen)
	mux.HandleFunc("/api/sites", d.handleSites)
	mux.HandleFunc("/api/hosting", d.handleHosting)
	return mux
}

// Shutdown arrête la connexion au relais (fermeture de l'application).
func (d *Dashboard) Shutdown() {
	d.mu.Lock()
	n := d.node
	d.node = nil
	d.mu.Unlock()
	if n != nil {
		n.Stop()
	}
	// le proxy local s'arrête avec l'app : on rend au système ses réglages d'origine
	_ = sysint.Restore(utils.GetT2CDataDir())
}

// Start : tableau de bord dans le navigateur (mode sans fenêtre), http://127.0.0.1:port.
func (d *Dashboard) Start(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/api/", d.APIHandler())
	mux.Handle("/", http.FileServerFS(Assets()))

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("\n=== T2C Dashboard ===\nInterface : http://%s\n=====================\n\n", addr)

	// anti DNS-rebinding : seul l'hôte local exact est accepté.
	// anti CSRF : une page web ouverte dans le navigateur ne doit pas pouvoir
	// piloter l'app (POST/DELETE depuis une autre origine refusés).
	local := map[string]bool{"http://" + addr: true, fmt.Sprintf("http://localhost:%d", port): true}
	guard := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != addr && r.Host != fmt.Sprintf("localhost:%d", port) {
				http.Error(w, "Forbidden: Invalid Host", http.StatusForbidden)
				return
			}
			if !crossSiteSafe(r, local) {
				http.Error(w, "Forbidden: cross-site", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	srv := &http.Server{
		Addr: addr, Handler: guard(mux),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 30 * time.Second,
	}
	return srv.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func relayURL() string {
	if u := os.Getenv("T2C_RELAY_URL"); u != "" {
		return u
	}
	return node.DefaultRelay
}

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	_, err := net.DefaultResolver.LookupHost(ctx, gatewayHost)
	writeJSON(w, map[string]any{
		"gatewayHost":      gatewayHost,
		"gatewayIP":        "relais",
		"gatewayMultiaddr": relayURL(), // champ « adresse du relais » de l'interface
		"cloudflareReady":  err == nil,
		"proxyPort":        proxyPort,
		"tld":              ".t2c",
		"tldCourses":       ".tdc",
	})
}

func (d *Dashboard) status() (node.Status, bool) {
	d.mu.Lock()
	n := d.node
	d.mu.Unlock()
	if n == nil {
		return node.Status{}, false
	}
	return n.Status(), true
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	st, running := d.status()
	out := map[string]any{"status": "stopped", "address": "", "proxyPort": 0, "peers": 0, "seeding": false, "hopRelay": false}
	if running {
		out = map[string]any{
			"status":       "running",
			"connected":    st.Connected,
			"address":      st.ID,
			"proxyPort":    proxyPort,
			"relayAddr":    st.Relay,
			"peers":        len(st.Peers),
			"seeding":      st.Volunteer,
			"hopRelay":     false, // pas de relais entre pairs : tout passe chiffré par le VPS
			"hosted":       st.Hosted,
			"lastError":    st.LastError,
			"catalogVer":   st.CatalogVer,
			"chunksPeers":  st.FromPeers,
			"chunksOrigin": st.FromOrigin,
			"badChunks":    st.BadChunks,
			"anonymous":    st.Anonymous,
			"circuitHops":  st.Hops,
			"relays":       st.Relays,
		}
	}
	writeJSON(w, out)
}

func (d *Dashboard) handleStartClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RelayAddr      string `json:"relay_addr"`
		DisableSeeding bool   `json:"disable_seeding"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req)

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.node != nil {
		http.Error(w, "déjà connecté", http.StatusBadRequest)
		return
	}
	relay := relayURL()
	if strings.HasPrefix(req.RelayAddr, "wss://") || strings.HasPrefix(req.RelayAddr, "ws://127.0.0.1") || strings.HasPrefix(req.RelayAddr, "ws://localhost") {
		relay = req.RelayAddr
	}
	n, err := node.Start(context.Background(), node.Config{
		RelayURL: relay,
		DataDir:  utils.GetPath("peer"),
		Carnet:   LoadCarnet(utils.GetPath("carnet.json")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.node = n
	d.volunteer = n.Status().Volunteer
	d.ensureProxyLocked()
	// plug-and-play : si l'utilisateur a déjà accepté la configuration, on remet le PAC
	if sysint.Enabled(utils.GetT2CDataDir()) && os.Getenv("T2C_NO_BROWSER_SETUP") == "" {
		go sysint.Reapply(utils.GetT2CDataDir(), fmt.Sprintf("127.0.0.1:%d", proxyPort))
	}
	writeJSON(w, map[string]any{"success": true, "address": n.ID(), "seeding": d.volunteer, "role": "pair"})
}

// ensureProxy démarre le proxy .t2c local (il sert aussi le fichier PAC).
func (d *Dashboard) ensureProxy() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureProxyLocked()
}

func (d *Dashboard) ensureProxyLocked() {
	if d.proxyOn {
		return
	}
	d.proxyOn = true
	go func() {
		if err := proxy.StartProxyServer(&nodeRef{d: d}, fmt.Sprintf("127.0.0.1:%d", proxyPort)); err != nil {
			fmt.Println("proxy:", err)
			d.mu.Lock()
			d.proxyOn = false
			d.mu.Unlock()
		}
	}()
}

func (d *Dashboard) handleStop(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	n := d.node
	d.node = nil
	d.mu.Unlock()
	if n != nil {
		n.Stop()
	}
	writeJSON(w, map[string]any{"success": true})
}

// handleProxyTest teste une URL .t2c via le proxy local.
func (d *Dashboard) handleProxyTest(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"ok": false, "error": "url manquante"})
		return
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"ok": false, "error": "url invalide"})
		return
	}
	host := strings.ToLower(strings.Split(u.Host, ":")[0])
	if !strings.HasSuffix(host, ".t2c") {
		w.WriteHeader(http.StatusBadRequest)
		msg := "domaine doit finir par .t2c"
		if strings.HasSuffix(host, ".tdc") {
			msg = ".tdc = cours TUTODECODE, pas un site réseau — utilise .t2c"
		}
		writeJSON(w, map[string]any{"ok": false, "error": msg, "host": host})
		return
	}
	d.mu.Lock()
	ready := d.node != nil && d.proxyOn
	d.mu.Unlock()
	if !ready {
		writeJSON(w, map[string]any{"ok": false, "proxyReady": false, "url": raw,
			"error": "proxy local inactif — lance d'abord la connexion", "hint": "Firefox → proxy 127.0.0.1:8080 puis réessaie"})
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(&url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", proxyPort)}),
			TLSClientConfig: d.localCATLS(),
		},
		Timeout:       2 * time.Minute, // premier accès : téléchargement du site
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	start := time.Now()
	resp, err := client.Get(raw)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "proxyReady": true, "url": raw, "host": host, "error": err.Error(), "latencyMs": latency})
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	writeJSON(w, map[string]any{
		"ok": resp.StatusCode < 400, "proxyReady": true, "url": raw, "host": host,
		"status": resp.StatusCode, "latencyMs": latency, "seenVia": fmt.Sprintf("127.0.0.1:%d", proxyPort),
		"root": resp.Header.Get("X-T2C-Root"),
	})
}

// localCATLS : vérifie les réponses HTTPS du proxy avec la CA locale (pas d'InsecureSkipVerify).
func (d *Dashboard) localCATLS() *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if p, err := os.ReadFile(proxy.LocalCAPEMPath()); err == nil {
		pool, _ := x509Pool(p)
		cfg.RootCAs = pool
	}
	return cfg
}

func (d *Dashboard) handleAssoCAFile(w http.ResponseWriter, r *http.Request) {
	p, err := os.ReadFile(proxy.LocalCAPEMPath())
	if err != nil || len(p) == 0 {
		http.Error(w, "CA locale absente — lance la connexion d'abord", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="tutodecode-ca.crt"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(p)
}

func localCAFingerprint() (string, string) {
	p, err := os.ReadFile(proxy.LocalCAPEMPath())
	if err != nil {
		return "", ""
	}
	blk, _ := pem.Decode(p)
	if blk == nil {
		return "", string(p)
	}
	sum := sha256.Sum256(blk.Bytes)
	return hex.EncodeToString(sum[:]), string(p)
}

// catalogSites : sites du catalogue signé (tous authentiques, donc tous en HTTPS).
func (d *Dashboard) catalogSites(ctx context.Context) []map[string]any {
	d.mu.Lock()
	n := d.node
	d.mu.Unlock()
	out := []map[string]any{}
	if n == nil {
		return out
	}
	cat, err := n.Catalog(ctx)
	if err != nil {
		cat = n.CachedCatalog()
	}
	if cat == nil {
		return out
	}
	for _, s := range cat.Sites {
		out = append(out, map[string]any{
			"alias": s.Alias + ".t2c", "id": s.ID, "title": s.Title, "root": s.Root, "size": s.Size,
			"asso": true, "https": true, "source": fmt.Sprintf("catalogue signé v%d", cat.Version),
		})
	}
	return out
}

func (d *Dashboard) handleAssoSites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var sites []string
	for _, s := range d.catalogSites(ctx) {
		sites = append(sites, s["alias"].(string))
	}
	fp, _ := localCAFingerprint()
	var installedIn []string
	if ca, err := assohttps.LoadOrCreate(); err == nil {
		installedIn = sysint.Status(ca.CertPEMPath(), assohttps.Nick(), "").CA
	}
	writeJSON(w, map[string]any{
		"sites": sites, "count": len(sites),
		"caDownload": "/api/asso-ca.crt", "caFilename": "tutodecode-ca.crt", "caNick": assohttps.Nick(),
		"serverCAFingerprint": fp, "serverCADisplay": fp != "",
		"caInstalled":   len(installedIn) > 0,
		"caInstalledIn": installedIn,
		"canAutoTrust":  os.Getenv("T2C_NO_BROWSER_SETUP") == "",
		"setupEnabled":  sysint.Enabled(utils.GetT2CDataDir()),
		"note":          "Une seule CA locale couvre tous les sites du catalogue signé",
	})
}

// handleSetup : configuration automatique (CA + proxy) pour tous les navigateurs.
// GET = état ; POST = installer ; DELETE = désactiver et restaurer.
func (d *Dashboard) handleSetup(w http.ResponseWriter, r *http.Request) {
	ca, err := assohttps.LoadOrCreate()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	dataDir, addr := utils.GetT2CDataDir(), fmt.Sprintf("127.0.0.1:%d", proxyPort)
	switch r.Method {
	case http.MethodGet:
		st := sysint.Status(ca.CertPEMPath(), assohttps.Nick(), addr)
		writeJSON(w, map[string]any{"enabled": sysint.Enabled(dataDir), "status": st})
	case http.MethodPost:
		if os.Getenv("T2C_NO_BROWSER_SETUP") != "" {
			writeJSON(w, map[string]any{"ok": false, "error": "configuration désactivée (T2C_NO_BROWSER_SETUP)"})
			return
		}
		rep := sysint.Setup(dataDir, ca.CertPEMPath(), assohttps.Nick(), addr)
		writeJSON(w, map[string]any{"ok": len(rep.CA) > 0, "report": rep})
	case http.MethodDelete:
		err := sysint.Disable(dataDir)
		writeJSON(w, map[string]any{"ok": err == nil})
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// handleCertValidate : compare le certificat importé dans le navigateur avec la CA locale,
// et (option) vérifie qu'un site du catalogue répond bien en HTTPS via le proxy.
func (d *Dashboard) handleCertValidate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PEM         string `json:"pem"`
		Fingerprint string `json:"fingerprint"`
		Alias       string `json:"alias"`
		CheckHTTPS  bool   `json:"checkHttps"`
	}
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body)
	} else {
		body.Alias = r.URL.Query().Get("alias")
		body.Fingerprint = r.URL.Query().Get("fingerprint")
		body.CheckHTTPS = r.URL.Query().Get("https") == "1"
	}
	localFP, localPEM := localCAFingerprint()
	out := map[string]any{"officialFingerprint": localFP, "officialPem": localPEM, "officialDisplay": localFP != ""}
	submitted := strings.ToLower(strings.NewReplacer(":", "", " ", "").Replace(body.Fingerprint))
	if submitted == "" && body.PEM != "" {
		if blk, _ := pem.Decode([]byte(body.PEM)); blk != nil {
			sum := sha256.Sum256(blk.Bytes)
			submitted = hex.EncodeToString(sum[:])
		}
	}
	if submitted == "" {
		out["ok"], out["awaitingCert"] = false, true
		out["message"] = "Envoie le certificat (PEM ou empreinte SHA-256) importé dans le navigateur"
		writeJSON(w, out)
		return
	}
	out["submittedFingerprint"] = submitted
	out["certMatch"] = submitted == localFP
	if submitted != localFP {
		out["ok"], out["error"] = false, "ce certificat n'est pas la CA locale de T2C Phantom"
		writeJSON(w, out)
		return
	}
	ok := true
	if body.CheckHTTPS {
		alias := strings.TrimSuffix(strings.ToLower(body.Alias), ".t2c")
		if alias == "" {
			alias = "tutodecode"
		}
		client := &http.Client{Timeout: 2 * time.Minute, Transport: &http.Transport{
			Proxy:           http.ProxyURL(&url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", proxyPort)}),
			TLSClientConfig: d.localCATLS(),
		}}
		resp, err := client.Get("https://" + alias + ".t2c/")
		if err != nil {
			ok = false
			out["error"] = "HTTPS injoignable: " + err.Error()
		} else {
			resp.Body.Close()
			ok = resp.StatusCode < 400
			out["httpsStatus"] = resp.StatusCode
		}
		out["httpsOk"] = ok
	}
	out["ok"], out["validated"] = ok, ok
	if ok {
		out["message"] = "CA locale reconnue · HTTPS .t2c opérationnel"
	}
	writeJSON(w, out)
}

func (d *Dashboard) handleNetwork(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	carnet := LoadCarnet(utils.GetPath("carnet.json"))
	local := []map[string]any{}
	for alias, target := range carnet {
		local = append(local, map[string]any{"alias": alias, "target": target, "source": "carnet-local", "status": "listed"})
	}
	st, running := d.status()
	hosted := []map[string]any{}
	for _, h := range st.Hosted {
		hosted = append(hosted, map[string]any{"alias": h.Site + ".t2c", "root": h.Root, "status": "bénévole", "expires": h.Expires})
	}
	seeders := []map[string]any{}
	for _, p := range st.Peers {
		role := fmt.Sprintf("bénévole · score origine %d · local %d", p.OriginScore, p.LocalScore)
		if p.Banned {
			role = "banni (morceau faux)"
		}
		seeders = append(seeders, map[string]any{"peerID": p.Peer, "role": role, "via": "relais (chiffré de bout en bout)"})
	}
	fp, caPEM := localCAFingerprint()
	writeJSON(w, map[string]any{
		"running": running, "connected": st.Connected, "clientID": st.ID, "peers": len(st.Peers),
		"seeding": st.Volunteer, "hopRelay": false,
		"sites": local, "hostedSites": hosted, "remoteCatalog": d.catalogSites(ctx), "seeders": seeders,
		"polledAt": time.Now().UTC().Format(time.RFC3339),
		"note":     "Identité = clé aléatoire. Seul Cloudflare voit ton IP ; le relais, l'origine et les autres pairs ne voient que ta clé.",
		"assoCA": map[string]any{
			"display": fp != "", "fingerprintSha256": fp, "pem": caPEM,
			"note": "CA générée sur ta machine ; elle ne signe que les sites du catalogue signé",
		},
		"abuse": proxy.GlobalAbuse.Snapshot(),
	})
}

// LoadCarnet : alias locaux (alias.t2c → id ou alias du catalogue).
func LoadCarnet(path string) map[string]string {
	m := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// crossSiteSafe : lectures toujours permises ; toute action (POST, DELETE…)
// doit venir de l'interface elle-même, jamais d'un autre site.
func crossSiteSafe(r *http.Request, local map[string]bool) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if s := r.Header.Get("Sec-Fetch-Site"); s == "cross-site" || s == "same-site" {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" && !local[o] {
		return false
	}
	return true
}
