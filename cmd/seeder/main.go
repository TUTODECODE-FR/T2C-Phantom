package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	t2ccrypto "t2c/pkg/crypto"
	"t2c/pkg/swarm"

	ic "github.com/libp2p/go-libp2p/core/crypto"
)

// zipDirectory lit un répertoire complet et le compresse en mémoire
func zipDirectory(source string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(f, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	err = w.Close()
	return buf.Bytes(), err
}

func main() {
	siteDir := flag.String("site-dir", "", "Chemin vers le dossier du site Web (ex: /chemin/vers/Documents/tutodecode_site)")
	privKeyB64 := flag.String("key", "", "Clé privée du CT100 en Base64")
	flag.Parse()

	if *siteDir == "" || *privKeyB64 == "" {
		fmt.Println("Erreur : --site-dir et --key sont obligatoires.")
		os.Exit(1)
	}

	fmt.Println("=== T2C Swarm Seeder ===")
	
	// 1. Charger la clé privée
	privBytes, err := base64.StdEncoding.DecodeString(*privKeyB64)
	if err != nil {
		fmt.Printf("❌ Format de clé invalide : %v\n", err)
		os.Exit(1)
	}
	privKey, err := ic.UnmarshalPrivateKey(privBytes)
	if err != nil {
		fmt.Printf("❌ Impossible de lire la clé privée : %v\n", err)
		os.Exit(1)
	}

	// 2. Récupérer l'adresse T2C
	addr, err := t2ccrypto.GetT2CAddress(privKey.GetPublic())
	if err != nil {
		fmt.Printf("❌ Erreur crypto : %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Identité reconnue : %s\n", addr)

	// 3. Zipper le site
	fmt.Println("📦 Compression du site en cours...")
	zipBytes, err := zipDirectory(*siteDir)
	if err != nil {
		fmt.Printf("❌ Erreur de compression : %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Site compressé (%d octets)\n", len(zipBytes))

	// 4. Générer la signature cryptographique (Record)
	fmt.Println("🔐 Génération de la signature cryptographique inforgeable...")
	record, err := swarm.CreateRecord(privKey, zipBytes)
	if err != nil {
		fmt.Printf("❌ Erreur de signature : %v\n", err)
		os.Exit(1)
	}

	recordBytes, _ := swarm.SerializeRecord(record)

	// 5. Créer l'arborescence finale
	outDir := "export_swarm/data"
	zipsDir := filepath.Join(outDir, "swarm_zips")
	sitesDir := filepath.Join(outDir, "sites", addr)

	os.MkdirAll(zipsDir, 0755)
	os.MkdirAll(sitesDir, 0755)

	zipPath := filepath.Join(zipsDir, record.ZipHash+".zip")
	recordPath := filepath.Join(sitesDir, "record.json")

	os.WriteFile(zipPath, zipBytes, 0644)
	os.WriteFile(recordPath, recordBytes, 0644)

	fmt.Println("====================================================")
	fmt.Println("🎉 SUCCÈS : Capsule cryptographique créée avec succès !")
	fmt.Println("Dossier à transférer sur le CT100 :")
	fmt.Println("-> export_swarm/data/")
	fmt.Println("====================================================")
}
