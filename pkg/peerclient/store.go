package peerclient

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"t2c/pkg/trust"
)

// ContentStore : copies hébergées par ce pair (bénévole), vérifiées à l'écriture.
//
//	<dir>/<root>.blob  contenu
//	<dir>/<root>.json  manifeste signé
type ContentStore struct {
	Dir string
	mu  sync.RWMutex
	man map[string]*trust.Manifest
}

func NewContentStore(dir string) (*ContentStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &ContentStore{Dir: dir, man: map[string]*trust.Manifest{}}, nil
}

// Put enregistre un contenu complet dont chaque morceau a été vérifié.
func (s *ContentStore) Put(m *trust.Manifest, data []byte) error {
	if int64(len(data)) != m.Size {
		return errors.New("taille incohérente")
	}
	for i := range m.Chunks {
		off := i * m.ChunkSize
		if !m.VerifyChunk(i, data[off:off+m.ChunkLen(i)]) {
			return errors.New("contenu non conforme au manifeste")
		}
	}
	mb, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(s.Dir, m.Root+".json"), mb, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.Dir, m.Root+".blob"), data, 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	s.man[m.Root] = m
	s.mu.Unlock()
	return nil
}

// Load relit un manifeste stocké (au redémarrage), signature revérifiée.
func (s *ContentStore) Load(root string, rootKey []byte) (*trust.Manifest, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, root+".json"))
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

// Chunk lit le morceau i (relu depuis le disque à chaque fois).
func (s *ContentStore) Chunk(root string, i int) ([]byte, error) {
	m := s.Manifest(root)
	if m == nil || i < 0 || i >= len(m.Chunks) {
		return nil, errors.New("non hébergé")
	}
	f, err := os.Open(filepath.Join(s.Dir, root+".blob"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, m.ChunkLen(i))
	if _, err := f.ReadAt(buf, int64(i)*int64(m.ChunkSize)); err != nil {
		return nil, err
	}
	return buf, nil
}

// Read renvoie le contenu complet.
func (s *ContentStore) Read(root string) ([]byte, error) {
	if s.Manifest(root) == nil {
		return nil, errors.New("non hébergé")
	}
	return os.ReadFile(filepath.Join(s.Dir, root+".blob"))
}
