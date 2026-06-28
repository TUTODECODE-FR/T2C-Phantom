package tracker

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

// AntiReplayCache stocke les nonces récemment vus pour éviter les attaques par rejeu
var antiReplayCache = make(map[string]time.Time)

// AuthMiddleware protège les endpoints avec la cryptographie asymétrique (Ed25519) et l'anti-rejeu
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Récupération des en-têtes de sécurité
		pubKeyB64 := r.Header.Get("X-T2C-PubKey")
		signatureHex := r.Header.Get("X-T2C-Signature")
		timestampStr := r.Header.Get("X-T2C-Timestamp")
		nonce := r.Header.Get("X-T2C-Nonce")

		// 1. Validation de l'Anti-Rejeu (Timestamp)
		ts, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid Timestamp", http.StatusUnauthorized)
			return
		}
		
		reqTime := time.Unix(ts, 0)
		if time.Since(reqTime) > 5*time.Minute || time.Since(reqTime) < -5*time.Minute {
			http.Error(w, "Request expired or from the future (Replay Attack Prevention)", http.StatusUnauthorized)
			return
		}

		// 2. Validation du Nonce (Unicité de la requête)
		if _, exists := antiReplayCache[nonce]; exists {
			http.Error(w, "Nonce already used (Replay Attack Prevention)", http.StatusUnauthorized)
			return
		}
		antiReplayCache[nonce] = time.Now()

		// 3. Validation de la Signature Ed25519
		pubKey, err := base64.StdEncoding.DecodeString(pubKeyB64)
		if err != nil || len(pubKey) != ed25519.PublicKeySize {
			http.Error(w, "Invalid Public Key", http.StatusUnauthorized)
			return
		}

		signature, err := hex.DecodeString(signatureHex)
		if err != nil || len(signature) != ed25519.SignatureSize {
			http.Error(w, "Invalid Signature", http.StatusUnauthorized)
			return
		}

		// Le message signé doit être la concaténation de la méthode, de l'URL, du timestamp et du nonce
		message := r.Method + r.URL.Path + timestampStr + nonce
		
		if !ed25519.Verify(pubKey, []byte(message), signature) {
			http.Error(w, "Cryptographic Signature Verification Failed", http.StatusUnauthorized)
			return
		}

		// Nettoyage asynchrone basique du cache des nonces
		go cleanNonceCache()

		// Authentification réussie, passage au handler suivant
		next.ServeHTTP(w, r)
	}
}

func cleanNonceCache() {
	now := time.Now()
	for nonce, t := range antiReplayCache {
		if now.Sub(t) > 5*time.Minute {
			delete(antiReplayCache, nonce)
		}
	}
}
