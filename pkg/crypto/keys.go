package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"
	"time"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// GenerateKeyPair creates a new Ed25519 key pair for the node or service.
func GenerateKeyPair() (crypto.PrivKey, crypto.PubKey, error) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// GenerateSelfSignedCert generates an ephemeral TLS certificate in memory for end-to-end encryption.
func GenerateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"T2C Phantom Onion"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	return cert, nil
}

// GetT2CAddress returns the .t2c address derived from a public key.
func GetT2CAddress(pub crypto.PubKey) (string, error) {
	rawPub, err := pub.Raw()
	if err != nil {
		return "", err
	}
	
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawPub)
	encoded = strings.ToLower(encoded)
	
	return fmt.Sprintf("%s.t2c", encoded), nil
}

// ParseT2CAddress takes a .t2c address and returns the libp2p peer ID.
func ParseT2CAddress(addr string) (peer.ID, error) {
	if !strings.HasSuffix(addr, ".t2c") {
		return "", fmt.Errorf("invalid address format: must end with .t2c")
	}
	
	encoded := strings.TrimSuffix(addr, ".t2c")
	encoded = strings.ToUpper(encoded)
	
	rawPub, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base32 address: %w", err)
	}
	
	pubKey, err := crypto.UnmarshalEd25519PublicKey(rawPub)
	if err != nil {
		return "", err
	}
	
	return peer.IDFromPublicKey(pubKey)
}

// ParseT2CPublicKey takes a .t2c address and returns the libp2p public key.
func ParseT2CPublicKey(addr string) (crypto.PubKey, error) {
	if !strings.HasSuffix(addr, ".t2c") {
		addr = addr + ".t2c"
	}
	
	encoded := strings.TrimSuffix(addr, ".t2c")
	encoded = strings.ToUpper(encoded)
	
	rawPub, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base32 address: %w", err)
	}
	
	return crypto.UnmarshalEd25519PublicKey(rawPub)
}
