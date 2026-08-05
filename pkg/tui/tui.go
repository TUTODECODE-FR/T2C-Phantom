package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ClientConfig contient la configuration générée par 't2c init'
type ClientConfig struct {
	P2PPort        int    `json:"p2p_port"`
	MetricsPort    int    `json:"metrics_port"`
	VaultPass      string `json:"vault_pass"`
	UseI2P         bool   `json:"use_i2p"`
	SAMAddr        string `json:"sam_addr"`
	MaxPeers       int    `json:"max_peers"`
	MaxMemoryBytes int64  `json:"max_memory_bytes"`
}

// RunInitWizard lance l'assistant d'initialisation interactif
func RunInitWizard() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("==================================================")
	fmt.Println("   Assistant d'Initialisation T2C-Phantom         ")
	fmt.Println("==================================================")

	config := ClientConfig{
		P2PPort:        4001,
		MetricsPort:    2112,
		UseI2P:         false,
		SAMAddr:        "127.0.0.1:7656",
		MaxPeers:       50,
		MaxMemoryBytes: 256 * 1024 * 1024, // 256 MB
	}

	// 1. Port P2P
	fmt.Print("Entrez le port d'écoute P2P [4001] : ")
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		if port, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			config.P2PPort = port
		}
	}

	// 2. Chiffrement du cache
	fmt.Print("Voulez-vous activer le chiffrement du cache local ? (o/N) : ")
	if val, _ := reader.ReadString('\n'); strings.ToLower(strings.TrimSpace(val)) == "o" {
		fmt.Print("Entrez le mot de passe du coffre-fort (Vault) : ")
		pass, _ := reader.ReadString('\n')
		config.VaultPass = strings.TrimSpace(pass)
	}

	// 3. Configuration I2P
	fmt.Print("Voulez-vous forcer l'usage du réseau anonyme I2P ? (o/N) : ")
	if val, _ := reader.ReadString('\n'); strings.ToLower(strings.TrimSpace(val)) == "o" {
		config.UseI2P = true
		fmt.Print("Adresse du pont SAM I2P [127.0.0.1:7656] : ")
		if sam, _ := reader.ReadString('\n'); strings.TrimSpace(sam) != "" {
			config.SAMAddr = strings.TrimSpace(sam)
		}
	}

	// 4. Limites matérielles LXC
	fmt.Print("Limite de pairs P2P maximum [50] : ")
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		if peers, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			config.MaxPeers = peers
		}
	}

	fmt.Print("Limite de mémoire vive max en Mo (ex: 256) [256] : ")
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		if memMB, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
			config.MaxMemoryBytes = memMB * 1024 * 1024
		}
	}

	// Sauvegarde
	_ = os.MkdirAll("data", 0700)
	configPath := filepath.Join("data", "config.json")
	file, err := os.Create(configPath)
	if err == nil {
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(config)
		fmt.Printf("\n[+] Configuration enregistrée dans : %s\n", configPath)
	} else {
		fmt.Printf("\n[!] Erreur lors de l'enregistrement de la configuration : %v\n", err)
	}
	fmt.Println("Initialisation terminée avec succès !")
}

// StatsDashboard représente les données dynamiques à afficher sur la TUI
type StatsDashboard struct {
	ActiveProtocol   string
	ConnectedPeers   int
	DownloadSpeed    float64 // en Ko/s
	DownloadedBytes  int64
	TotalBytesToSync int64
	ProgressPercent  int
	SecurityAlerts   int
}

// DisplayTUIDashboard affiche une interface interactive légère avec des codes ANSI
func DisplayTUIDashboard(stopChan chan struct{}, statsFunc func() StatsDashboard) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			stats := statsFunc()

			// Nettoyage de l'écran et retour curseur en haut à gauche
			fmt.Print("\033[H\033[2J")

			fmt.Println("==============================================================")
			fmt.Println("          🛡️  T2C-PHANTOM SECURE CLIENT DAEMON V2.5           ")
			fmt.Println("==============================================================")

			// Protocole & État Réseau
			protoColor := "\033[32m" // Vert pour WSS
			if stats.ActiveProtocol == "I2P" {
				protoColor = "\033[35m" // Violet pour I2P
			}
			fmt.Printf(" Protocole Actif  : %s%s\033[0m\n", protoColor, stats.ActiveProtocol)
			fmt.Printf(" Pairs Connectés  : \033[36m%d\033[0m\n", stats.ConnectedPeers)
			fmt.Printf(" Vitesse Transfert: \033[33m%.2f Ko/s\033[0m\n", stats.DownloadSpeed)

			// Barre de Progression Merkle
			fmt.Println("\n--- Synchronisation des Chunks (Zero-Trust) ---")
			progressBar := drawProgressBar(stats.ProgressPercent, 30)
			fmt.Printf(" %s  %d%% (%.2f/%.2f Mo)\n", 
				progressBar, 
				stats.ProgressPercent, 
				float64(stats.DownloadedBytes)/(1024*1024), 
				float64(stats.TotalBytesToSync)/(1024*1024),
			)

			// État SOC & Alerte Sécurité
			alertColor := "\033[32m[Sain]"
			if stats.SecurityAlerts > 0 {
				alertColor = fmt.Sprintf("\033[31m[🚨 ATTENTION : %d alertes de sécurité détectées !]", stats.SecurityAlerts)
			}
			fmt.Printf("\n Statut de Sécurité (Loki/SOC) : %s\033[0m\n", alertColor)
			fmt.Println("==============================================================")
			fmt.Println(" Pour quitter le démon proprement, faites CTRL+C.")
		}
	}
}

func drawProgressBar(percent int, width int) string {
	filledLength := (percent * width) / 100
	if filledLength < 0 {
		filledLength = 0
	} else if filledLength > width {
		filledLength = width
	}

	filled := strings.Repeat("█", filledLength)
	empty := strings.Repeat("░", width-filledLength)

	return fmt.Sprintf("[%s%s]", filled, empty)
}
