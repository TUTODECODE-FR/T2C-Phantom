package tracker

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"t2c/pkg/crypto"
)

func TestTrackerImmuneSystem(t *testing.T) {
	// Seuil de 3 IPs pour bannir
	tracker := NewTracker(3)
	
	// Utilisation de 127.0.0.1 (IPv4) au lieu de ::1 pour la compatibilité avec GitLab CI et Docker
	server := httptest.NewUnstartedServer(http.HandlerFunc(tracker.HandleReportPeer))
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = l
	server.Start()
	defer server.Close()

	maliciousPeer := "QmFakeMaliciousPeerID123"

	// Helper pour faire une requete POST avec PoW
	makeReport := func(ip string) {
		nonce := crypto.GeneratePoW(maliciousPeer, crypto.TargetDifficulty)
		reqBody := []byte(`{"malicious_peer_id": "` + maliciousPeer + `", "pow_nonce": "` + nonce + `"}`)
		req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatal(err)
		}
		// Injection de l'IP simulée par le LXC Proxy
		req.Header.Set("X-Forwarded-For", ip)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	// 1. Scénario de triche (Review Bombing)
	// La même IP signale 3 fois le pair
	makeReport("192.168.1.50")
	makeReport("192.168.1.50")
	makeReport("192.168.1.50")

	tracker.mu.Lock()
	banLevel := tracker.bans[maliciousPeer]
	uniqueReports := len(tracker.reports[maliciousPeer])
	tracker.mu.Unlock()

	if uniqueReports != 1 {
		t.Errorf("Expected 1 unique report, got %d", uniqueReports)
	}
	if banLevel > 0 {
		t.Errorf("CRITICAL: Peer was banned from a single IP spam (Review Bombing vulnerability)")
	} else {
		t.Log("✅ Immune Check: Spam from single IP ignored.")
	}

	// 2. Scénario Réel (Consensus de triche)
	// 2 nouvelles IPs indépendantes signalent le pair (Total = 3)
	makeReport("10.0.0.1")
	makeReport("8.8.8.8")

	tracker.mu.Lock()
	banLevel = tracker.bans[maliciousPeer]
	uniqueReports = len(tracker.reports[maliciousPeer])
	tracker.mu.Unlock()

	if uniqueReports != 3 {
		t.Errorf("Expected 3 unique reports, got %d", uniqueReports)
	}
	if banLevel == 0 {
		t.Errorf("CRITICAL: Peer was NOT banned despite %d independent reports", uniqueReports)
	} else {
		t.Logf("✅ Immune Check: Peer was successfully banned (Level %d) after consensus.", banLevel)
	}
}
