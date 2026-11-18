package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"
)

func keys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func TestCatalogSignVerifyTamper(t *testing.T) {
	pub, priv := keys(t)
	root := bytes.Repeat([]byte("a"), 64)
	c := &Catalog{Version: 3, Sites: []SiteEntry{{ID: "s", Alias: "s", Root: string(root)}}}
	if err := c.Sign(priv); err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(pub); err != nil {
		t.Fatalf("signature valide refusée: %v", err)
	}
	// altération du document reçu (Sites après Verify est une vue dérivée de la base signée)
	tc := reload(t, c)
	tc.Sites[0].Root = string(bytes.Repeat([]byte("b"), 64))
	if tc.Verify(pub) == nil {
		t.Fatal("catalogue modifié accepté")
	}
	other, _ := keys(t)
	if reload(t, c).Verify(other) == nil {
		t.Fatal("mauvaise clé acceptée")
	}
}

func TestDomainSeparation(t *testing.T) {
	pub, priv := keys(t)
	tk := &Ticket{Peer: "p", Site: "s", Root: "r", Expires: 1}
	_ = tk.Sign(priv)
	h := &HostsList{Sig: tk.Sig}
	if h.Verify(pub) == nil {
		t.Fatal("signature d'un type rejouée sur un autre")
	}
}

func TestManifestChunks(t *testing.T) {
	pub, priv := keys(t)
	data := make([]byte, ChunkSize*2+123)
	_, _ = rand.Read(data)
	m, err := BuildManifest("s", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Chunks) != 3 || m.ChunkLen(2) != 123 {
		t.Fatalf("découpage faux: %d morceaux, dernier %d", len(m.Chunks), m.ChunkLen(2))
	}
	_ = m.Sign(priv)
	if err := m.Verify(pub); err != nil {
		t.Fatal(err)
	}
	if !m.VerifyChunk(1, data[ChunkSize:2*ChunkSize]) {
		t.Fatal("morceau valide refusé")
	}
	bad := append([]byte{}, data[:ChunkSize]...)
	bad[10] ^= 1
	if m.VerifyChunk(0, bad) {
		t.Fatal("morceau modifié accepté")
	}
	if m.VerifyChunk(2, data[2*ChunkSize:]) == false || m.VerifyChunk(2, data[2*ChunkSize:len(data)-1]) {
		t.Fatal("contrôle de taille du dernier morceau")
	}
	m.Chunks[0], m.Chunks[1] = m.Chunks[1], m.Chunks[0]
	if m.CheckStructure() == nil {
		t.Fatal("racine incohérente acceptée")
	}
}

func TestVersionGuard(t *testing.T) {
	p := filepath.Join(t.TempDir(), "g.json")
	g := NewVersionGuard(p)
	if g.Accept("catalog", 5) != nil || g.Accept("catalog", 5) != nil {
		t.Fatal("version égale/nouvelle refusée")
	}
	if g.Accept("catalog", 4) == nil {
		t.Fatal("rollback accepté")
	}
	if NewVersionGuard(p).Accept("catalog", 4) == nil {
		t.Fatal("garde non persistée")
	}
}

func TestKeyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	pub, err := GenerateKey(p)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := LoadPrivateKey(p)
	if err != nil || !bytes.Equal(priv.Public().(ed25519.PublicKey), pub) {
		t.Fatal("rechargement de clé")
	}
	if _, err := GenerateKey(p); err == nil {
		t.Fatal("écrasement de clé autorisé")
	}
	tk := &Ticket{Expires: time.Now().Add(time.Hour).Unix()}
	if !tk.Valid(time.Now()) {
		t.Fatal("ticket valide")
	}
}

func TestDelegation(t *testing.T) {
	rootPub, rootPriv := keys(t)
	svcPub, svcPriv := keys(t)
	_, otherPriv := keys(t)

	d, err := NewDelegation(rootPriv, svcPub, time.Hour, ScopeHosts, ScopeTicket)
	if err != nil {
		t.Fatal(err)
	}
	tk := &Ticket{Peer: "p", Site: "s", Root: "r", Expires: time.Now().Add(time.Hour).Unix()}
	if err := tk.SignDelegated(svcPriv, d); err != nil {
		t.Fatal(err)
	}
	if err := tk.Verify(rootPub); err != nil {
		t.Fatalf("ticket délégué refusé: %v", err)
	}
	h := &HostsList{Version: 1}
	if err := h.SignDelegated(svcPriv, d); err != nil || h.Verify(rootPub) != nil {
		t.Fatal("hosts délégué refusé")
	}

	// clé non déléguée qui se fait passer pour le service
	forged := *tk
	_ = forged.SignDelegated(otherPriv, d)
	if forged.Verify(rootPub) == nil {
		t.Fatal("signature par une clé non déléguée acceptée")
	}
	// délégation auto-signée par un attaquant
	fake, _ := NewDelegation(otherPriv, svcPub, time.Hour, ScopeHosts, ScopeTicket)
	tk2 := *tk
	_ = tk2.SignDelegated(svcPriv, fake)
	if tk2.Verify(rootPub) == nil {
		t.Fatal("délégation non signée par la racine acceptée")
	}
	// portée : délégation « ticket » seule ne signe pas hosts
	dt, _ := NewDelegation(rootPriv, svcPub, time.Hour, ScopeTicket)
	h2 := &HostsList{Version: 2}
	_ = h2.SignDelegated(svcPriv, dt)
	if h2.Verify(rootPub) == nil {
		t.Fatal("portée de délégation ignorée")
	}
	// expiration
	exp := *d
	exp.Expires = time.Now().Add(-time.Second).Unix()
	if _, err := exp.ServiceKey(rootPub, ScopeHosts, time.Now()); err == nil {
		t.Fatal("délégation expirée acceptée")
	}
	// durée excessive refusée
	if _, err := NewDelegation(rootPriv, svcPub, 365*24*time.Hour, ScopeHosts); err == nil {
		t.Fatal("délégation d'un an acceptée")
	}
}
