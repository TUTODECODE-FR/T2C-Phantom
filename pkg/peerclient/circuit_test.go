package peerclient

import (
	"errors"
	"testing"

	"t2c/pkg/wire"
)

// Échec fermé : en mode anonyme sans circuit, rien ne part (ni demande ni échange).
func TestAnonymousFailsClosed(t *testing.T) {
	c := &Client{opt: Options{Anonymous: true}, building: map[uint64]*circuit{}}
	if err := c.sendData(&wire.Frame{T: wire.TGet}); !errors.Is(err, ErrNoCircuit) {
		t.Fatalf("demande envoyée sans circuit: %v", err)
	}
	if _, err := c.e2eIdentity(); !errors.Is(err, ErrNoCircuit) {
		t.Fatal("identité réelle présentée sans circuit")
	}
	dead := &circuit{dead: make(chan struct{})}
	dead.kill()
	c.circ = dead
	if err := c.sendData(&wire.Frame{T: wire.TFwd}); !errors.Is(err, ErrNoCircuit) {
		t.Fatal("échange envoyé par un circuit mort")
	}
	if c.Anonymous() {
		t.Fatal("anonymat annoncé sans circuit actif")
	}
}
