package trust

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptedKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "signing.key")
	pub, err := GenerateKey(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncryptKeyFile(p, []byte("court")); err == nil {
		t.Fatal("phrase trop courte acceptée")
	}
	if _, err := EncryptKeyFile(p, []byte("une phrase de passe solide")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !IsEncryptedKey(raw) || strings.Contains(string(raw), "seed") {
		t.Fatal("fichier non chiffré")
	}
	old := Passphrase
	defer func() { Passphrase = old }()
	Passphrase = func(string) ([]byte, error) { return []byte("une phrase de passe solide"), nil }
	k, err := LoadPrivateKey(p)
	if err != nil || !k.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatalf("relecture: %v", err)
	}
	Passphrase = func(string) ([]byte, error) { return []byte("une phrase de passe fausse"), nil }
	if _, err := LoadPrivateKey(p); err != ErrBadPassphrase {
		t.Fatalf("mauvaise phrase: %v", err)
	}
	// fichier altéré
	tampered := strings.Replace(string(raw), `"data": "`, `"data": "AA`, 1)
	_ = os.WriteFile(p, []byte(tampered), 0o600)
	Passphrase = func(string) ([]byte, error) { return []byte("une phrase de passe solide"), nil }
	if _, err := LoadPrivateKey(p); err == nil {
		t.Fatal("fichier altéré accepté")
	}
	if _, err := EncryptKeyFile(p, []byte("une phrase de passe solide")); err == nil {
		t.Fatal("double chiffrement accepté")
	}
}
