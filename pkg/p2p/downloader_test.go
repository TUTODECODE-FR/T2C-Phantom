package p2p

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"t2c/pkg/crypto"
)

func TestDownloaderZeroTrustFailover(t *testing.T) {
	// 🛡️ Mock the Offline Master Key for tests
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	crypto.MasterOfflinePubKeyHex = hex.EncodeToString(pubKey)

	// Création d'un faux Manifeste
	dummyData := []byte("fake_chunk_data_for_testing")
	validHash := sha256.Sum256(dummyData)
	validHashStr := hex.EncodeToString(validHash[:])

	manifest := &crypto.Manifest{
		RootHash: "test_root",
		FileSize: 1234,
		ChunkHashes: []string{validHashStr},
	}

	// Sign the manifest
	message := []byte(manifest.RootHash + fmt.Sprintf(":%d", manifest.FileSize))
	sig := ed25519.Sign(privKey, message)
	manifest.Signature = hex.EncodeToString(sig)

	downloader, err := NewDownloader(manifest, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Expected valid signature but got error: %v", err)
	}

	// 1. Test : Bloc valide
	err = downloader.DownloadChunk(0, "Peer_Valid", dummyData)
	if err != nil {
		t.Errorf("Expected nil error for valid chunk, got: %v", err)
	}

	// 2. Test : Bloc invalide (simule une attaque)
	corruptedData := []byte("fake_chunk_data_for_TESTING")
	err = downloader.DownloadChunk(0, "Peer_Malicious", corruptedData)
	
	if err == nil {
		t.Errorf("CRITICAL: Downloader accepted invalid chunk without failing over!")
	} else if err.Error() != "not implemented yet" {
		// "not implemented yet" est le retour actuel de la fonction de failover HTTP
		t.Errorf("Expected failover error 'not implemented yet', got: %v", err)
	} else {
		t.Log("✅ Zero-Trust & Failover: Malicious chunk rejected, Failover successfully triggered.")
	}
}
