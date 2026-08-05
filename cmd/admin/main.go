package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"t2c/pkg/crypto"
	"t2c/pkg/p2p"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: t2c-admin start --key <base64>")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "start":
		startCmd := flag.NewFlagSet("start", flag.ExitOnError)
		privKeyB64 := startCmd.String("key", "", "Base64 Private Key")
		bootstrap := startCmd.String("bootstrap", "", "Comma-separated bootstrap peers")
		useRelay := startCmd.String("use-relay", "", "Connect to a Relay to mask your real IP")
		// The expose flag was here in admin originally!
		expose := startCmd.Int("expose", 0, "Expose a local port")

		startCmd.Parse(os.Args[2:])

		privBytes, err := base64.StdEncoding.DecodeString(*privKeyB64)
		if err != nil {
			fmt.Println("Invalid private key format.")
			return
		}

		privKey, err := ic.UnmarshalPrivateKey(privBytes)
		if err != nil {
			fmt.Println("Failed to parse private key:", err)
			return
		}

		ctx := context.Background()
		// FOR ADMIN NODE, isRelay IS TRUE! This forces Public reachability for DHT Kademlia!
		node, err := p2p.NewNode(ctx, privKey, 4001, true, nil, 0, 0)
		if err != nil {
			fmt.Println("Failed to start node:", err)
			return
		}
		
		node.ExitPort = *expose

		addr, _ := crypto.GetT2CAddress(privKey.GetPublic())
		fmt.Println("=== T2C Host Node Started (CLI Mode) ===")
		fmt.Printf("Address (.t2c): %s\n", addr)
		fmt.Printf("Caching allowed: false\n")

		if *expose > 0 {
			fmt.Printf("Service on local port %d exposed via P2P on Peer ID: %s\n", *expose, node.Host.ID().String())
		}

		go func() {
			var peers []string
			if *bootstrap != "" {
				peers = strings.Split(*bootstrap, ",")
			}
			bootstrapCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			node.Bootstrap(bootstrapCtx, peers)

			if *useRelay != "" {
				relayCtx, cancelRelay := context.WithTimeout(ctx, 20*time.Second)
				defer cancelRelay()
				node.ReserveRelay(relayCtx, *useRelay)
			}
		}()

		select {}
	}
}
