package onion

import (
	"crypto/ed25519"
	"testing"
)

// Cellules et poignées de main reçues d'inconnus : jamais de panique, et
// aucun chiffré forgé ne s'ouvre.
func FuzzOnion(f *testing.F) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	c, _ := NewClient(pub)
	f.Add(c.Message())
	f.Add([]byte{})
	f.Add(make([]byte, 600))
	_, hop, _ := Respond(priv, c.Message())
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = Unpad(b)
		_, _, _ = Respond(priv, b)
		if hop != nil {
			if _, err := hop.OpenForward(b); err == nil {
				t.Fatal("cellule forgée acceptée")
			}
			if _, err := hop.OpenBackward(b); err == nil {
				t.Fatal("cellule retour forgée acceptée")
			}
		}
		if p, ok := Pad(b); ok && len(b) > 0 { // commande vide : refusée exprès par Unpad
			if q, ok := Unpad(p); !ok || string(q) != string(b) {
				t.Fatal("bourrage non réversible")
			}
		}
	})
}
