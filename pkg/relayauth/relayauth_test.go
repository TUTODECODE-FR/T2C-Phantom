package relayauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	v := &Verifier{Keys: []ed25519.PublicKey{pub}}
	now := time.Now()

	h := Sign(priv, "POST", "/v1/volunteer", []byte(`{"a":1}`))
	if err := v.Verify(h, "POST", "/v1/volunteer", []byte(`{"a":1}`), now); err != nil {
		t.Fatalf("requête valide refusée: %v", err)
	}
	if v.Verify(h, "POST", "/v1/volunteer", []byte(`{"a":1}`), now) == nil {
		t.Fatal("rejeu accepté")
	}
	h = Sign(priv, "POST", "/v1/volunteer", []byte(`{"a":1}`))
	if v.Verify(h, "POST", "/v1/volunteer", []byte(`{"a":2}`), now) == nil {
		t.Fatal("corps modifié accepté")
	}
	if v.Verify(Sign(priv, "GET", "/v1/catalog", nil), "GET", "/v1/hosts", nil, now) == nil {
		t.Fatal("chemin modifié accepté")
	}
	if v.Verify(Sign(other, "GET", "/v1/catalog", nil), "GET", "/v1/catalog", nil, now) == nil {
		t.Fatal("clé inconnue acceptée")
	}
	if v.Verify(Sign(priv, "GET", "/v1/catalog", nil), "GET", "/v1/catalog", nil, now.Add(5*time.Minute)) == nil {
		t.Fatal("requête périmée acceptée")
	}
	for _, bad := range []string{"", "x", "1.2.3", "a.b.c.d"} {
		if v.Verify(bad, "GET", "/", nil, now) == nil {
			t.Fatalf("en-tête %q accepté", bad)
		}
	}
}
