package peerclient

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"t2c/pkg/trust"
)

// ContentStore : copies hébergées par ce pair (bénévole).
//
// Sur le disque, chaque morceau est chiffré (AES-256-GCM, clé locale aléatoire) :
// le bénévole ne stocke aucun fichier lisible ou modifiable à la main. À chaque
// lecture, le morceau est déchiffré puis revérifié contre le manifeste signé :
// une copie altérée sur le disque n'est jamais servie.
//
//	<dir>/.key         clé locale (0600)
//	<dir>/<root>.json  manifeste signé
//	<dir>/<root>.enc   morceaux chiffrés
type ContentStore struct {
	Dir  string
	mu   sync.RWMutex
	man  map[string]*trust.Manifest
	aead cipher.AEAD
}

var (
	ErrNotHosted = errors.New("non hébergé")
	ErrCorrupt   = errors.New("copie locale altérée")
)

func NewContentStore(dir string) (*ContentStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	key, err := os.ReadFile(filepath.Join(dir, ".key"))
	if err != nil || len(key) != 32 {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, ".key"), key, 0o600); err != nil {
			return nil, err
		}
		// ancienne clé perdue ou absente : les copies chiffrées existantes sont illisibles
		matches, _ := filepath.Glob(filepath.Join(dir, "*.enc"))
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &ContentStore{Dir: dir, man: map[string]*trust.Manifest{}, aead: aead}, nil
}

func (s *ContentStore) path(root, ext string) string { return filepath.Join(s.Dir, root+ext) }

// nonce unique par (racine, index) ; le clair d'un couple est fixé par son empreinte.
func (s *ContentStore) nonce(root string, i int) ([]byte, []byte) {
	ad := make([]byte, 0, len(root)+4)
	ad = append(ad, root...)
	ad = binary.BigEndian.AppendUint32(ad, uint32(i))
	h := sha256.Sum256(ad)
	return h[:s.aead.NonceSize()], ad
}

func (s *ContentStore) slot(m *trust.Manifest) int64 { return int64(m.ChunkSize + s.aead.Overhead()) }

// Put enregistre un contenu complet dont chaque morceau a été vérifié.
func (s *ContentStore) Put(m *trust.Manifest, data []byte) error {
	if int64(len(data)) != m.Size {
		return errors.New("taille incohérente")
	}
	out := make([]byte, 0, int64(len(m.Chunks))*s.slot(m))
	for i := range m.Chunks {
		off := i * m.ChunkSize
		c := data[off : off+m.ChunkLen(i)]
		if !m.VerifyChunk(i, c) {
			return errors.New("contenu non conforme au manifeste")
		}
		n, ad := s.nonce(m.Root, i)
		out = s.aead.Seal(out, n, c, ad) // tous les morceaux sauf le dernier remplissent exactement un emplacement
	}
	if err := os.WriteFile(s.path(m.Root, ".enc.tmp"), out, 0o600); err != nil {
		return err
	}
	if err := os.Rename(s.path(m.Root, ".enc.tmp"), s.path(m.Root, ".enc")); err != nil {
		return err
	}
	mb, _ := json.Marshal(m)
	if err := os.WriteFile(s.path(m.Root, ".json"), mb, 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	s.man[m.Root] = m
	s.mu.Unlock()
	return nil
}

// Load relit un manifeste stocké (au redémarrage), signature revérifiée.
// Une ancienne copie en clair (.blob) est chiffrée puis supprimée.
func (s *ContentStore) Load(root string, rootKey []byte) (*trust.Manifest, error) {
	b, err := os.ReadFile(s.path(root, ".json"))
	if err != nil {
		return nil, err
	}
	var m trust.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if err := m.Verify(rootKey); err != nil {
		return nil, err
	}
	if _, err := os.Stat(s.path(root, ".enc")); err != nil {
		legacy, lerr := os.ReadFile(s.path(root, ".blob"))
		if lerr != nil {
			return nil, ErrNotHosted
		}
		if err := s.Put(&m, legacy); err != nil {
			return nil, err
		}
		_ = os.Remove(s.path(root, ".blob"))
	}
	s.mu.Lock()
	s.man[root] = &m
	s.mu.Unlock()
	return &m, nil
}

func (s *ContentStore) Manifest(root string) *trust.Manifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.man[root]
}

// Chunk lit, déchiffre et revérifie le morceau i.
func (s *ContentStore) Chunk(root string, i int) ([]byte, error) {
	m := s.Manifest(root)
	if m == nil || i < 0 || i >= len(m.Chunks) {
		return nil, ErrNotHosted
	}
	f, err := os.Open(s.path(root, ".enc"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, m.ChunkLen(i)+s.aead.Overhead())
	if _, err := f.ReadAt(buf, int64(i)*s.slot(m)); err != nil {
		return nil, err
	}
	n, ad := s.nonce(root, i)
	c, err := s.aead.Open(buf[:0], n, buf, ad)
	if err != nil || !m.VerifyChunk(i, c) {
		return nil, ErrCorrupt
	}
	return c, nil
}

// Read renvoie le contenu complet (déchiffré et vérifié).
func (s *ContentStore) Read(root string) ([]byte, error) {
	m := s.Manifest(root)
	if m == nil {
		return nil, ErrNotHosted
	}
	out := make([]byte, 0, m.Size)
	for i := range m.Chunks {
		c, err := s.Chunk(root, i)
		if err != nil {
			return nil, err
		}
		out = append(out, c...)
	}
	return out, nil
}

// Remove supprime une copie hébergée.
func (s *ContentStore) Remove(root string) {
	s.mu.Lock()
	delete(s.man, root)
	s.mu.Unlock()
	for _, ext := range []string{".enc", ".json", ".blob"} {
		_ = os.Remove(s.path(root, ext))
	}
}

// Roots : racines des copies chargées.
func (s *ContentStore) Roots() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.man))
	for r := range s.man {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// Size : octets occupés par les copies chargées.
func (s *ContentStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int64
	for _, m := range s.man {
		n += m.Size
	}
	return n
}
