package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// VersionGuard refuse les retours en arrière : un attaquant (ou un pair) qui
// rejoue un vieux catalogue signé — valide mais périmé — est rejeté.
// Persisté sur disque pour survivre aux redémarrages.
type VersionGuard struct {
	mu   sync.Mutex
	path string
	seen map[string]uint64
}

func NewVersionGuard(path string) *VersionGuard {
	g := &VersionGuard{path: path, seen: map[string]uint64{}}
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(b, &g.seen)
		}
	}
	return g
}

// Accept : ok si version ≥ dernière vue pour ce type ; mémorise la nouvelle.
func (g *VersionGuard) Accept(kind string, version uint64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if last := g.seen[kind]; version < last {
		return fmt.Errorf("trust: %s v%d plus ancien que v%d (rollback refusé)", kind, version, last)
	}
	if version == g.seen[kind] {
		return nil
	}
	g.seen[kind] = version
	if g.path == "" {
		return nil
	}
	b, _ := json.Marshal(g.seen)
	if err := os.MkdirAll(filepath.Dir(g.path), 0o700); err != nil {
		return err
	}
	tmp := g.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, g.path)
}
