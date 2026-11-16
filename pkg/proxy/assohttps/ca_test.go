package assohttps

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
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
