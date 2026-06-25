package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"t2c/pkg/api"
	"t2c/pkg/crypto"
	"t2c/pkg/p2p"
	"t2c/pkg/proxy"
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

		// Nouveaux drapeaux pour la V2 (Relay)
		useRelay := startCmd.String("use-relay", "", "Hide behind a specific Relay (multiaddr) to mask IP")

		startCmd.Parse(os.Args[2:])

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
		// For the client, we hardcode isRelay to false for security.
		node, err := p2p.NewNode(ctx, privKey, *port, false)
		if err != nil {
			fmt.Println("Failed to start node:", err)
			return
		}

		addr, _ := crypto.GetT2CAddress(privKey.GetPublic())
		fmt.Println("=== T2C Client Daemon Started ===")
		fmt.Printf("Node Address: %s\n", addr)
		fmt.Printf("Peer ID: %s\n", node.Host.ID().String())
		fmt.Println("P2P Listening on:")
		for _, a := range node.Host.Addrs() {
			fmt.Printf("  %s/p2p/%s\n", a.String(), node.Host.ID().String())
		}

		// Start proxy FIRST so it is immediately available, even before relay is up.
		if *proxyPort > 0 {
			proxyAddr := fmt.Sprintf("127.0.0.1:%d", *proxyPort)
			go func() {
				if err := proxy.StartProxyServer(node, proxyAddr, *carnetPath); err != nil {
					fmt.Println("Proxy error:", err)
				}
			}()
		}

		// Bootstrap and relay reservation run in background — non-blocking.
		go func() {
			var peers []string
			if *bootstrap != "" {
				peers = strings.Split(*bootstrap, ",")
			}
			bootstrapCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := node.Bootstrap(bootstrapCtx, peers); err != nil {
				fmt.Println("Bootstrap info:", err)
			}

			// Réservation d'un relais si demandé pour masquer son IP
			if *useRelay != "" {
				relayCtx, cancelRelay := context.WithTimeout(ctx, 20*time.Second)
				defer cancelRelay()
				if err := node.ReserveRelay(relayCtx, *useRelay); err != nil {
					fmt.Println("Failed to reserve relay:", err)
				}
			}
		}()

		// Keep alive
		select {}

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("T2C Phantom - Client Network Node")
	fmt.Println("\nUsage:")
	fmt.Println("  t2c genkey           Generate a new identity")
	fmt.Println("  t2c start            Start the daemon via CLI")
	fmt.Println("  t2c dashboard-client Start the User Interface (Port 9090)")
	fmt.Println("\nStart Options:")
	fmt.Println("  --key <base64>       Private key (Optional: if empty, an ephemeral key is generated)")
	fmt.Println("  --port <int>         P2P Port (default 4001)")
	fmt.Println("  --proxy <int>        Enable local HTTP proxy on this port")
	fmt.Println("  --bootstrap <addr>   Comma-separated bootstrap multiaddrs")
	fmt.Println("  --carnet <path>      Path to carnet.json (DNS aliases)")
	fmt.Println("  --use-relay <addr>   Connect to a Relay to mask your real IP")
}
