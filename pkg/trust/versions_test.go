package trust

import (
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func signedNotice(t *testing.T, priv ed25519.PrivateKey, v *VersionNotice) *VersionNotice {
	t.Helper()
	if v.Version == 0 {
		v.Version = 1
	}
	v.Issued = time.Now().Unix()
	if err := v.Sign(priv); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestVersionNoticeSignVerify(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	_, other, _ := ed25519.GenerateKey(nil)
	v := signedNotice(t, priv, &VersionNotice{Latest: "2.6.0", Important: "2.5.1", Minimum: "2.5.0", Notes: "correctifs", URL: "https://github.com/x/y/releases/tag/v2.6.0"})
	if err := v.Verify(pub); err != nil {
		t.Fatal(err)
	}
	if v.Verify(other.Public().(ed25519.PublicKey)) == nil {
		t.Fatal("avis accepté sous une autre racine")
	}
	// altérations : chaque champ est couvert par la signature
	for _, tamper := range []func(*VersionNotice){
		func(x *VersionNotice) { x.Minimum = "2.4.0" },
		func(x *VersionNotice) { x.Latest = "2.7.0" },
		func(x *VersionNotice) { x.URL = "https://evil.example/" },
		func(x *VersionNotice) { x.Notes = "autre" },
		func(x *VersionNotice) { x.Version++ },
	} {
		x := *v
		tamper(&x)
		if x.Verify(pub) == nil {
			t.Fatalf("avis altéré accepté : %+v", x)
		}
	}
	// une clé de service déléguée ne peut pas signer l'avis : seule la racine
	_, svc, _ := ed25519.GenerateKey(nil)
	w := signedNotice(t, svc, &VersionNotice{Latest: "9.9.9"})
	if w.Verify(pub) == nil {
		t.Fatal("avis signé par une autre clé accepté")
	}
}

func TestVersionNoticeInvariants(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	pub := priv.Public().(ed25519.PublicKey)
	bad := []VersionNotice{
		{Latest: ""},
		{Latest: "2.5"},
		{Latest: "2.5.0", Minimum: "2.6.0"},
		{Latest: "2.5.0", Important: "2.6.0"},
		{Latest: "2.6.0", Important: "2.5.0", Minimum: "2.5.1"},
		{Latest: "2.5.0", Minimum: "x"},
		{Latest: "2.5.0", URL: "http://example.org/"},
		{Latest: "2.5.0", URL: "javascript:alert(1)"},
		{Latest: "2.5.0", URL: "https://user:pw@example.org/"},
		{Latest: "2.5.0", URL: "https:///chemin"},
		{Latest: "2.5.0", Notes: strings.Repeat("é", MaxVersionNotes+1)},
		{Latest: "2.5.0", Notes: "\xff"},
	}
	for _, v := range bad {
		x := signedNotice(t, priv, &v)
		if x.Verify(pub) == nil {
			t.Fatalf("avis invalide accepté : %+v", v)
		}
	}
	ok := signedNotice(t, priv, &VersionNotice{Latest: "2.5.0", Important: "2.5.0", Minimum: "2.5.0", Notes: strings.Repeat("é", MaxVersionNotes)})
	if err := ok.Verify(pub); err != nil {
		t.Fatal(err)
	}
}

func TestVersionNoticeLevels(t *testing.T) {
	v := &VersionNotice{Latest: "2.7.0", Important: "2.6.0", Minimum: "2.5.0"}
	for cur, want := range map[string]string{
		"2.4.9": LevelObsolete, "v2.4.9": LevelObsolete, "2.5.0": LevelImportant, "2.5.9": LevelImportant,
		"2.6.0": LevelAvailable, "v2.6.3": LevelAvailable, "2.7.0": LevelOK, "3.0.0": LevelOK,
		"dev": LevelOK, "": LevelOK, "n'importe": LevelOK,
	} {
		if got := v.Level(cur); got != want {
			t.Errorf("Level(%q) = %s, attendu %s", cur, got, want)
		}
	}
	if (*VersionNotice)(nil).Level("1.0.0") != LevelOK {
		t.Fatal("sans avis : ok")
	}
	for _, c := range []struct {
		cur      string
		allowDev bool
		refused  bool
	}{
		{"2.4.9", false, true}, {"2.5.0", false, false}, {"v3.0.0", false, false},
		{"", false, true}, {"dev", false, true}, {"dev", true, false}, {"2.5.0-rc1", false, true},
	} {
		if got := v.Refuses(c.cur, c.allowDev); got != c.refused {
			t.Errorf("Refuses(%q, %v) = %v", c.cur, c.allowDev, got)
		}
	}
	// sans minimum : personne n'est refusé, pas même les anciennes applications
	if (&VersionNotice{Latest: "2.7.0"}).Refuses("", false) || (*VersionNotice)(nil).Refuses("dev", false) {
		t.Fatal("refus sans minimum")
	}
}

func TestSemver(t *testing.T) {
	for _, s := range []string{"", "1", "1.2", "1.2.3.4", "01.2.3", "1.2.-3", "1.2.+3", " 1.2.3", "1.2.3 ", "1.2.3-rc1",
		"vv1.2.3", "1..3", "1.2.3\n", "4294967296.0.0", "1.2.x", "١.٢.٣"} {
		if _, err := ParseSemver(s); err == nil {
			t.Errorf("ParseSemver(%q) accepté", s)
		}
	}
	for a, b := range map[string]string{"1.2.3": "1.2.4", "1.9.9": "1.10.0", "0.0.9": "1.0.0", "v2.5.0": "2.5.1"} {
		if c, err := CompareSemver(a, b); err != nil || c != -1 {
			t.Errorf("%s < %s : %d %v", a, b, c, err)
		}
		if c, _ := CompareSemver(b, a); c != 1 {
			t.Errorf("%s > %s", b, a)
		}
	}
	if c, err := CompareSemver("v1.0.0", "1.0.0"); err != nil || c != 0 {
		t.Fatal("v1.0.0 = 1.0.0")
	}
}

func TestVersionNoticeInCatalog(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	cat := &Catalog{Version: 3, Issued: time.Now().Unix(), Sites: []SiteEntry{{ID: "demo", Root: strings.Repeat("a", 64)}}}
	if err := cat.Sign(priv); err != nil {
		t.Fatal(err)
	}
	// ajouté après la signature du catalogue (hors signature)
	cat.Versions = signedNotice(t, priv, &VersionNotice{Latest: "2.6.0", Minimum: "2.5.0"})
	b, _ := json.Marshal(cat)
	var got Catalog
	_ = json.Unmarshal(b, &got)
	if err := got.Verify(pub); err != nil || got.Versions == nil || got.Versions.Minimum != "2.5.0" {
		t.Fatalf("avis perdu : %v %+v", err, got.Versions)
	}
	// avis altéré par un serveur : catalogue toujours valide, avis ignoré
	got.Versions.Minimum = "2.6.0"
	if err := got.Verify(pub); err != nil || got.Versions != nil {
		t.Fatal("avis altéré conservé")
	}
	// anti-retour arrière : un ancien avis rejoué est refusé par la garde
	g := NewVersionGuard("")
	if g.Accept("versions", 5) != nil || g.Accept("versions", 4) == nil {
		t.Fatal("garde : retour arrière accepté")
	}
}

// FuzzSemver : jamais de panique ; tout ce qui est accepté se réécrit à
// l'identique (forme canonique) et la comparaison est un ordre total.
func FuzzSemver(f *testing.F) {
	for _, s := range []string{"1.2.3", "v0.0.0", "4294967295.1.1", "01.2.3", "1.2.3-rc1", "", "v", "1..2"} {
		f.Add(s, "2.5.0")
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		x, err := ParseSemver(a)
		if err != nil {
			return
		}
		if x.String() != strings.TrimPrefix(a, "v") {
			t.Fatalf("forme non canonique acceptée : %q", a)
		}
		y, err := ParseSemver(b)
		if err != nil {
			return
		}
		if x.Compare(y) != -y.Compare(x) || (x.Compare(y) == 0) != (x == y) {
			t.Fatalf("comparaison incohérente %q %q", a, b)
		}
		v := &VersionNotice{Version: 1, Issued: 1, Latest: a, Minimum: b}
		if v.Check() == nil && v.Level(b) == LevelObsolete {
			t.Fatal("le minimum lui-même est obsolète")
		}
	})
}
