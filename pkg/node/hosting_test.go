package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
			d, _, _ = strings.Cut(d, "#") // les candidats sont des URL sans clé
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

func TestHiddenRelayNeverDialed(t *testing.T) {
	dir := t.TempDir()
	n := &Node{cfg: Config{DataDir: dir}}
	key := "985c6d08ed885e08d69611f861595b7d4d6ae7a11a0cfc4c6a7f8267fa2d984f"
	learned := []string{"wss://vps.example/v1/ws#k=" + key, "hidden:maison#k=" + key[:62] + "aa"}
	b, _ := json.Marshal(learned)
	_ = os.WriteFile(filepath.Join(dir, "relays.json"), b, 0o600)
	for _, u := range n.relayCandidates() {
		if strings.HasPrefix(u, "hidden:") {
			t.Fatal("relais caché proposé à la connexion")
		}
	}
	if keys := n.relayKeys(); len(keys) != 2 {
		t.Fatalf("le relais caché doit servir aux circuits: %d clés", len(keys))
	}
}

// Entrées de la liste racine sans clé épinglée : ignorées.
func TestKeyedRelays(t *testing.T) {
	k := strings.Repeat("ab", 32)
	got := keyedRelays([]string{
		"wss://a.example/v1/ws#k=" + k,
		"wss://b.example/v1/ws",
		"hidden:c#k=" + k,
		"hidden:d",
		"http://e.example/v1/ws#k=" + k,
	})
	if !slices.Equal(got, []string{"wss://a.example/v1/ws#k=" + k, "hidden:c#k=" + k}) {
		t.Fatalf("%v", got)
	}
}
