package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRelayCandidates(t *testing.T) {
	dir := t.TempDir()
	n := &Node{cfg: Config{DataDir: dir, RelayURL: DefaultRelay}}
	learned := []string{"wss://r1.example/v1/ws", "wss://r2.example/v1/ws", "http://pas-un-relais", "wss://r3.example/v1/ws"}
	b, _ := json.Marshal(learned)
	_ = os.WriteFile(filepath.Join(dir, "relays.json"), b, 0o600)

	firsts := map[string]bool{}
	for i := 0; i < 60; i++ {
		c := n.relayCandidates()
		if slices.Contains(c, "http://pas-un-relais") {
			t.Fatal("adresse non wss acceptée")
		}
		for _, d := range DefaultRelays {
			if !slices.Contains(c, d) {
				t.Fatalf("relais par défaut absent: %s", d)
			}
		}
		firsts[c[0]] = true
	}
	if len(firsts) < 2 {
		t.Fatal("les utilisateurs ne se répartissent pas entre les relais")
	}
	// relais choisi par l'utilisateur : toujours en premier
	n.cfg.RelayURL = "wss://perso.example/v1/ws"
	if c := n.relayCandidates(); c[0] != "wss://perso.example/v1/ws" {
		t.Fatalf("relais choisi pas en premier: %v", c)
	}
}
