package crypto

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

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
