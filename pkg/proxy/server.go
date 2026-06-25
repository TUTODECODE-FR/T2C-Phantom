package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/network"
	"t2c/pkg/crypto"
	"t2c/pkg/onion"
	"t2c/pkg/p2p"
	"t2c/pkg/swarm"
)

func resolveCarnetAlias(carnetPath, host string) string {
	carnetData, err := os.ReadFile(carnetPath)
	if err != nil {
		return host
	}

	var carnet map[string]string
	if err := json.Unmarshal(carnetData, &carnet); err != nil {
		return host
	}

	if alias, exists := carnet[host]; exists {
		return alias
	}

	return host
}

func StartProxyServer(node *p2p.Node, listenAddr, carnetPath string) error {
	carnetData, _ := os.ReadFile(carnetPath)
	var carnet map[string]string
	json.Unmarshal(carnetData, &carnet)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hostPort := req.Host
		domain := strings.Split(hostPort, ":")[0]

		if !strings.HasSuffix(domain, ".t2c") {
			http.Error(w, "Proxy T2C: Seuls les domaines .t2c sont supportés", http.StatusForbidden)
			return
		}

		pubKeyHash := domain
		if mapped, ok := carnet[domain]; ok {
			pubKeyHash = mapped
		}
		if !strings.HasSuffix(pubKeyHash, ".t2c") {
			pubKeyHash = pubKeyHash + ".t2c"
		}

		siteDir, err := swarm.SyncSite(req.Context(), node.SwarmManager, node.Host, pubKeyHash)
		if err == nil {
			// --- SWARM MODE (Serve from local cache) ---
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
			http.FileServer(http.Dir(siteDir)).ServeHTTP(w, req)
			return
		}
		fmt.Println("Swarm Sync Error:", err)

		// --- DIRECT P2P TUNNEL MODE (Fallback for dynamic services/Git) ---
		peerID, parseErr := crypto.ParseT2CAddress(pubKeyHash)
		if parseErr != nil {
			http.Error(w, "Site Inaccessible: "+err.Error(), http.StatusBadGateway)
			return
		}

		// Attempt to resolve peer in DHT
		pi, _ := node.DHT.FindPeer(req.Context(), peerID)
		if pi.ID != "" {
			node.Host.Connect(req.Context(), pi)
		}

		// --- SELECT ONION ROUTING RELAYS (Brouillage du signal) ---
		var relays []peer.ID
		allPeers := node.Host.Network().Peers()
		for _, p := range allPeers {
			if p != node.Host.ID() && p != peerID {
				// Only select peers that actually run the T2C protocol and support onion routing
				protocols, err := node.Host.Peerstore().SupportsProtocols(p, "/t2c/onion/1.0")
				if err == nil && len(protocols) > 0 {
					relays = append(relays, p)
				}
			}
		}
		// Select up to 1 intermediate relay for route masking
		if len(relays) > 1 {
			relays = relays[:1]
		}
		
		// Bypass Onion Routing if we managed to connect directly (useful for NAT/LAN targets)
		if node.Host.Network().Connectedness(peerID) == network.Connected {
			relays = nil
		}

		// Open direct tunnel stream via Onion routing (Multi-hop path)
		stream, dialErr := onion.BuildOnionCircuit(req.Context(), node.Host, peerID, relays)
		if dialErr != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			
			htmlPage := `
			<!DOCTYPE html>
			<html lang="fr">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>T2C - Recherche en cours...</title>
				<meta http-equiv="refresh" content="10">
				<style>
					body { background-color: #0a0a0a; color: #ffffff; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100vh; margin: 0; text-align: center; }
					.loader { border: 4px solid #333; border-top: 4px solid #00ff88; border-radius: 50%%; width: 50px; height: 50px; animation: spin 1s linear infinite; margin-bottom: 20px; }
					@keyframes spin { 0%% { transform: rotate(0deg); } 100%% { transform: rotate(360deg); } }
					h1 { font-size: 24px; margin-bottom: 10px; color: #00ff88; }
					p { font-size: 16px; color: #888; max-width: 500px; line-height: 1.5; }
					.error-details { margin-top: 30px; padding: 15px; background: #1a1a1a; border-radius: 8px; font-family: monospace; font-size: 12px; color: #ff5555; word-wrap: break-word; max-width: 80%%; }
					.counter { font-weight: bold; color: #fff; }
				</style>
			</head>
			<body>
				<div class="loader"></div>
				<h1>Recherche du site sur le réseau P2P...</h1>
				<p>Le site <strong>%s</strong> n'a pas encore été trouvé dans l'annuaire mondial décentralisé (DHT).</p>
				<p>Si le serveur vient de démarrer, l'établissement de la route peut prendre quelques minutes. Cette page s'actualisera automatiquement dans <span id="countdown" class="counter">10</span> secondes.</p>
				
				<div class="error-details">Détails techniques : Swarm: %v | Direct: %v</div>

				<script>
					let count = 10;
					setInterval(() => {
						count--;
						if(count >= 0) document.getElementById("countdown").innerText = count;
					}, 1000);
				</script>
			</body>
			</html>
			`
			fmt.Fprintf(w, htmlPage, domain, err, dialErr)
			return
		}
		defer stream.Close()

		// Write HTTP request to stream
		err = req.Write(stream)
		if err != nil {
			http.Error(w, "Failed to write request to direct tunnel: "+err.Error(), http.StatusBadGateway)
			return
		}

		// Read HTTP response from stream
		bufReader := bufio.NewReader(stream)
		resp, err := http.ReadResponse(bufReader, req)
		if err != nil {
			http.Error(w, "Failed to read response from direct tunnel: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy headers to client response
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// Copy body to client response
		io.Copy(w, resp.Body)
	})

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	fmt.Printf("🚀 T2C Swarm Gateway running on http://%s\n", listenAddr)
	return server.ListenAndServe()
}
