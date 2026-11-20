package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testRelease() *RelayRelease {
	return &RelayRelease{Version: "2.6.0", Issued: time.Now().Unix(), Site: "relais", Percent: 100,
		Binaries: map[string]ReleaseBinary{"linux/amd64": {Path: "linux-amd64/t2c-relay", SHA256: strings.Repeat("ab", 32), Size: 1234}}}
}

func TestRelayReleaseDelegated(t *testing.T) {
	root, rootPriv, _ := ed25519.GenerateKey(nil)
	upd, updPriv, _ := ed25519.GenerateKey(nil)
	now := time.Now()
	d, err := NewDelegation(rootPriv, upd, 30*24*time.Hour, ScopeRelayUpdate)
	if err != nil {
		t.Fatal(err)
	}
	r := testRelease()
	if err := r.SignDelegated(updPriv, d); err != nil {
		t.Fatal(err)
	}
	if err := r.Verify(root, now); err != nil {
		t.Fatal(err)
	}
	if r.Signer() != d.Key {
		t.Fatal("signataire")
	}
	// la racine elle-même est acceptée
	rr := testRelease()
	_ = rr.SignDelegated(rootPriv, nil)
	if err := rr.Verify(root, now); err != nil || rr.Signer() != "root" {
		t.Fatal("release signée par la racine refusée", err)
	}
	// délégation expirée
	if r.Verify(root, now.Add(31*24*time.Hour)) == nil {
		t.Fatal("délégation expirée acceptée")
	}
	// autre racine
	other, _, _ := ed25519.GenerateKey(nil)
	if r.Verify(other, now) == nil {
		t.Fatal("acceptée sous une autre racine")
	}
	// clé de mise à jour sans délégation : refusée
	nd := testRelease()
	_ = nd.SignDelegated(updPriv, nil)
	if nd.Verify(root, now) == nil {
		t.Fatal("clé de mise à jour non déléguée acceptée")
	}
	// clé de service d'une AUTRE portée (consignes, hosts, site) : refusée
	for _, scope := range []string{ScopePolicy, ScopeHosts, ScopeTicket, ScopeSite("relais")} {
		svc, svcPriv, _ := ed25519.GenerateKey(nil)
		sd, _ := NewDelegation(rootPriv, svc, time.Hour, scope)
		x := testRelease()
		_ = x.SignDelegated(svcPriv, sd)
		if x.Verify(root, now) == nil {
			t.Fatalf("délégation %q acceptée pour une release", scope)
		}
	}
	// la clé de mise à jour ne signe rien d'autre (consignes réseau)
	p := &NetworkPolicy{Version: 1, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix()}
	_ = p.SignDelegated(updPriv, d)
	if p.Verify(root, now) == nil {
		t.Fatal("clé de mise à jour acceptée pour des consignes")
	}
	// altérations
	for _, tamper := range []func(*RelayRelease){
		func(x *RelayRelease) { x.Version = "2.7.0" },
		func(x *RelayRelease) { x.Percent = 50 },
		func(x *RelayRelease) { x.Site = "autre" },
		func(x *RelayRelease) { x.Notes = "x" },
		func(x *RelayRelease) {
			b := x.Binaries["linux/amd64"]
			b.SHA256 = strings.Repeat("cd", 32)
			x.Binaries["linux/amd64"] = b
		},
		func(x *RelayRelease) {
			x.Binaries["linux/arm64"] = ReleaseBinary{Path: "a", SHA256: strings.Repeat("ab", 32), Size: 1}
		},
	} {
		var x RelayRelease
		raw, _ := json.Marshal(r)
		_ = json.Unmarshal(raw, &x)
		tamper(&x)
		if x.Verify(root, now) == nil {
			t.Fatal("altération acceptée")
		}
	}
}

func TestRelayReleaseBounds(t *testing.T) {
	root, rootPriv, _ := ed25519.GenerateKey(nil)
	now := time.Now()
	for i, mod := range []func(*RelayRelease){
		func(x *RelayRelease) { x.Version = "dev" },
		func(x *RelayRelease) { x.Version = "2.6.0-rc1" },
		func(x *RelayRelease) { x.Percent = 0 },
		func(x *RelayRelease) { x.Percent = 101 },
		func(x *RelayRelease) { x.Site = "" },
		func(x *RelayRelease) { x.Site = "../x" },
		func(x *RelayRelease) { x.Issued = now.Add(time.Hour).Unix() },
		func(x *RelayRelease) { x.Binaries = nil },
		func(x *RelayRelease) { x.Notes = strings.Repeat("é", MaxReleaseNotes+1) },
		func(x *RelayRelease) { x.Binaries["windows/amd64"] = x.Binaries["linux/amd64"] },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Size = MaxReleaseBinary + 1 }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Size = 0 }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.SHA256 = "abcd" }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.SHA256 = strings.Repeat("AB", 32) }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Path = "../t2c-relay" }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Path = "/etc/passwd" }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Path = "a/./b" }) },
		func(x *RelayRelease) { setBin(x, func(b *ReleaseBinary) { b.Path = "" }) },
	} {
		x := testRelease()
		mod(x)
		_ = x.SignDelegated(rootPriv, nil)
		if x.Verify(root, now) == nil {
			t.Fatalf("cas %d accepté", i)
		}
	}
	// limites acceptées
	x := testRelease()
	setBin(x, func(b *ReleaseBinary) { b.Size = MaxReleaseBinary })
	x.Percent = 1
	_ = x.SignDelegated(rootPriv, nil)
	if err := x.Verify(root, now); err != nil {
		t.Fatal(err)
	}
}

func setBin(x *RelayRelease, f func(*ReleaseBinary)) {
	b := x.Binaries["linux/amd64"]
	f(&b)
	x.Binaries["linux/amd64"] = b
}

func TestRelayReleaseInCatalog(t *testing.T) {
	root, rootPriv, _ := ed25519.GenerateKey(nil)
	upd, updPriv, _ := ed25519.GenerateKey(nil)
	d, _ := NewDelegation(rootPriv, upd, time.Hour, ScopeRelayUpdate)
	cat := &Catalog{Version: 1, Issued: time.Now().Unix(), Sites: []SiteEntry{{ID: "relais", Root: strings.Repeat("00", 32)}}}
	_ = cat.Sign(rootPriv)
	r := testRelease()
	_ = r.SignDelegated(updPriv, d)
	cat.RelayRelease = r
	raw, _ := json.Marshal(cat)
	var c Catalog
	_ = json.Unmarshal(raw, &c)
	if err := c.Verify(root); err != nil || c.RelayRelease == nil {
		t.Fatal("release valide perdue", err)
	}
	// release altérée : catalogue toujours valide, release ignorée
	c2 := Catalog{}
	_ = json.Unmarshal(raw, &c2)
	c2.RelayRelease.Percent = 7
	if err := c2.Verify(root); err != nil || c2.RelayRelease != nil {
		t.Fatal("release altérée conservée", err)
	}
	// clé de mise à jour révoquée par la liste racine des relais
	rl := &RelayList{Version: 1, Issued: time.Now().Unix(), Relays: []string{"wss://a/v1/ws#k=00"}, Revoked: []string{d.Key}}
	_ = rl.Sign(rootPriv)
	c3 := Catalog{}
	_ = json.Unmarshal(raw, &c3)
	c3.Relays = rl
	if err := c3.Verify(root); err != nil || c3.RelayRelease != nil {
		t.Fatal("release d'une clé révoquée conservée", err)
	}
}

func TestRelayReleaseRollout(t *testing.T) {
	for _, pct := range []int{1, 10, 50, 90, 100} {
		r := testRelease()
		r.Percent = pct
		in := 0
		for i := 0; i < 1000; i++ {
			k, _, _ := ed25519.GenerateKey(rand.Reader)
			a := r.Includes(k)
			if a != r.Includes(k) { // déterministe
				t.Fatal("tirage instable")
			}
			if a {
				in++
			}
		}
		if lo, hi := pct*10-60, pct*10+60; in < lo || in > hi {
			t.Fatalf("%d %% : %d relais sur 1000", pct, in)
		}
	}
	// stable pour une version, indépendant d'une version à l'autre
	k, _, _ := ed25519.GenerateKey(rand.Reader)
	r := testRelease()
	r.Percent = 50
	same := 0
	for i := 0; i < 200; i++ {
		r.Version = "2.6." + itoaTest(i)
		x := r.Includes(k)
		r2 := *r
		if r2.Includes(k) != x {
			t.Fatal("instable")
		}
		if x {
			same++
		}
	}
	if same == 0 || same == 200 {
		t.Fatal("tirage indépendant de la version")
	}
}

func itoaTest(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
