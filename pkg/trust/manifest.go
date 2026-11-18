package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// ChunkSize : taille des morceaux échangés entre pairs (identique à pkg/crypto).
const ChunkSize = 256 * 1024

// MaxContentSize : taille max d'un site publié (borne mémoire / anti-abus).
const MaxContentSize = 256 << 20

// Manifest décrit un contenu (le zip d'un site) découpé en morceaux.
// Root = sha256(hash0 ‖ hash1 ‖ …) : c'est l'adresse du contenu dans la DHT.
type Manifest struct {
	Site      string   `json:"site"`
	Root      string   `json:"root"`
	Size      int64    `json:"size"`
	ChunkSize int      `json:"chunk_size"`
	Chunks    []string `json:"chunks"`
	// Deleg : clé de portée site:<Site> qui a signé (absente = clé racine).
	Deleg *Delegation `json:"deleg,omitempty"`
	Sig   string      `json:"sig"`
}

// BuildManifest lit r en entier (≤ MaxContentSize) et calcule le manifeste.
func BuildManifest(site string, r io.Reader) (*Manifest, error) {
	m := &Manifest{Site: site, ChunkSize: ChunkSize}
	rootH := sha256.New()
	buf := make([]byte, ChunkSize)
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			m.Size += int64(n)
			if m.Size > MaxContentSize {
				return nil, fmt.Errorf("trust: contenu > %d octets", MaxContentSize)
			}
			sum := sha256.Sum256(buf[:n])
			rootH.Write(sum[:])
			m.Chunks = append(m.Chunks, hex.EncodeToString(sum[:]))
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if len(m.Chunks) == 0 {
		return nil, fmt.Errorf("trust: contenu vide")
	}
	m.Root = hex.EncodeToString(rootH.Sum(nil))
	return m, nil
}

// CheckStructure vérifie la cohérence interne (racine = hash des hashes, tailles).
func (m *Manifest) CheckStructure() error {
	if m.ChunkSize != ChunkSize || m.Size <= 0 || m.Size > MaxContentSize {
		return fmt.Errorf("trust: manifeste: tailles invalides")
	}
	want := int((m.Size + int64(m.ChunkSize) - 1) / int64(m.ChunkSize))
	if len(m.Chunks) != want {
		return fmt.Errorf("trust: manifeste: %d morceaux, attendu %d", len(m.Chunks), want)
	}
	h := sha256.New()
	for _, c := range m.Chunks {
		b, err := hex.DecodeString(c)
		if err != nil || len(b) != sha256.Size {
			return fmt.Errorf("trust: manifeste: hash de morceau invalide")
		}
		h.Write(b)
	}
	if hex.EncodeToString(h.Sum(nil)) != m.Root {
		return fmt.Errorf("trust: manifeste: racine incohérente")
	}
	return nil
}

// ChunkLen : taille attendue du morceau i.
func (m *Manifest) ChunkLen(i int) int {
	if i == len(m.Chunks)-1 {
		if r := int(m.Size % int64(m.ChunkSize)); r != 0 {
			return r
		}
	}
	return m.ChunkSize
}

// VerifyChunk : le morceau i a la bonne taille et le bon hash.
func (m *Manifest) VerifyChunk(i int, data []byte) bool {
	if i < 0 || i >= len(m.Chunks) || len(data) != m.ChunkLen(i) {
		return false
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) == m.Chunks[i]
}

func (m *Manifest) Sign(priv ed25519.PrivateKey) error {
	u := *m
	u.Sig = ""
	s, err := sign(priv, domManifest, u)
	m.Sig = s
	return err
}

// Verify : signature de l'origine + structure interne.
func (m *Manifest) Verify(root ed25519.PublicKey) error {
	k, err := signerKey(root, m.Deleg, ScopeSite(m.Site))
	if err != nil {
		return err
	}
	u := *m
	u.Sig = ""
	if err := verify(k, domManifest, u, m.Sig); err != nil {
		return err
	}
	return m.CheckStructure()
}

// AuditHash : réponse à un défi de stockage — prouve qu'on détient le morceau
// sans le renvoyer en entier (le nonce empêche de précalculer).
func AuditHash(nonce []byte, index int, chunk []byte) []byte {
	h := sha256.New()
	h.Write([]byte("t2c/audit/v1\n"))
	h.Write(nonce)
	var ib [8]byte
	for k := 0; k < 8; k++ {
		ib[7-k] = byte(uint64(index) >> (8 * k))
	}
	h.Write(ib[:])
	h.Write(chunk)
	return h.Sum(nil)
}
