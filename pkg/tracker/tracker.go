package tracker

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"t2c/pkg/crypto"
	"t2c/pkg/metrics"
)

// Tracker représente l'annuaire (LXC Serveur)
type Tracker struct {
	mu          sync.Mutex
	reports     map[string]map[string]bool // map[maliciousPeerID]map[reporterIP]bool
	bans        map[string]int             // Niveau de ban par peerID
	BanThreshold int
}

func NewTracker(threshold int) *Tracker {
	return &Tracker{
		reports:     make(map[string]map[string]bool),
		bans:        make(map[string]int),
		BanThreshold: threshold,
	}
}

// HandleReportPeer recoit un signalement d'un client
func (t *Tracker) HandleReportPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MaliciousPeerID string `json:"malicious_peer_id"`
		PoWNonce        string `json:"pow_nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// 🛡️ Proof-of-Work Verification (Anti-Sybil Cost)
	// The challenge is the MaliciousPeerID to bind the PoW to this specific report.
	if !crypto.VerifyPoW(req.MaliciousPeerID, req.PoWNonce, crypto.TargetDifficulty) {
		http.Error(w, "Invalid Proof-of-Work", http.StatusForbidden)
		return
	}

	// Récupérer l'IP du rapporteur (X-Forwarded-For transmis par le LXC Proxy)
	reporterIP := r.Header.Get("X-Forwarded-For")
	if reporterIP == "" {
		reporterIP = r.RemoteAddr
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.reports[req.MaliciousPeerID] == nil {
		t.reports[req.MaliciousPeerID] = make(map[string]bool)
	}

	// Si cette IP n'a pas encore signalé ce peer, on ajoute
	if !t.reports[req.MaliciousPeerID][reporterIP] {
		t.reports[req.MaliciousPeerID][reporterIP] = true
		uniqueReports := len(t.reports[req.MaliciousPeerID])

		log.Printf("[IMMUNE SYSTEM] Peer %s reported by IP %s (%d/%d)", req.MaliciousPeerID, reporterIP, uniqueReports, t.BanThreshold)

		if uniqueReports >= t.BanThreshold {
			t.bans[req.MaliciousPeerID]++
			
			// 📊 [SOC] Incrémenter la métrique des pairs bannis
			metrics.BannedPeers.Inc()
			
			log.Printf("[IMMUNE SYSTEM] 🚨 BANNED Peer %s (Level %d)", req.MaliciousPeerID, t.bans[req.MaliciousPeerID])
			// Logique de blacklist ou d'expulsion de l'annuaire ici
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"report_received"}`))
}
