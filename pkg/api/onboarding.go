package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"t2c/pkg/node"
	"t2c/pkg/proxy/assohttps"
	"t2c/pkg/sysint"
	"t2c/pkg/utils"
)

// Assistant du premier lancement : 1 certificat, 2 navigateur, 3 proxy, 4 réseau.
// Ensuite l'app n'affiche plus que le carnet de sites (et le bénévolat).

func onboardedPath() string { return filepath.Join(utils.GetT2CDataDir(), "onboarded") }

// setupState : où la CA est de confiance et si le PAC est actif.
func setupState() (caWhere, proxyWhere []string) {
	ca, err := assohttps.LoadOrCreate()
	if err != nil {
		return nil, nil
	}
	rep := sysint.Status(ca.CertPEMPath(), assohttps.Nick(), fmt.Sprintf("127.0.0.1:%d", proxyPort))
	return rep.CA, rep.Proxy
}

func (d *Dashboard) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, err := os.Stat(onboardedPath())
		caWhere, proxyWhere := setupState()
		writeJSON(w, map[string]any{
			"done":        err == nil,
			"caInstalled": len(caWhere) > 0, "caWhere": caWhere,
			"proxyOn": len(proxyWhere) > 0, "proxyWhere": proxyWhere,
			"canSetup":  os.Getenv("T2C_NO_BROWSER_SETUP") == "",
			"proxyPort": proxyPort,
		})
	case http.MethodPost:
		if err := os.MkdirAll(utils.GetT2CDataDir(), 0o700); err == nil {
			_ = os.WriteFile(onboardedPath(), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o600)
		}
		writeJSON(w, map[string]any{"ok": true})
	case http.MethodDelete: // refaire l'assistant
		_ = os.Remove(onboardedPath())
		writeJSON(w, map[string]any{"ok": true})
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

func setupAllowed(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return false
	}
	if os.Getenv("T2C_NO_BROWSER_SETUP") != "" {
		writeJSON(w, map[string]any{"ok": false, "error": "configuration désactivée (T2C_NO_BROWSER_SETUP)"})
		return false
	}
	return true
}

func (d *Dashboard) handleSetupCA(w http.ResponseWriter, r *http.Request) {
	if !setupAllowed(w, r) {
		return
	}
	ca, err := assohttps.LoadOrCreate()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	rep := sysint.InstallCA(ca.CertPEMPath(), assohttps.Nick())
	writeJSON(w, map[string]any{"ok": len(rep.CA) > 0, "report": rep})
}

func (d *Dashboard) handleSetupProxy(w http.ResponseWriter, r *http.Request) {
	if !setupAllowed(w, r) {
		return
	}
	d.ensureProxy()
	d.mu.Lock()
	on := d.proxyOn
	d.mu.Unlock()
	if !on {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("port %d occupé par un autre programme : proxy non activé", proxyPort)})
		return
	}
	rep := sysint.EnableProxy(utils.GetT2CDataDir(), fmt.Sprintf("127.0.0.1:%d", proxyPort))
	writeJSON(w, map[string]any{"ok": len(rep.Proxy) > 0 || len(rep.Errors) == 0, "report": rep})
}

func (d *Dashboard) handleBrowsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"browsers": sysint.Browsers()})
}

func (d *Dashboard) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Browser string `json:"browser"`
		URL     string `json:"url"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req) != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	if req.Browser == "" {
		req.Browser = readBrowserPref()
	} else {
		saveBrowserPref(req.Browser)
	}
	if err := sysint.OpenURL(req.Browser, req.URL); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleUpdateOpen : ouvre la page de mise à jour de l'avis de version signé
// par la racine (jamais une adresse fournie par l'interface ou un relais).
func (d *Dashboard) handleUpdateOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST attendu", http.StatusMethodNotAllowed)
		return
	}
	st, running := d.status()
	if !running || st.Version.URL == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "aucune page de mise à jour annoncée"})
		return
	}
	if err := sysint.OpenURL(readBrowserPref(), st.Version.URL); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func browserPrefPath() string { return filepath.Join(utils.GetT2CDataDir(), "browser") }
func readBrowserPref() string {
	b, _ := os.ReadFile(browserPrefPath())
	if len(b) == 0 {
		return "default"
	}
	return string(b)
}
func saveBrowserPref(id string) { _ = os.WriteFile(browserPrefPath(), []byte(id), 0o600) }

// handleSites : carnet = sites du catalogue signé, avec l'état d'hébergement.
func (d *Dashboard) handleSites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	st, _ := d.status()
	hosted := map[string]bool{}
	for _, h := range st.Hosted {
		hosted[h.Site] = true
	}
	sites := d.catalogSites(ctx)
	for _, s := range sites {
		id, _ := s["id"].(string)
		s["hosted"] = hosted[id]
		s["hostWanted"] = st.Hosting.All || containsStr(st.Hosting.Sites, id)
		s["url"] = "https://" + s["alias"].(string) + "/"
	}
	writeJSON(w, map[string]any{"sites": sites, "catalogVersion": st.CatalogVer, "browser": readBrowserPref()})
}

func containsStr(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// handleHosting : GET = choix et état du bénévolat ; POST = nouveau choix.
func (d *Dashboard) handleHosting(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	n := d.node
	d.mu.Unlock()
	if n == nil {
		http.Error(w, "non connecté", http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodPost {
		var h node.Hosting
		if json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&h) != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		if err := n.SetHosting(h); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}
	st := n.Status()
	writeJSON(w, map[string]any{
		"ok": true, "hosting": st.Hosting, "hosted": st.Hosted,
		"storeBytes": st.StoreBytes, "servedBytes": st.Served, "volunteer": st.Volunteer,
		"hiddenRelay": st.HiddenRelay, "relayLinks": st.RelayLinks,
		"entryRelay": st.EntryRelay, "entryVerified": st.EntryVerified,
	})
}
