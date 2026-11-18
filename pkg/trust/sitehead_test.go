package trust

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func headFixture(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, ed25519.PrivateKey, *Delegation, *Catalog) {
	t.Helper()
	rootPub, rootPriv, _ := ed25519.GenerateKey(nil)
	_, svc, _ := ed25519.GenerateKey(nil)
	d, err := NewDelegation(rootPriv, svc.Public().(ed25519.PublicKey), time.Hour, ScopeSite("repo"))
	if err != nil {
		t.Fatal(err)
	}
	cat := &Catalog{Version: 3, Issued: 1, Sites: []SiteEntry{
		{ID: "repo", Alias: "repo", Root: strings.Repeat("a", 64), Size: 10},
		{ID: "tutodecode", Alias: "tutodecode", Root: strings.Repeat("b", 64), Size: 10},
	}}
	if err := cat.Sign(rootPriv); err != nil {
		t.Fatal(err)
	}
	return rootPub, rootPriv, svc, d, cat
}

func signedHead(t *testing.T, svc ed25519.PrivateKey, d *Delegation, site, root string, issued int64) SiteHead {
	t.Helper()
	h := SiteHead{Site: site, Root: root, Size: 42, Issued: issued}
	if err := h.SignDelegated(svc, d); err != nil {
		t.Fatal(err)
	}
	return h
}

func reload(t *testing.T, c *Catalog) *Catalog {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var out Catalog
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return &out
}

func TestSiteHeadApplied(t *testing.T) {
	root, _, svc, d, cat := headFixture(t)
	newRoot := strings.Repeat("c", 64)
	cat.Heads = []SiteHead{
		signedHead(t, svc, d, "repo", strings.Repeat("d", 64), 100),
		signedHead(t, svc, d, "repo", newRoot, 200), // la plus récente gagne
	}
	c := reload(t, cat)
	if err := c.Verify(root); err != nil {
		t.Fatal(err)
	}
	e, _ := c.Find("repo")
	if e.Root != newRoot || e.Size != 42 || c.HeadIssued("repo") != 200 {
		t.Fatalf("tête non appliquée: %+v", e)
	}
	if e, _ := c.Find("tutodecode"); e.Root != strings.Repeat("b", 64) {
		t.Fatal("autre site modifié")
	}
	// relu après application : même résultat, signature toujours valide
	c2 := reload(t, c)
	if err := c2.Verify(root); err != nil {
		t.Fatal(err)
	}
	if e, _ := c2.Find("repo"); e.Root != newRoot {
		t.Fatal("aller-retour JSON perd la tête")
	}
	if err := c.Verify(root); err != nil { // idempotent
		t.Fatal(err)
	}
}

func TestSiteHeadRejected(t *testing.T) {
	root, rootPriv, svc, d, cat := headFixture(t)
	_, other, _ := ed25519.GenerateKey(nil)
	expired := *d
	expired.Expires = time.Now().Add(-time.Minute).Unix()
	bad := []SiteHead{
		signedHead(t, svc, d, "tutodecode", strings.Repeat("c", 64), 300), // site non confié
		signedHead(t, other, d, "repo", strings.Repeat("c", 64), 300),     // mauvaise clé
		signedHead(t, svc, d, "inconnu", strings.Repeat("c", 64), 300),    // site absent du catalogue
		signedHead(t, svc, &expired, "repo", strings.Repeat("c", 64), 300),
		signedHead(t, rootPriv, nil, "repo", strings.Repeat("c", 64), 300), // sans délégation
	}
	tampered := signedHead(t, svc, d, "repo", strings.Repeat("c", 64), 300)
	tampered.Root = strings.Repeat("e", 64)
	bad = append(bad, tampered)
	for i, h := range bad {
		c := reload(t, cat)
		c.Heads = []SiteHead{h}
		if err := c.Verify(root); err != nil {
			t.Fatalf("%d: une tête invalide ne doit pas invalider le catalogue: %v", i, err)
		}
		for _, e := range c.Sites {
			if e.Root == strings.Repeat("c", 64) || e.Root == strings.Repeat("e", 64) {
				t.Fatalf("%d: tête invalide appliquée", i)
			}
		}
		if len(c.Heads) != 0 {
			t.Fatalf("%d: tête invalide conservée", i)
		}
	}
}

func TestDelegatedManifest(t *testing.T) {
	root, _, svc, d, _ := headFixture(t)
	m, err := BuildManifest("repo", bytes.NewReader([]byte("contenu")))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SignDelegated(svc, d); err != nil {
		t.Fatal(err)
	}
	if err := m.Verify(root); err != nil {
		t.Fatal(err)
	}
	m2, _ := BuildManifest("tutodecode", bytes.NewReader([]byte("contenu")))
	if err := m2.SignDelegated(svc, d); err != nil {
		t.Fatal(err)
	}
	if m2.Verify(root) == nil {
		t.Fatal("manifeste d'un site non confié accepté")
	}
}
