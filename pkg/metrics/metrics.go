// Package metrics : compteurs Prometheus du client, exposés en local uniquement.
package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"t2c/pkg/node"
)

var (
	// CorruptedBlocks : morceaux reçus d'un bénévole et rejetés (hash ≠ manifeste signé).
	CorruptedBlocks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "t2c_blocks_corrupted_total",
		Help: "Morceaux rejetés car non conformes au manifeste signé.",
	})
	// PeerChunks : morceaux reçus de bénévoles.
	PeerChunks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "t2c_chunks_from_peers_total",
		Help: "Morceaux reçus de bénévoles (via le relais, chiffrés de bout en bout).",
	})
	// Failovers : morceaux récupérés à l'origine faute de bénévole.
	Failovers = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "t2c_failovers_total",
		Help: "Morceaux récupérés via l'origine faute de bénévole.",
	})
	// Connected : 1 si connecté au relais.
	Connected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "t2c_relay_connected",
		Help: "1 si le client est connecté au relais.",
	})
	// HostedSites : sites hébergés en tant que bénévole.
	HostedSites = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "t2c_hosted_sites",
		Help: "Nombre de sites hébergés en tant que bénévole.",
	})
)

func init() {
	prometheus.MustRegister(CorruptedBlocks, PeerChunks, Failovers, Connected, HostedSites)
}

// Observe met les compteurs à jour à partir de deux états successifs du nœud.
func Observe(prev, cur node.Status) {
	if d := cur.BadChunks - prev.BadChunks; d > 0 {
		CorruptedBlocks.Add(float64(d))
	}
	if d := cur.FromPeers - prev.FromPeers; d > 0 {
		PeerChunks.Add(float64(d))
	}
	if d := cur.FromOrigin - prev.FromOrigin; d > 0 {
		Failovers.Add(float64(d))
	}
	if cur.Connected {
		Connected.Set(1)
	} else {
		Connected.Set(0)
	}
	HostedSites.Set(float64(len(cur.Hosted)))
}

// StartMetricsServer : /metrics sur addr (ex. 127.0.0.1:2112 — jamais exposé publiquement).
func StartMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("métriques Prometheus sur http://%s/metrics", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("métriques: %v", err)
	}
}
