package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"t2c/pkg/crypto"
	"t2c/pkg/p2p"
	"t2c/pkg/proxy"
)

type Dashboard struct {
	mu           sync.Mutex
	node         *p2p.Node
	cancelNode   context.CancelFunc
	proxyPort    int
	address      string
	relayAddr    string
	domain       string
	isServerMode bool
}

func NewDashboard(isServerMode bool) *Dashboard {
	return &Dashboard{isServerMode: isServerMode}
}

func (d *Dashboard) Start(port int) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "web-client")

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", fs)

	mux.HandleFunc("/api/status", d.handleStatus)
	mux.HandleFunc("/api/client/start", d.handleStartClient)
	mux.HandleFunc("/api/stop", d.handleStop)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("\n=== T2C Dashboard ===\n")
	fmt.Printf("Web Interface ready: http://%s\n", addr)
	fmt.Printf("Serving files from: %s\n", dir)
	fmt.Printf("=====================\n\n")
	return http.ListenAndServe(addr, mux)
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := "stopped"
	if d.node != nil {
		status = "running"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"address":   d.address,
		"proxyPort": d.proxyPort,
		"relayAddr": d.relayAddr,
		"domain":    d.domain,
	})
}

func (d *Dashboard) handleStartClient(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.node != nil {
		http.Error(w, "Node already running", http.StatusBadRequest)
		return
	}

	var req struct {
		UseRelay  bool   `json:"use_relay"`
		RelayAddr string `json:"relay_addr"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	priv, pub, _ := crypto.GenerateKeyPair()
	addr, _ := crypto.GetT2CAddress(pub)
	d.address = addr
	d.relayAddr = req.RelayAddr

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelNode = cancel

	node, err := p2p.NewNode(ctx, priv, 0, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.node = node

	// Relay and DHT bootstrap configuration
	go func() {
		bootstrapPeers := []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
			"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		}

		if req.UseRelay {
			if req.RelayAddr != "" {
				if err := node.ReserveRelay(ctx, req.RelayAddr); err == nil {
					serverPeerID := "12D3KooWGbs26VDbtjpg5gwTGiXcA5WS2MNVXRqKZqMuQP2nrftx"
					serverRelayAddr := fmt.Sprintf("%s/p2p-circuit/p2p/%s", req.RelayAddr, serverPeerID)
					bootstrapPeers = append(bootstrapPeers, serverRelayAddr)
				} else {
					fmt.Println("Failed to reserve custom relay:", err)
				}
			} else {
				publicRelays := []string{
					"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
				}
				for _, relay := range publicRelays {
					if err := node.ReserveRelay(ctx, relay); err == nil {
						serverPeerID := "12D3KooWGbs26VDbtjpg5gwTGiXcA5WS2MNVXRqKZqMuQP2nrftx"
						serverRelayAddr := fmt.Sprintf("%s/p2p-circuit/p2p/%s", relay, serverPeerID)
						bootstrapPeers = append(bootstrapPeers, serverRelayAddr)
						break
					}
				}
			}
		}

		if err := node.Bootstrap(ctx, bootstrapPeers); err == nil {
			fmt.Println("🚀 Client DHT Bootstrap complete!")
		} else {
			fmt.Println("Client DHT Bootstrap error:", err)
		}
	}()

	proxyAddr := "127.0.0.1:8080"
	d.proxyPort = 8080
	go proxy.StartProxyServer(node, proxyAddr, "data/carnet.json")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (d *Dashboard) handleStop(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancelNode != nil {
		d.cancelNode()
	}
	if d.node != nil {
		d.node.Host.Close()
	}

	d.node = nil
	d.address = ""
	d.proxyPort = 0
	d.domain = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
