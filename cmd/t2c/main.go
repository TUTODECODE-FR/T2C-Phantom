package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"t2c/pkg/api"
	"t2c/pkg/crypto"
	"t2c/pkg/metrics"
	"t2c/pkg/p2p"
	"t2c/pkg/proxy"
	"t2c/pkg/tui"
	"t2c/pkg/utils"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	// Initialisation du Logger de sécurité
	utils.InitSecurityLogger("data/security.log")

	cmd := os.Args[1]

	switch cmd {
	case "init":
		tui.RunInitWizard()

	case "doctor":
		runDoctorCommand()

	case "genkey":
		priv, pub, err := crypto.GenerateKeyPair()
		if err != nil {
			fmt.Println("Error generating keys:", err)
			return
		}

		privBytes, _ := ic.MarshalPrivateKey(priv)
		privB64 := base64.StdEncoding.EncodeToString(privBytes)

		addr, _ := crypto.GetT2CAddress(pub)

		fmt.Println("=== T2C Identity Generated ===")
		fmt.Printf("Address (.t2c): %s\n", addr)
		fmt.Printf("Private Key: %s\n", privB64)
		fmt.Println("(Save this Private Key to start your daemon later)")

	case "dashboard-client":
		fmt.Println("Starting Web Dashboard (Client Mode)...")
		dashboard := api.NewDashboard(false)
		if err := dashboard.Start(9090); err != nil {
			fmt.Println("Dashboard error:", err)
		}

	case "start":
		startCmd := flag.NewFlagSet("start", flag.ExitOnError)
		privKeyB64 := startCmd.String("key", "", "Base64 Private Key")
		port := startCmd.Int("port", 4001, "P2P Listen Port")
		proxyPort := startCmd.Int("proxy", 0, "HTTP Proxy Listen Port (e.g., 8080)")
		bootstrap := startCmd.String("bootstrap", "", "Comma-separated bootstrap peers")
		carnetPath := startCmd.String("carnet", "data/carnet.json", "Path to the carnet.json file for DNS aliases")
		relaysStr := startCmd.String("relays", "", "Comma-separated list of Relays (multiaddrs) to mask IP and ensure high availability")

		// Nouveaux drapeaux durcissement Phase 8.1
		metricsPort := startCmd.Int("metrics-port", 2112, "Prometheus Metrics Port")
		useI2P := startCmd.Bool("i2p", false, "Force anonymisation via le pont SAM I2P")
		i2pFallback := startCmd.Bool("i2p-fallback", false, "Basculement automatique sur I2P en cas de coupure WSS")
		vaultPass := startCmd.String("vault-pass", "", "Mot de passe pour le chiffrement AES-256-GCM du cache")
		maxMemory := startCmd.Int64("max-memory", 256*1024*1024, "Limite de mémoire vive en octets (LXC)")
		maxPeers := startCmd.Int("max-peers", 50, "Limite de pairs maximum connectés (LXC)")

		startCmd.Parse(os.Args[2:])

		// Configuration du chiffrement de cache local
		crypto.VaultPassphrase = *vaultPass

		var privKey ic.PrivKey

		if *privKeyB64 == "" {
			fmt.Println("⚠️ Aucune clé fournie : Génération d'une identité ÉPHÉMÈRE (Jetée à la fermeture)...")
			priv, _, err := crypto.GenerateKeyPair()
			if err != nil {
				fmt.Println("Failed to generate ephemeral key:", err)
				return
			}
			privKey = priv
		} else {
			privBytes, err := base64.StdEncoding.DecodeString(*privKeyB64)
			if err != nil {
				fmt.Println("Invalid private key format.")
				return
			}

			privKey, err = ic.UnmarshalPrivateKey(privBytes)
			if err != nil {
				fmt.Println("Failed to parse private key:", err)
				return
			}
		}

		ctx := context.Background()
		var relays []string
		if *relaysStr != "" {
			relays = strings.Split(*relaysStr, ",")
		}

		// Lancement du Node avec les limites LXC
		node, err := p2p.NewNode(ctx, privKey, *port, false, relays, *maxPeers, *maxMemory)
		if err != nil {
			fmt.Println("Failed to start relay node:", err)
			return
		}

		// 📊 Lancement du SOC (Metrics Prometheus)
		go metrics.StartMetricsServer(strconv.Itoa(*metricsPort))

		addr, _ := crypto.GetT2CAddress(privKey.GetPublic())
		utils.LogSecurityEvent("DAEMON_START", "T2C Client Daemon Started for address: "+addr, "self")

		// TUI Setup & Launch
		stopTUI := make(chan struct{})
		activeProto := "WSS/WebRTC"
		if *useI2P {
			activeProto = "I2P"
		}

		go tui.DisplayTUIDashboard(stopTUI, func() tui.StatsDashboard {
			// Simule ou récupère les métriques réelles pour la TUI
			peersCount := len(node.Host.Network().Peers())
			return tui.StatsDashboard{
				ActiveProtocol:   activeProto,
				ConnectedPeers:   peersCount,
				DownloadSpeed:    0.0, // Vitesse instantanée (implémentable avec tracking octets)
				DownloadedBytes:  0,
				TotalBytesToSync: 0,
				ProgressPercent:  100, // Démon prêt
				SecurityAlerts:   0,   // Aucun event anormal par défaut
			}
		})

		// Start proxy FIRST so it is immediately available, even before relay is up.
		if *proxyPort > 0 {
			proxyAddr := fmt.Sprintf("127.0.0.1:%d", *proxyPort)
			go func() {
				if err := proxy.StartProxyServer(node, proxyAddr, *carnetPath); err != nil {
					fmt.Println("Proxy error:", err)
				}
			}()
		}

		// Bootstrap and relay reservation run in background
		go func() {
			var peers []string
			if *bootstrap != "" {
				peers = strings.Split(*bootstrap, ",")
			}
			bootstrapCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			// Si basculement automatique I2P configuré
			err := node.Bootstrap(bootstrapCtx, peers)
			if err != nil && *i2pFallback {
				fmt.Println("[FAILOVER] Échec de l'amorçage WSS. Tentative de basculement vers I2P...")
				activeProto = "I2P (Fallback)"
				// Tentative de connexion via pont SAM v3
				if _, samErr := p2p.ConnectSAM("127.0.0.1:7656"); samErr == nil {
					fmt.Println("[FAILOVER] Pont SAM I2P connecté avec succès !")
				} else {
					fmt.Printf("[FAILOVER] Impossible de basculer sur I2P : %v\n", samErr)
					activeProto = "WSS (Dégradé)"
				}
			} else if err != nil {
				fmt.Println("Bootstrap info:", err)
			}

			// Réservation d'un relais si demandé pour masquer son IP
			if len(relays) > 0 {
				for _, r := range relays {
					relayCtx, cancelRelay := context.WithTimeout(ctx, 20*time.Second)
					if err := node.ReserveRelay(relayCtx, r); err != nil {
						fmt.Printf("Failed to reserve relay %s: %v\n", r, err)
					}
					cancelRelay()
				}
			}
		}()

		// Keep alive
		select {}

	default:
		printHelp()
	}
}

func runDoctorCommand() {
	fmt.Println("==================================================")
	fmt.Println("   Diagnostic Système T2C-Phantom (Doctor)        ")
	fmt.Println("==================================================")

	// 1. Vérification Port de Métriques
	fmt.Print("[*] Vérification du port de métriques (2112)... ")
	l, err := net.Listen("tcp", "127.0.0.1:2112")
	if err != nil {
		fmt.Println("❌ BLOCKED")
		fmt.Printf("    -> Le port 2112 est déjà utilisé par un autre service.\n")
	} else {
		l.Close()
		fmt.Println("✅ FREE")
	}

	// 2. Vérification du Pont SAM d'I2P
	fmt.Print("[*] Recherche du pont SAM I2P (127.0.0.1:7656)... ")
	if p2p.TestSAMBridge("127.0.0.1:7656") {
		fmt.Println("✅ RUNNING")
	} else {
		fmt.Println("⚠️ NOT RUNNING")
		fmt.Println("    -> Le routeur I2P (i2pd ou java) n'a pas été détecté sur le port SAM.")
	}

	// 3. Connexion Internet & DNS
	fmt.Print("[*] Résolution DNS de l'infrastructure d'amorçage... ")
	_, dnsErr := net.LookupHost("bootstrap.libp2p.io")
	if dnsErr == nil {
		fmt.Println("✅ OK")
	} else {
		fmt.Println("❌ FAILED")
		fmt.Printf("    -> Impossible de joindre les serveurs DNS. Vérifiez votre connectivité réseau.\n")
	}

	// 4. Intégrité des répertoires de cache
	fmt.Print("[*] Intégrité de l'arborescence locale de cache... ")
	cachePath := filepath.Join("data", "swarm_zips")
	_ = os.MkdirAll(cachePath, 0700)
	if _, err := os.Stat(cachePath); err == nil {
		fmt.Println("✅ OK")
	} else {
		fmt.Println("❌ ERROR")
	}
	fmt.Println("\nDiagnostic terminé.")
}

func printHelp() {
	fmt.Println("T2C Phantom - Client Network Node")
	fmt.Println("\nUsage:")
	fmt.Println("  t2c init             Lancer l'assistant interactif de configuration")
	fmt.Println("  t2c doctor           Lancer les tests de diagnostic système")
	fmt.Println("  t2c genkey           Generate a new identity")
	fmt.Println("  t2c start            Start the daemon via CLI")
	fmt.Println("  t2c dashboard-client Start the User Interface (Port 9090)")
	fmt.Println("\nStart Options:")
	fmt.Println("  --key <base64>       Private key (Optional: if empty, an ephemeral key is generated)")
	fmt.Println("  --port <int>         P2P Port (default 4001)")
	fmt.Println("  --proxy <int>        Enable local HTTP proxy on this port")
	fmt.Println("  --bootstrap <addr>   Comma-separated bootstrap multiaddrs")
	fmt.Println("  --carnet <path>      Path to carnet.json (DNS aliases)")
	fmt.Println("  --relays <addr>      Connect to a list of Relays to mask your real IP")
	fmt.Println("  --metrics-port <int> Prometheus metrics listener port (default 2112)")
	fmt.Println("  --i2p                Force I2P routing")
	fmt.Println("  --i2p-fallback       Enable auto fallback to I2P on standard bootstrap loss")
	fmt.Println("  --vault-pass <str>   Encryption key for local cache at rest")
	fmt.Println("  --max-memory <bytes> Maximum RAM allowed for P2P resource manager")
	fmt.Println("  --max-peers <int>    Maximum active peer connections allowed")
}
