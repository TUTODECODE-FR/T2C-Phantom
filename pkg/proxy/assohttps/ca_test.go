package assohttps

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// La CA ne doit pouvoir certifier que des noms .t2c.
func TestCANameConstraints(t *testing.T) {
	t.Setenv("T2C_DATA_DIR", t.TempDir())
	ca, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if !constrainedToT2C(ca.cert) {
		t.Fatal("CA sans Name Constraints .t2c")
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)
	cfg, err := ca.LeafTLS("tutodecode.t2c")
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(cfg.Certificates[0].Certificate[0])
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "tutodecode.t2c"}); err != nil {
		t.Fatalf("feuille .t2c refusée: %v", err)
	}
	if _, err := ca.LeafTLS("google.com"); err == nil {
		t.Fatal("LeafTLS a accepté un nom hors .t2c")
	}
	// même en signant directement avec la clé (clé volée), un certificat
	// pour google.com est rejeté par la vérification grâce aux Name Constraints
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "google.com"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"google.com"}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	forged, _ := x509.ParseCertificate(der)
	if _, err := forged.Verify(x509.VerifyOptions{Roots: roots, DNSName: "google.com"}); err == nil {
		t.Fatal("certificat google.com signé par la CA accepté")
	}
}

// Même avec la clé de la CA, impossible d'obtenir un certificat accepté pour
// une adresse IP, pour signer du code ou des e-mails ; clé en 0600.
func TestCAConstraintsIPAndUsage(t *testing.T) {
	t.Setenv("T2C_DATA_DIR", t.TempDir())
	ca, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)
	forge := func(tmpl *x509.Certificate) *x509.Certificate {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		tmpl.SerialNumber = big.NewInt(7)
		tmpl.NotBefore, tmpl.NotAfter = time.Now().Add(-time.Minute), time.Now().Add(time.Hour)
		der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
		if err != nil {
			t.Fatal(err)
		}
		c, _ := x509.ParseCertificate(der)
		return c
	}
	ip := forge(&x509.Certificate{Subject: pkix.Name{CommonName: "x"}, IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	if _, err := ip.Verify(x509.VerifyOptions{Roots: roots}); err == nil {
		t.Fatal("certificat pour une IPv4 accepté")
	}
	ip6 := forge(&x509.Certificate{Subject: pkix.Name{CommonName: "x"}, IPAddresses: []net.IP{net.ParseIP("2001:db8::1")},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	if _, err := ip6.Verify(x509.VerifyOptions{Roots: roots}); err == nil {
		t.Fatal("certificat pour une IPv6 accepté")
	}
	code := forge(&x509.Certificate{Subject: pkix.Name{CommonName: "Éditeur"}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}})
	if _, err := code.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}}); err == nil {
		t.Fatal("signature de code acceptée")
	}
	mail := forge(&x509.Certificate{Subject: pkix.Name{CommonName: "m"}, EmailAddresses: []string{"a@banque.example"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}})
	if _, err := mail.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}}); err == nil {
		t.Fatal("certificat S/MIME accepté")
	}
	st, err := os.Stat(filepath.Join(Dir(), caKeyFile))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("clé de la CA : %v %v", err, st.Mode().Perm())
	}
	// clé trop ouverte : remise en 0600 au chargement
	_ = os.Chmod(filepath.Join(Dir(), caKeyFile), 0o644)
	if _, err := LoadOrCreate(); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(filepath.Join(Dir(), caKeyFile)); st.Mode().Perm() != 0o600 {
		t.Fatalf("clé non resserrée : %o", st.Mode().Perm())
	}
	// ancienne CA (DNS seulement) : régénérée
	old := *ca.cert
	old.ExcludedIPRanges, old.ExtKeyUsage = nil, nil
	if constrainedToT2C(&old) {
		t.Fatal("ancienne CA sans contrainte IP gardée")
	}
}
