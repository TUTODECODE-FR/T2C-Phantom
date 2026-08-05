package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCryptoStorage(t *testing.T) {
	passphrase := "MaClefSuperSecurisee123"
	originalData := []byte("Ceci est un fragment de cours de mathématiques super confidentiel.")

	// Test de chiffrement
	encrypted, err := EncryptData(originalData, passphrase)
	if err != nil {
		t.Fatalf("Chiffrement échoué : %v", err)
	}

	if bytes.Equal(originalData, encrypted) {
		t.Fatalf("Le texte chiffré est identique au texte d'origine")
	}

	// Test de déchiffrement avec la bonne clé
	decrypted, err := DecryptData(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Déchiffrement échoué : %v", err)
	}

	if !bytes.Equal(originalData, decrypted) {
		t.Fatalf("Les données déchiffrées ne correspondent pas à l'original")
	}

	// Test de déchiffrement avec une mauvaise clé
	_, err = DecryptData(encrypted, "MauvaiseClef")
	if err == nil {
		t.Fatalf("Déchiffrement réussi avec une mauvaise clé (vulnérabilité)")
	}
}

func TestMachineUUID(t *testing.T) {
	uuid := GetMachineUUID()
	if uuid == "" {
		t.Fatalf("L'UUID machine récupéré est vide")
	}
}

func TestFileCryptoHelpers(t *testing.T) {
	testDir, err := os.MkdirTemp("", "t2c-test-*")
	if err != nil {
		t.Fatalf("Impossible de créer le répertoire temporaire : %v", err)
	}
	defer os.RemoveAll(testDir)

	filePath := filepath.Join(testDir, "test_file.zip")
	content := []byte("Contenu de test du zip")

	// Set passphrase
	VaultPassphrase = "TestPassword"

	// Write encrypted
	err = WriteEncryptedFile(filePath, content)
	if err != nil {
		t.Fatalf("Écriture chiffrée échouée : %v", err)
	}

	// Verify file is indeed encrypted (i.e. different content)
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Lecture brute échouée : %v", err)
	}
	if bytes.Equal(rawBytes, content) {
		t.Fatalf("Le fichier a été écrit en clair")
	}

	// Read decrypted
	decrypted, err := ReadDecryptedFile(filePath)
	if err != nil {
		t.Fatalf("Lecture déchiffrée échouée : %v", err)
	}
	if !bytes.Equal(decrypted, content) {
		t.Fatalf("Le contenu déchiffré ne correspond pas à l'original")
	}

	// Test DecryptToTempFile
	tempPath, cleanup, err := DecryptToTempFile(filePath)
	if err != nil {
		t.Fatalf("Déchiffrement vers fichier temp échoué : %v", err)
	}
	defer cleanup()

	tempBytes, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Lecture du fichier temporaire échouée : %v", err)
	}
	if !bytes.Equal(tempBytes, content) {
		t.Fatalf("Le contenu du fichier temporaire ne correspond pas à l'original")
	}
}
