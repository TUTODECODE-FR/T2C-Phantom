package crypto

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestMerkleManifestAndValidation(t *testing.T) {
	// 1. Création d'un fichier de test (ex: 500 Ko)
	testDir := t.TempDir()
	filePath := filepath.Join(testDir, "cours_test.bin")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Écrire 500 Ko de données aléatoires
	dataSize := int64(500 * 1024)
	randomData := make([]byte, dataSize)
	_, err = rand.Read(randomData)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	file.Write(randomData)
	file.Close()

	// 2. Génération du Manifeste
	manifest, err := GenerateManifest(filePath)
	if err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}

	// 500 Ko avec des blocs de 256 Ko (256 * 1024) devrait faire 2 chunks entiers (256+244)
	expectedChunks := 2
	if manifest.TotalChunks != expectedChunks {
		t.Errorf("Expected %d chunks, got %d", expectedChunks, manifest.TotalChunks)
	}

	if manifest.FileSize != dataSize {
		t.Errorf("Expected file size %d, got %d", dataSize, manifest.FileSize)
	}

	// 3. Test de validation en RAM (Cas de succès)
	chunk0Data := randomData[:ChunkSize]
	if !ValidateChunk(chunk0Data, manifest.ChunkHashes[0]) {
		t.Errorf("Chunk 0 failed validation, expected success")
	}

	chunk1Data := randomData[ChunkSize:]
	if !ValidateChunk(chunk1Data, manifest.ChunkHashes[1]) {
		t.Errorf("Chunk 1 failed validation, expected success")
	}

	// 4. TOCTOU / Fuzzing Test : Cas d'échec
	// On simule un pair malveillant qui envoie un bloc altéré
	corruptedChunk := make([]byte, len(chunk0Data))
	copy(corruptedChunk, chunk0Data)
	
	// Fuzzing : on change un seul octet
	corruptedChunk[10] = ^corruptedChunk[10]

	if ValidateChunk(corruptedChunk, manifest.ChunkHashes[0]) {
		t.Errorf("🚨 CRITICAL SECURITY FAILURE: ValidateChunk accepted corrupted data!")
	} else {
		t.Log("✅ Zero-Trust Check Passed: Corrupted chunk was strictly rejected.")
	}
}
