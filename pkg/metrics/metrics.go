package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// CorruptedBlocks enregistre le nombre de blocs rejetés par la vérification RAM (Zero-Trust)
	CorruptedBlocks = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "t2c_blocks_corrupted_total",
			Help: "Nombre total de fragments corrompus interceptés en mémoire.",
		},
	)

	// BannedPeers enregistre le nombre de pairs exclus par le tracker
	BannedPeers = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "t2c_peers_banned_total",
			Help: "Nombre total de pairs malveillants définitivement bannis du réseau.",
		},
	)

	// Failovers enregistre le nombre de basculements vers le proxy HTTP
	Failovers = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "t2c_failovers_total",
			Help: "Nombre total de basculements d'urgence vers le LXC Serveur (Failover).",
		},
	)
)

func init() {
	// Enregistrement des métriques auprès de Prometheus
	prometheus.MustRegister(CorruptedBlocks)
	prometheus.MustRegister(BannedPeers)
	prometheus.MustRegister(Failovers)
}

// StartMetricsServer démarre un serveur HTTP minimal dédié au scraping Prometheus/Grafana
func StartMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("📊 Prometheus SOC Metrics Server listening on http://0.0.0.0:%s/metrics", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}
