package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"t2c/pkg/crypto"
	"t2c/pkg/api"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

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

	case "dashboard":
		fmt.Println("Starting Web Dashboard (Client Mode)...")
		dashboard := api.NewDashboard(false)
		if err := dashboard.Start(9090); err != nil {
			fmt.Println("Dashboard error:", err)
		}

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("T2C-Phantom Overlay Network Node")
	fmt.Println("\nUsage:")
	fmt.Println("  t2c genkey         Generate a new identity")
	fmt.Println("  t2c dashboard      Start the User Interface (Port 9090)")
}
