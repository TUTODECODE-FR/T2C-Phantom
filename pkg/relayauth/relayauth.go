// Package relayauth : authentification des requêtes relais → origine sans
// secret partagé. Le relais signe chaque requête avec sa clé Ed25519 ; l'origine
// ne connaît que la clé publique (rien de sensible à voler sur le serveur).
// Horodatage + nonce : une requête capturée ne peut pas être rejouée.
package relayauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Header = "X-T2C-Relay-Auth"
	domain = "t2c/relay-req/v1"
	// Window : écart d'horloge toléré entre relais et origine.
	Window = 60 * time.Second
)

var ErrAuth = errors.New("relayauth: requête non authentifiée")

func payload(method, path string, ts int64, nonce string, body []byte) []byte {
	h := sha256.Sum256(body)
	return []byte(domain + "\n" + method + "\n" + path + "\n" + strconv.FormatInt(ts, 10) + "\n" + nonce + "\n" + hex.EncodeToString(h[:]))
}

// Sign renvoie la valeur de l'en-tête Header pour cette requête.
func Sign(priv ed25519.PrivateKey, method, path string, body []byte) string {
	var n [16]byte
	_, _ = rand.Read(n[:])
	nonce := hex.EncodeToString(n[:])
	ts := time.Now().Unix()
	sig := ed25519.Sign(priv, payload(method, path, ts, nonce, body))
	return strconv.FormatInt(ts, 10) + "." + nonce + "." + hex.EncodeToString(sig)
}

// Verifier : côté origine. Garde les nonces vus pendant 2×Window.
type Verifier struct {
	Keys []ed25519.PublicKey
	mu   sync.Mutex
	seen map[string]int64
}

func (v *Verifier) Verify(header, method, path string, body []byte, now time.Time) error {
	parts := strings.Split(header, ".")
	if len(parts) != 3 || len(parts[1]) != 32 {
		return ErrAuth
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ErrAuth
	}
	if d := now.Unix() - ts; d > int64(Window/time.Second) || d < -int64(Window/time.Second) {
		return ErrAuth
	}
	sig, err := hex.DecodeString(parts[2])
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrAuth
	}
	p := payload(method, path, ts, parts[1], body)
	ok := false
	for _, k := range v.Keys {
		if ed25519.Verify(k, p, sig) {
			ok = true
			break
		}
	}
	if !ok {
		return ErrAuth
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen == nil {
		v.seen = map[string]int64{}
	}
	if len(v.seen) > 100_000 {
		for k, exp := range v.seen {
			if exp < now.Unix() {
				delete(v.seen, k)
			}
		}
	}
	if _, dup := v.seen[parts[1]]; dup {
		return ErrAuth
	}
	v.seen[parts[1]] = now.Add(2 * Window).Unix()
	return nil
}
