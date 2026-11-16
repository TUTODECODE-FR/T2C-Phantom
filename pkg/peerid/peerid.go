// Package peerid : identité pseudonyme d'un pair.
//
// Un pair = une clé Ed25519 générée au hasard sur sa machine. Son ID (32 car.)
// est dérivé de la clé publique : il permet de le reconnaître (score, ban) sans
// rien révéler de son IP ni de qui il est. Créer une identité coûte une preuve
// de travail, pour qu'un banni ne puisse pas se recréer gratuitement.
package peerid

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultMintBits : difficulté de la PoW d'identité (~4 M de hashs, quelques secondes).
const DefaultMintBits = 22

const domNoise = "t2c/noise-key/v1\n"

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// IDFromPub : 32 caractères base32 minuscules de sha256(pub).
func IDFromPub(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	s := b32.EncodeToString(sum[:])
	out := make([]byte, 32)
	for i := 0; i < 32; i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func leadingZeroBits(b []byte) int {
	n := 0
	for _, x := range b {
		if x == 0 {
			n += 8
			continue
		}
		for m := byte(0x80); m != 0 && x&m == 0; m >>= 1 {
			n++
		}
		break
	}
	return n
}

func mintHash(pub ed25519.PublicKey, nonce uint64) [32]byte {
	var buf [ed25519.PublicKeySize + 8]byte
	copy(buf[:], pub)
	binary.BigEndian.PutUint64(buf[ed25519.PublicKeySize:], nonce)
	return sha256.Sum256(buf[:])
}

// CheckMint : la PoW d'identité atteint bits zéros de tête.
func CheckMint(pub ed25519.PublicKey, nonce uint64, bits int) bool {
	h := mintHash(pub, nonce)
	return leadingZeroBits(h[:]) >= bits
}

// Identity : clés d'un pair (ne quitte jamais sa machine).
type Identity struct {
	Priv      ed25519.PrivateKey
	Pub       ed25519.PublicKey
	MintNonce uint64
	MintBits  int
	Noise     *ecdh.PrivateKey // X25519 pour le chiffrement de bout en bout
	NoiseSig  []byte           // signature Ed25519 de la clé Noise publique
}

func (id *Identity) ID() string { return IDFromPub(id.Pub) }

// SignNoise / VerifyNoise : lie la clé X25519 à l'identité Ed25519.
func VerifyNoise(pub ed25519.PublicKey, noisePub, sig []byte) bool {
	return len(noisePub) == 32 && ed25519.Verify(pub, append([]byte(domNoise), noisePub...), sig)
}

// New génère une identité et calcule sa PoW.
func New(bits int) (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	id := &Identity{Priv: priv, Pub: pub, MintBits: bits}
	for n := uint64(0); ; n++ {
		if CheckMint(pub, n, bits) {
			id.MintNonce = n
			break
		}
	}
	return id, id.newNoise()
}

func (id *Identity) newNoise() error {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	id.Noise = k
	id.NoiseSig = ed25519.Sign(id.Priv, append([]byte(domNoise), k.PublicKey().Bytes()...))
	return nil
}

type fileFormat struct {
	Seed      string `json:"seed"`
	MintNonce uint64 `json:"mint_nonce"`
	MintBits  int    `json:"mint_bits"`
	Noise     string `json:"noise"`
}

// LoadOrCreate lit l'identité dans path (0600) ou en crée une.
func LoadOrCreate(path string, bits int) (*Identity, error) {
	if b, err := os.ReadFile(path); err == nil {
		var f fileFormat
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("peerid: %s illisible: %w", path, err)
		}
		seed, err := hex.DecodeString(f.Seed)
		if err != nil || len(seed) != ed25519.SeedSize {
			return nil, errors.New("peerid: graine invalide")
		}
		priv := ed25519.NewKeyFromSeed(seed)
		id := &Identity{Priv: priv, Pub: priv.Public().(ed25519.PublicKey), MintNonce: f.MintNonce, MintBits: f.MintBits}
		if !CheckMint(id.Pub, id.MintNonce, id.MintBits) {
			return nil, errors.New("peerid: PoW d'identité invalide")
		}
		nb, err := hex.DecodeString(f.Noise)
		if err != nil {
			return nil, err
		}
		if id.Noise, err = ecdh.X25519().NewPrivateKey(nb); err != nil {
			return nil, err
		}
		id.NoiseSig = ed25519.Sign(priv, append([]byte(domNoise), id.Noise.PublicKey().Bytes()...))
		return id, nil
	}
	id, err := New(bits)
	if err != nil {
		return nil, err
	}
	f := fileFormat{
		Seed:      hex.EncodeToString(id.Priv.Seed()),
		MintNonce: id.MintNonce,
		MintBits:  id.MintBits,
		Noise:     hex.EncodeToString(id.Noise.Bytes()),
	}
	b, _ := json.Marshal(f)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return nil, err
	}
	return id, nil
}
