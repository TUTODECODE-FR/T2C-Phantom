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
		// Ancienne CA (autre nom, ou sans Name Constraints .t2c) → on la remplace.
		// La clé de l'ancienne est supprimée : même restée « de confiance » dans un
		// navigateur, elle ne peut plus rien signer.
		if c.cert.Subject.CommonName != caCN || !constrainedToT2C(c.cert) {
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
		// Usage limité aux serveurs web (pas de signature de code, ni de mail).
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// Name Constraints : noms DNS en .t2c seulement, aucune adresse IP,
		// aucun e-mail ni URI utilisable. Un navigateur à jour refuse donc tout
		// certificat de cette CA pour un autre site, même signé avec sa clé
		// volée. Les vieux logiciels qui ignorent ces contraintes ne sont pas
		// protégés : la clé reste lisible par le seul compte du membre (0600).
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         []string{"t2c"},
		ExcludedIPRanges:            noIPs(),
		PermittedEmailAddresses:     []string{"t2c.invalid"},
		PermittedURIDomains:         []string{"t2c.invalid"},
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
	// clé privée lisible par le seul compte du membre
	if st, err := os.Stat(keyPath); err == nil && st.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(keyPath, 0o600); err != nil {
			return nil, fmt.Errorf("asso-ca: clé trop ouverte (%o) : %w", st.Mode().Perm(), err)
		}
	}
	_ = os.Chmod(dir, 0o700)
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

// noIPs : toutes les adresses IPv4 et IPv6 exclues.
func noIPs() []*net.IPNet {
	return []*net.IPNet{
		{IP: net.IPv4zero.To4(), Mask: net.CIDRMask(0, 32)},
		{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
	}
}

// constrainedToT2C : la CA est limitée aux noms .t2c, sans aucune IP, et aux
// serveurs web. Une CA plus ancienne (sans ces contraintes) est régénérée.
func constrainedToT2C(cert *x509.Certificate) bool {
	if !cert.PermittedDNSDomainsCritical || len(cert.PermittedDNSDomains) != 1 || cert.PermittedDNSDomains[0] != "t2c" {
		return false
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth || len(cert.PermittedIPRanges) != 0 {
		return false
	}
	v4, v6 := false, false
	for _, n := range cert.ExcludedIPRanges {
		if ones, bits := n.Mask.Size(); ones == 0 {
			v4 = v4 || bits == 32
			v6 = v6 || bits == 128
		}
	}
	return v4 && v6
}
