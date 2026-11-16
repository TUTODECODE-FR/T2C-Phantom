// Package trust : tout ce qui circule dans T2C Phantom est soit signé par la
// clé Ed25519 de l'origine (catalogue, manifestes, hosts, tickets), soit adressé
// par son hash (morceaux). Un relais, un pair ou le VPS peut transporter ces
// données mais pas les modifier sans que ce soit détecté.
package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// RootPubKeyHex : clé publique de l'origine intégrée au client.
// Fixée au build : -ldflags "-X t2c/pkg/trust.RootPubKeyHex=<hex>"
// ou à l'exécution via T2C_ROOT_PUBKEY (tests / réseau de dev).
var RootPubKeyHex = ""

var (
	rootOnce sync.Once
	rootKey  ed25519.PublicKey
	rootErr  error
)

// RootKey renvoie la clé publique de confiance.
func RootKey() (ed25519.PublicKey, error) {
	rootOnce.Do(func() {
		h := strings.TrimSpace(os.Getenv("T2C_ROOT_PUBKEY"))
		if h == "" {
			h = RootPubKeyHex
		}
		if h == "" {
			rootErr = errors.New("trust: aucune clé racine (RootPubKeyHex / T2C_ROOT_PUBKEY)")
			return
		}
		rootKey, rootErr = ParsePublicKey(h)
	})
	return rootKey, rootErr
}

func ParsePublicKey(h string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(strings.TrimSpace(h))
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("trust: clé publique invalide")
	}
	return ed25519.PublicKey(b), nil
}

// GenerateKey crée une paire et écrit la graine privée (hex, 0600) dans path.
func GenerateKey(path string) (ed25519.PublicKey, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("trust: %s existe déjà (refus d'écraser)", path)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	seed := hex.EncodeToString(priv.Seed())
	if err := os.WriteFile(path, []byte(seed+"\n"), 0o600); err != nil {
		return nil, err
	}
	return pub, nil
}

// LoadPrivateKey lit une graine hex écrite par GenerateKey.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("trust: %s doit être en 0600 (actuel %o)", path, st.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("trust: graine invalide dans %s", path)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
