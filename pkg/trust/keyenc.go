package trust

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// Clé privée chiffrée par une phrase de passe (clé racine du poste de
// l'éditeur) : volé seul, le fichier ne sert à rien.
//
//	scrypt (N=2^17, r=8, p=1 : ~0,5 s et 128 Mo par essai, rend la force brute
//	très coûteuse) → clé AES-256-GCM, la graine Ed25519 chiffrée et authentifiée.
const keyEncMagic = "t2c-cle-chiffree-v1"

type encKeyFile struct {
	Format string `json:"format"`
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	Salt   string `json:"salt"`
	Nonce  string `json:"nonce"`
	Data   string `json:"data"`
	Pub    string `json:"pub"` // clé publique en clair : vérifie le déchiffrement
}

// MinPassphrase : longueur minimale d'une phrase de passe.
const MinPassphrase = 12

// Passphrase fournit la phrase de passe d'une clé chiffrée. Par défaut :
// variable T2C_PASSPHRASE ; les outils en ligne de commande la remplacent par
// une saisie au clavier sans écho.
var Passphrase = func(path string) ([]byte, error) {
	if p := os.Getenv("T2C_PASSPHRASE"); p != "" {
		return []byte(p), nil
	}
	return nil, fmt.Errorf("trust: %s est chiffrée : phrase de passe requise", path)
}

// ErrBadPassphrase : phrase de passe incorrecte (ou fichier altéré).
var ErrBadPassphrase = errors.New("trust: phrase de passe incorrecte")

func deriveKey(pass, salt []byte, n, r, p int) ([]byte, error) {
	return scrypt.Key(pass, salt, n, r, p, 32)
}

// IsEncryptedKey : le fichier est-il une clé chiffrée ?
func IsEncryptedKey(data []byte) bool { return strings.Contains(string(data), keyEncMagic) }

func encryptSeed(seed, pass []byte) ([]byte, error) {
	if len([]rune(string(pass))) < MinPassphrase {
		return nil, fmt.Errorf("trust: phrase de passe trop courte (%d caractères minimum)", MinPassphrase)
	}
	salt := make([]byte, 32)
	nonce := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	f := encKeyFile{Format: keyEncMagic, N: 1 << 17, R: 8, P: 1,
		Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(nonce),
		Pub: hex.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey))}
	k, err := deriveKey(pass, salt, f.N, f.R, f.P)
	if err != nil {
		return nil, err
	}
	blk, _ := aes.NewCipher(k)
	gcm, _ := cipher.NewGCM(blk)
	f.Data = base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, seed, []byte(keyEncMagic+f.Pub)))
	b, err := json.MarshalIndent(f, "", "  ")
	return append(b, '\n'), err
}

func decryptSeed(data, pass []byte) ([]byte, error) {
	var f encKeyFile
	if err := json.Unmarshal(data, &f); err != nil || f.Format != keyEncMagic {
		return nil, errors.New("trust: fichier de clé chiffrée illisible")
	}
	if f.N < 1<<14 || f.N > 1<<22 || f.R < 1 || f.R > 32 || f.P < 1 || f.P > 16 {
		return nil, errors.New("trust: paramètres de chiffrement invalides")
	}
	salt, e1 := base64.StdEncoding.DecodeString(f.Salt)
	nonce, e2 := base64.StdEncoding.DecodeString(f.Nonce)
	ct, e3 := base64.StdEncoding.DecodeString(f.Data)
	if e1 != nil || e2 != nil || e3 != nil || len(nonce) != 12 {
		return nil, errors.New("trust: fichier de clé chiffrée illisible")
	}
	k, err := deriveKey(pass, salt, f.N, f.R, f.P)
	if err != nil {
		return nil, err
	}
	blk, _ := aes.NewCipher(k)
	gcm, _ := cipher.NewGCM(blk)
	seed, err := gcm.Open(nil, nonce, ct, []byte(keyEncMagic+f.Pub))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, ErrBadPassphrase
	}
	if hex.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)) != f.Pub {
		return nil, ErrBadPassphrase
	}
	return seed, nil
}

// EncryptKeyFile chiffre sur place une clé en clair (écrite par GenerateKey).
// Le fichier en clair est d'abord écrasé puis remplacé.
func EncryptKeyFile(path string, pass []byte) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if IsEncryptedKey(data) {
		return nil, fmt.Errorf("trust: %s est déjà chiffrée", path)
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("trust: graine invalide dans %s", path)
	}
	enc, err := encryptSeed(seed, pass)
	if err != nil {
		return nil, err
	}
	// vérification avant de toucher à l'original
	if s2, err := decryptSeed(enc, pass); err != nil || hex.EncodeToString(s2) != hex.EncodeToString(seed) {
		return nil, errors.New("trust: vérification du chiffrement échouée, clé inchangée")
	}
	tmp := path + ".chiffree"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return nil, err
	}
	// écrase la graine en clair avant de la remplacer (au mieux : SSD/APFS)
	_ = os.WriteFile(path, make([]byte, len(data)), 0o600)
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey), nil
}
