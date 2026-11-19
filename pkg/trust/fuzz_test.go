package trust

import (
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"
)

// Documents signés reçus des relais : aucune entrée forgée ne doit passer la
// vérification sans la clé racine, ni faire paniquer l'analyse.
var fuzzRoot, fuzzRootPriv, _ = ed25519.GenerateKey(nil)

func fuzzSeeds(f *testing.F) {
	now := time.Now()
	cat := &Catalog{Version: 1, Issued: now.Unix(), Sites: []SiteEntry{{ID: "demo", Root: "00"}}}
	_ = cat.Sign(fuzzRootPriv)
	rl := &RelayList{Version: 1, Issued: now.Unix(), Relays: []string{"wss://a/v1/ws#k=00"}}
	_ = rl.Sign(fuzzRootPriv)
	p := &NetworkPolicy{Version: 1, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix(), Emergency: true}
	_ = p.SignDelegated(fuzzRootPriv, nil)
	cat.Relays, cat.Policy = rl, p
	for _, v := range []any{cat, rl, p} {
		b, _ := json.Marshal(v)
		f.Add(b)
	}
	f.Add([]byte(`{"version":1,"relays":{"version":1,"relays":["hidden:x"],"revoked":["zz"]},"policy":{"deleg":{"scopes":["policy"]}}}`))
}

func FuzzCatalogVerify(f *testing.F) {
	fuzzSeeds(f)
	_, other, _ := ed25519.GenerateKey(nil)
	f.Fuzz(func(t *testing.T, b []byte) {
		var c Catalog
		if json.Unmarshal(b, &c) != nil {
			return
		}
		// avec une racine dont l'attaquant n'a pas la clé : rien n'est accepté
		// (sauf un document réellement signé par fuzzRoot, présent dans les graines)
		if c.Verify(other.Public().(ed25519.PublicKey)) == nil {
			t.Fatal("catalogue accepté sous une racine inconnue")
		}
		_ = c.Verify(fuzzRoot)
	})
}

func FuzzPolicyVerify(f *testing.F) {
	fuzzSeeds(f)
	_, other, _ := ed25519.GenerateKey(nil)
	f.Fuzz(func(t *testing.T, b []byte) {
		var p NetworkPolicy
		if json.Unmarshal(b, &p) != nil {
			return
		}
		if p.Verify(other.Public().(ed25519.PublicKey), time.Now()) == nil {
			t.Fatal("consignes acceptées sous une racine inconnue")
		}
		if p.Verify(fuzzRoot, time.Now()) == nil {
			// une consigne valide ne peut jamais desserrer
			if p.MinBits(24) < 24 || p.RelayKBs(128) > 128 || p.VolunteerShare() > 100 {
				t.Fatal("consigne valide qui desserre")
			}
		}
	})
}

func FuzzRelayListVerify(f *testing.F) {
	fuzzSeeds(f)
	_, other, _ := ed25519.GenerateKey(nil)
	f.Fuzz(func(t *testing.T, b []byte) {
		var r RelayList
		if json.Unmarshal(b, &r) != nil {
			return
		}
		if r.Verify(other.Public().(ed25519.PublicKey)) == nil {
			t.Fatal("liste de relais acceptée sous une racine inconnue")
		}
	})
}
