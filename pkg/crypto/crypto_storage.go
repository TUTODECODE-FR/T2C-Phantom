package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"t2c/pkg/utils"
)

const (
	keySize    = 32 // AES-256
	iterations = 100000
)

// VaultPassphrase est la clé globale configurée au démarrage pour le chiffrement au repos.
// Si elle reste vide, GetMachineUUID() sera utilisé comme clé de repli locale.
var VaultPassphrase string

// DeriveKey dérive une clé AES-256 à partir d'un mot de passe et d'un sel fixe (pour correspondre au client Dart)
func DeriveKey(passphrase string) []byte {
	salt := []byte("t2c-phantom-salt")
	return pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)
}

// GetMachineUUID tente de récupérer un identifiant matériel unique et stable
func GetMachineUUID() string {
	var id string
	switch runtime.GOOS {
	case "linux":
		// Lecture de machine-id standard sous Linux
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id = strings.TrimSpace(string(data))
		} else if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			id = strings.TrimSpace(string(data))
		}
	}

	// Si aucun identifiant matériel n'est trouvé (macOS/Windows sans privilèges ou Linux altéré),
	// on utilise un fichier de clé d'ancrage local pour sécuriser le stockage local.
	if id == "" {
		keyPath := utils.GetPath("machine.key")
		if data, err := os.ReadFile(keyPath); err == nil {
			id = strings.TrimSpace(string(data))
		} else {
			// Génération d'une clé d'ancrage unique si elle n'existe pas
			token := make([]byte, 32)
			if _, err := rand.Read(token); err == nil {
				_ = os.MkdirAll(utils.GetT2CDataDir(), 0700)
				_ = os.WriteFile(keyPath, token, 0600)
				id = string(token)
			} else {
				id = "fallback-hardcoded-salt-t2c-phantom"
			}
		}
	}
	return id
}

// GetEffectivePassphrase renvoie le mot de passe configuré ou l'UUID machine par défaut
func GetEffectivePassphrase() string {
	if VaultPassphrase != "" {
		return VaultPassphrase
	}
	return GetMachineUUID()
}

// EncryptData chiffre des données avec AES-256-GCM
func EncryptData(data []byte, passphrase string) ([]byte, error) {
	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Format de sortie : [Nonce (12B)] [Ciphertext + Tag (Variable)]
	encrypted := gcm.Seal(nil, nonce, data, nil)
	result := make([]byte, 0, len(nonce)+len(encrypted))
	result = append(result, nonce...)
	result = append(result, encrypted...)

	return result, nil
}

// DecryptData déchiffre des données avec AES-256-GCM
func DecryptData(ciphertext []byte, passphrase string) ([]byte, error) {
	if len(ciphertext) < 12+16 { // Nonce (12B) + Tag (16B) minimum
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:12]
	actualCiphertext := ciphertext[12:]

	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	decrypted, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid key or corrupted data")
	}

	return decrypted, nil
}

// WriteEncryptedFile chiffre et écrit un fichier sur le disque
func WriteEncryptedFile(path string, data []byte) error {
	passphrase := GetEffectivePassphrase()
	encrypted, err := EncryptData(data, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encrypted, 0600)
}

// ReadDecryptedFile lit et déchiffre un fichier du disque
func ReadDecryptedFile(path string) ([]byte, error) {
	passphrase := GetEffectivePassphrase()
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecryptData(ciphertext, passphrase)
}

// DecryptToTempFile déchiffre un fichier dans un fichier temporaire sécurisé (Utile pour zip.OpenReader et streaming)
// Retourne le chemin du fichier temporaire et une fonction de nettoyage pour supprimer le fichier temporaire après usage.
func DecryptToTempFile(path string) (string, func(), error) {
	decrypted, err := ReadDecryptedFile(path)
	if err != nil {
		return "", nil, err
	}

	tmpFile, err := os.CreateTemp("", "t2c-decrypt-*.zip")
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmpFile.Name()

	_, err = tmpFile.Write(decrypted)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", nil, err
	}

	cleanup := func() {
		os.Remove(tmpPath)
	}

	return tmpPath, cleanup, nil
}
