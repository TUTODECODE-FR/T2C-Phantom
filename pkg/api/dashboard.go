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
	"t2c/pkg/service"
)

type Dashboard struct {
	mu           sync.Mutex
	node         *p2p.Node
	cancelNode   context.CancelFunc
	proxyPort    int
	exposePort   int
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
	if d.isServerMode {
		dir = filepath.Join(cwd, "web-server")
	}

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", fs)

	mux.HandleFunc("/api/status", d.handleStatus)
	mux.HandleFunc("/api/client/start", d.handleStartClient)
	mux.HandleFunc("/api/server/start", d.handleStartServer)
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
		"status":     status,
		"address":    d.address,
		"proxyPort":  d.proxyPort,
		"exposePort": d.exposePort,
		"relayAddr":  d.relayAddr,
		"domain":     d.domain,
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
		DisableSeeding bool   `json:"disable_seeding"`
		UseRelay       bool   `json:"use_relay"`
		RelayAddr      string `json:"relay_addr"`
		IsRelay        bool   `json:"is_relay"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	priv, pub, _ := crypto.GenerateKeyPair()
	addr, _ := crypto.GetT2CAddress(pub)
	d.address = addr
	d.relayAddr = req.RelayAddr

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelNode = cancel

	// Volunteer Relay Mode: Pass is_relay value to NewNode
	node, err := p2p.NewNode(ctx, priv, 0, req.IsRelay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.node = node

	// Private Mode (Leech only): Disable seeding if checked
	node.SwarmManager.DisableSeeding = req.DisableSeeding

	// Relay connection configuration
	if req.UseRelay {
		if req.RelayAddr != "" {
			node.ReserveRelay(ctx, req.RelayAddr)
		} else {
			// Connect to public free relays in the P2P community
			publicRelays := []string{
				"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfj4YgX94229AoWpRMok3v4nV77YvWkvyC9yK",
				"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2mHeqm4479ZJF7M89AgJz3u33u52Hj5kW6hC",
			}
			go func() {
				for _, relay := range publicRelays {
					if err := node.ReserveRelay(ctx, relay); err == nil {
						break // successfully reserved a slot on a public relay
					}
				}
			}()
		}
	}

	proxyAddr := "127.0.0.1:8080"
	d.proxyPort = 8080
	go proxy.StartProxyServer(node, proxyAddr, "data/carnet.json")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}



func (d *Dashboard) handleStartServer(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.node != nil {
		http.Error(w, "Node already running", http.StatusBadRequest)
		return
	}

	var req struct {
		LocalPort          int  `json:"local_port"`
		AllowGlobalCaching bool `json:"allow_caching"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.LocalPort <= 0 {
		http.Error(w, "Invalid local port", http.StatusBadRequest)
		return
	}

	priv, pub, _ := crypto.GenerateKeyPair()
	addr, _ := crypto.GetT2CAddress(pub)
	d.address = addr

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelNode = cancel

	// Start as a relay node on port 4001
	node, err := p2p.NewNode(ctx, priv, 4001, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.node = node

	// Server mode specific settings
	node.SwarmManager.DisableSeeding = false
	// We can control the swarm manager to restrict caching of this node's content if needed.
	// (Placeholder for actual content policy configuration)

	// Expose local service
	d.exposePort = req.LocalPort
	go service.ExposeService(ctx, node, req.LocalPort)

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
	d.exposePort = 0
	d.domain = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
