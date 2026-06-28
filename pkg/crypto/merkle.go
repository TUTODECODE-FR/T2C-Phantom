package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const ChunkSize = 256 * 1024 // 256 Ko

// La clé publique "Air-Gapped" ancrée en dur dans le client.
// Seul le détenteur de la clé privée associée (physiquement isolée) peut publier un nouveau cours.
var MasterOfflinePubKeyHex = "e461a657c4f4544f8087fc7868f7b764c673d317cd7008103c80a0684a0deed7" 

// Manifest (clef.json) structure
type Manifest struct {
	RootHash   string   `json:"root_hash"`
	ChunkHashes []string `json:"chunk_hashes"`
	TotalChunks int      `json:"total_chunks"`
	FileSize   int64    `json:"file_size"`
	Signature  string   `json:"signature"` // Ed25519 Offline Signature
}

// VerifyManifestSignature garantit que le manifeste provient bien de l'autorité hors-ligne TUTODECODE
func VerifyManifestSignature(m *Manifest) bool {
	pubKeyBytes, err := hex.DecodeString(MasterOfflinePubKeyHex)
	if err != nil || len(pubKeyBytes) != 32 { // ed25519.PublicKeySize == 32
		return false
	}
	
	sigBytes, err := hex.DecodeString(m.Signature)
	if err != nil || len(sigBytes) != 64 { // ed25519.SignatureSize == 64
		return false
	}

	// Le message signé est la concaténation du RootHash et du FileSize
	message := []byte(m.RootHash + fmt.Sprintf(":%d", m.FileSize))
	return ed25519.Verify(pubKeyBytes, message, sigBytes)
}

// GenerateManifest découpe un fichier et génère son manifeste (Merkle/Chunks)
func GenerateManifest(filePath string) (*Manifest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, _ := file.Stat()
	manifest := &Manifest{
		FileSize: stat.Size(),
	}

	h := sha256.New()
	buffer := make([]byte, ChunkSize)
	var chunkHashes []string

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if bytesRead == 0 {
			break
		}

		chunkHash := sha256.Sum256(buffer[:bytesRead])
		hashStr := hex.EncodeToString(chunkHash[:])
		chunkHashes = append(chunkHashes, hashStr)
		
		// Update root hash
		h.Write(chunkHash[:])
	}

	manifest.ChunkHashes = chunkHashes
	manifest.TotalChunks = len(chunkHashes)
	manifest.RootHash = hex.EncodeToString(h.Sum(nil))

	return manifest, nil
}

// ValidateChunk verifie le hash d'un bloc en RAM (Zero-Trust TOCTOU protection)
func ValidateChunk(data []byte, expectedHash string) bool {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]) == expectedHash
}
