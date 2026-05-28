package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
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
				relays = append(relays, p)
			}
		}
		// Select up to 1 intermediate relay for route masking
		if len(relays) > 1 {
			relays = relays[:1]
		}

		// Open direct tunnel stream via Onion routing (Multi-hop path)
		stream, dialErr := onion.BuildOnionCircuit(req.Context(), node.Host, peerID, relays)
		if dialErr != nil {
			http.Error(w, fmt.Sprintf("Service Inaccessible (Swarm & Direct indisponibles). Swarm: %v, Direct: %v", err, dialErr), http.StatusBadGateway)
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
		Addr:    listenAddr,
		Handler: handler,
	}

	fmt.Printf("🚀 T2C Swarm Gateway running on http://%s\n", listenAddr)
	return server.ListenAndServe()
}
