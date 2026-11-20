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
// Fixée au build : -ldflags "-X t2c/pkg/trust.RootPubKeyHex=<hex>".
// T2C_ROOT_PUBKEY n'est lue que si aucune clé n'est intégrée (version de
// développement, tests) : elle ne remplace JAMAIS la clé d'une release.
var RootPubKeyHex = ""

var (
	rootOnce sync.Once
	rootKey  ed25519.PublicKey
	rootErr  error
)

// RootKey renvoie la clé publique de confiance.
func RootKey() (ed25519.PublicKey, error) {
	rootOnce.Do(func() {
		h := rootHex(RootPubKeyHex, os.Getenv("T2C_ROOT_PUBKEY"))
		if h == "" {
			rootErr = errors.New("trust: aucune clé racine (RootPubKeyHex / T2C_ROOT_PUBKEY)")
			return
		}
		rootKey, rootErr = ParsePublicKey(h)
	})
	return rootKey, rootErr
}

// rootHex : la clé intégrée l'emporte toujours ; la variable d'environnement
// ne sert qu'aux versions sans clé intégrée.
func rootHex(compiled, env string) string {
	if c := strings.TrimSpace(compiled); c != "" {
		return c
	}
	return strings.TrimSpace(env)
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

// LoadPrivateKey lit une graine hex écrite par GenerateKey, ou une clé chiffrée
// (EncryptKeyFile) en demandant sa phrase de passe (Passphrase).
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
	if IsEncryptedKey(data) {
		pass, err := Passphrase(path)
		if err != nil {
			return nil, err
		}
		seed, err := decryptSeed(data, pass)
		if err != nil {
			return nil, err
		}
		return ed25519.NewKeyFromSeed(seed), nil
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("trust: graine invalide dans %s", path)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
