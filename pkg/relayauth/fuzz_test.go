package relayauth

import (
	"crypto/ed25519"
	"testing"
	"time"
)

// En-tête d'authentification des relais auprès de l'origine : aucune valeur
// forgée n'est acceptée, aucune ne fait paniquer.
func FuzzVerify(f *testing.F) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	f.Add(Sign(priv, "POST", "/v1/load", []byte("{}")), "POST", "/v1/load", []byte("{}"))
	f.Add("", "GET", "/", []byte(nil))
	f.Add("k=;t=;n=;s=", "GET", "/v1/catalog", []byte(nil))
	v := &Verifier{Keys: []ed25519.PublicKey{pub}}
	f.Fuzz(func(t *testing.T, h, method, path string, body []byte) {
		if v.Verify(h, method, path, body, time.Now()) == nil {
			// seule une signature réelle passe : elle doit couvrir la requête exacte
			if v.Verify(h, method, path+"x", body, time.Now()) == nil {
				t.Fatal("signature acceptée pour un autre chemin")
			}
		}
	})
}
