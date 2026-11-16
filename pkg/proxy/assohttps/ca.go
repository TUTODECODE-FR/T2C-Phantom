package assohttps

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"t2c/pkg/utils"
)

const (
	caCertFile = "asso-ca.crt"
	caKeyFile  = "asso-ca.key"
	caOrg      = "tutodecode"
	caCN       = "tutodecode CA"
)

// CA locale plug-and-play : ne signe des feuilles que pour des hostnames .t2c
// déjà autorisés par le caller (catalogue asso). La clé reste sur la machine.
type CA struct {
	mu   sync.Mutex
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	dir  string
}

func Dir() string {
	return filepath.Join(utils.GetT2CDataDir(), "asso-ca")
}

func LoadOrCreate() (*CA, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	certPath := filepath.Join(dir, caCertFile)
	keyPath := filepath.Join(dir, caKeyFile)
	if _, err := os.Stat(certPath); err == nil {
		c, err := load(dir, certPath, keyPath)
		if err != nil {
			return nil, err
		}
		// Ancienne CA « TUTODECODE Asso » → régénérer en « tutodecode »
		if c.cert.Subject.CommonName != caCN {
			_ = os.Remove(certPath)
			_ = os.Remove(keyPath)
			_ = os.Remove(filepath.Join(dir, ".trusted-v1"))
			return create(dir, certPath, keyPath)
		}
		return c, nil
	}
	return create(dir, certPath, keyPath)
}

func create(dir, certPath, keyPath string) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{caOrg},
			CommonName:   caCN,
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		// Pas de Name Constraints critiques : Firefox les refuse souvent sur les CA locales.
		// L'anti-spoof reste côté proxy (catalogue asso + publisher).
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key, dir: dir}, nil
}

func load(dir, certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	cb, _ := pem.Decode(certPEM)
	kb, _ := pem.Decode(keyPEM)
	if cb == nil || kb == nil {
		return nil, fmt.Errorf("asso-ca: PEM invalide")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParseECPrivateKey(kb.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key, dir: dir}, nil
}

func (c *CA) CertPEMPath() string {
	return filepath.Join(c.dir, caCertFile)
}

// CertPEM returns the CA certificate PEM bytes.
func (c *CA) CertPEM() ([]byte, error) {
	return os.ReadFile(c.CertPEMPath())
}

// ExportInstallFile copie la CA vers un chemin prêt à importer (Bureau / téléchargement).
func (c *CA) ExportInstallFile(dest string) error {
	pem, err := c.CertPEM()
	if err != nil {
		return err
	}
	return os.WriteFile(dest, pem, 0644)
}

// Nick is the NSS nickname for Firefox import.
func Nick() string { return caCN }

// Org returns the expected organization name ("tutodecode").
func Org() string { return caOrg }

// IsTutodecodeCA : true si le sujet est bien O=tutodecode / CN=tutodecode CA.
func (c *CA) IsTutodecodeCA() bool {
	if c == nil || c.cert == nil {
		return false
	}
	cn := c.cert.Subject.CommonName
	orgOK := false
	for _, o := range c.cert.Subject.Organization {
		if strings.EqualFold(o, caOrg) {
			orgOK = true
			break
		}
	}
	return orgOK && strings.EqualFold(cn, caCN)
}

// FingerprintSHA256 of the local CA cert.
func (c *CA) FingerprintSHA256() string {
	if c == nil || c.cert == nil {
		return ""
	}
	sum := sha256.Sum256(c.cert.Raw)
	return hex.EncodeToString(sum[:])
}

// LeafTLS pour un hostname .t2c. Le caller DOIT avoir vérifié IsAssoHTTPSAlias
// (anti-spoof catalogue). On refuse tout hors .t2c + Name Constraints CA.
func (c *CA) LeafTLS(hostname string) (*tls.Config, error) {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if !strings.HasSuffix(hostname, ".t2c") || strings.Count(hostname, ".") != 1 {
		// Un seul label : foo.t2c — refuse a.b.t2c / evil.com
		return nil, fmt.Errorf("hostname non autorisé")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname, Organization: []string{caOrg}},
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{hostname},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		return nil, err
	}
	leaf := tls.Certificate{
		Certificate: [][]byte{der, c.cert.Raw},
		PrivateKey:  key,
	}
	return &tls.Config{
		Certificates: []tls.Certificate{leaf},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
	}, nil
}

// ListenLocalHTTPS démarre un listener TLS sur addr (ex. 127.0.0.1:8443)
// qui présente un cert pour le SNI demandé (doit être asso).
func (c *CA) GetCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := strings.ToLower(chi.ServerName)
	if name == "" {
		return nil, fmt.Errorf("SNI manquant")
	}
	cfg, err := c.LeafTLS(name)
	if err != nil {
		return nil, err
	}
	return &cfg.Certificates[0], nil
}

func IsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
