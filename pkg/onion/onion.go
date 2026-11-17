// Package onion : chiffrement en couches des circuits T2C (routage en oignon).
//
// Un client bâtit un circuit à travers plusieurs relais de la fédération. Avec
// chaque relais il négocie des clés propres (échange X25519 authentifié, sur le
// modèle « ntor » de Tor) :
//
//	client : x éphémère, envoie X
//	relais : y éphémère, b = clé statique (dérivée de sa clé Ed25519)
//	secret = X25519(y, X) ‖ X25519(b, X)   (côté client : X25519(x, Y) ‖ X25519(x, B))
//	clés   = HKDF(secret, sel, info = clé du relais ‖ X ‖ Y) → avant, arrière, auth
//	le relais prouve qu'il détient b par un MAC sur Y.
//
// Confidentialité persistante : y et x sont jetés après l'échange ; voler plus
// tard la clé statique d'un relais ne déchiffre pas un circuit passé.
//
// Chaque cellule est chiffrée une fois par saut (AES-256-GCM, compteur comme nonce,
// une clé par sens) : chaque relais ne retire ou n'ajoute qu'une couche.
package onion

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"math/big"
	"sync"
)

const (
	domain   = "t2c/onion/v1"
	KeyLen   = 32
	ReplyLen = 64 // Y ‖ MAC
	// Overhead : octets ajoutés par une couche.
	Overhead = 16
)

var ErrHandshake = errors.New("onion: échange de clés refusé")

// --- clés X25519 des relais, dérivées de leur clé Ed25519 ----------------

var p25519 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))

// RelayPublic : clé X25519 d'un relais à partir de sa clé publique Ed25519
// (u = (1+y)/(1−y) mod p). Donnée publique : pas besoin de temps constant.
func RelayPublic(ed ed25519.PublicKey) (*ecdh.PublicKey, error) {
	if len(ed) != ed25519.PublicKeySize {
		return nil, ErrHandshake
	}
	le := make([]byte, 32)
	copy(le, ed)
	le[31] &= 0x7f
	be := make([]byte, 32)
	for i := range le {
		be[31-i] = le[i]
	}
	y := new(big.Int).SetBytes(be)
	if y.Cmp(p25519) >= 0 {
		return nil, ErrHandshake
	}
	one := big.NewInt(1)
	num := new(big.Int).Add(one, y)
	den := new(big.Int).Sub(one, y)
	den.Mod(den, p25519)
	if den.Sign() == 0 {
		return nil, ErrHandshake
	}
	u := num.Mul(num, new(big.Int).ModInverse(den, p25519))
	u.Mod(u, p25519)
	ub := u.FillBytes(make([]byte, 32))
	out := make([]byte, 32)
	for i := range ub {
		out[31-i] = ub[i]
	}
	return ecdh.X25519().NewPublicKey(out)
}

// RelayPrivate : clé X25519 correspondante (scalaire Ed25519 = SHA-512(graine)[:32]).
func RelayPrivate(ed ed25519.PrivateKey) (*ecdh.PrivateKey, error) {
	h := sha512.Sum512(ed.Seed())
	return ecdh.X25519().NewPrivateKey(h[:32])
}

// --- échange de clés -----------------------------------------------------

func derive(shared, relayEd, X, Y []byte) (kf, kb, kauth []byte, err error) {
	info := append(append(append([]byte{}, relayEd...), X...), Y...)
	okm, err := hkdf.Key(sha256.New, shared, []byte(domain), string(info), 3*KeyLen)
	if err != nil {
		return nil, nil, nil, err
	}
	return okm[:KeyLen], okm[KeyLen : 2*KeyLen], okm[2*KeyLen:], nil
}

func mac(kauth, Y []byte) []byte {
	m := hmac.New(sha256.New, kauth)
	m.Write([]byte(domain + "/server"))
	m.Write(Y)
	return m.Sum(nil)
}

// ClientHandshake : moitié client d'un échange avec un relais donné.
type ClientHandshake struct {
	x       *ecdh.PrivateKey
	relayEd ed25519.PublicKey
	relayX  *ecdh.PublicKey
}

// NewClient prépare l'échange avec le relais relayEd ; Message() est à lui envoyer.
func NewClient(relayEd ed25519.PublicKey) (*ClientHandshake, error) {
	rx, err := RelayPublic(relayEd)
	if err != nil {
		return nil, err
	}
	x, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ClientHandshake{x: x, relayEd: relayEd, relayX: rx}, nil
}

func (c *ClientHandshake) Message() []byte { return c.x.PublicKey().Bytes() }

// Finish vérifie la réponse du relais et renvoie les clés du saut (côté client).
func (c *ClientHandshake) Finish(reply []byte) (*Hop, error) {
	if len(reply) != ReplyLen {
		return nil, ErrHandshake
	}
	Y, err := ecdh.X25519().NewPublicKey(reply[:32])
	if err != nil {
		return nil, ErrHandshake
	}
	s1, err1 := c.x.ECDH(Y)
	s2, err2 := c.x.ECDH(c.relayX)
	if err1 != nil || err2 != nil {
		return nil, ErrHandshake
	}
	kf, kb, kauth, err := derive(append(s1, s2...), c.relayEd, c.Message(), reply[:32])
	if err != nil || !hmac.Equal(mac(kauth, reply[:32]), reply[32:]) {
		return nil, ErrHandshake
	}
	return newHop(kf, kb)
}

// Respond : côté relais. Renvoie la réponse à transmettre et les clés du saut.
func Respond(relay ed25519.PrivateKey, msg []byte) ([]byte, *Hop, error) {
	X, err := ecdh.X25519().NewPublicKey(msg)
	if err != nil {
		return nil, nil, ErrHandshake
	}
	b, err := RelayPrivate(relay)
	if err != nil {
		return nil, nil, err
	}
	y, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	s1, err1 := y.ECDH(X)
	s2, err2 := b.ECDH(X)
	if err1 != nil || err2 != nil {
		return nil, nil, ErrHandshake
	}
	Y := y.PublicKey().Bytes()
	kf, kb, kauth, err := derive(append(s1, s2...), relay.Public().(ed25519.PublicKey), msg, Y)
	if err != nil {
		return nil, nil, err
	}
	h, err := newHop(kf, kb)
	if err != nil {
		return nil, nil, err
	}
	return append(Y, mac(kauth, Y)...), h, nil
}

// --- couches --------------------------------------------------------------

// Hop : clés d'un saut. Le client et le relais en détiennent la même copie.
// Sens « avant » = client → sortie ; « arrière » = sortie → client.
type Hop struct {
	mu       sync.Mutex
	fwd, bwd cipher.AEAD
	nf, nb   uint64
}

func newHop(kf, kb []byte) (*Hop, error) {
	f, err := gcm(kf)
	if err != nil {
		return nil, err
	}
	b, err := gcm(kb)
	if err != nil {
		return nil, err
	}
	return &Hop{fwd: f, bwd: b}, nil
}

func gcm(k []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

func nonce(n uint64) []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint64(b[4:], n)
	return b
}

var ErrCell = errors.New("onion: cellule invalide")

// SealForward (client) ajoute la couche de ce saut dans le sens avant.
func (h *Hop) SealForward(p []byte) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nf++
	return h.fwd.Seal(nil, nonce(h.nf), p, nil)
}

// OpenForward (relais) retire sa couche dans le sens avant.
func (h *Hop) OpenForward(c []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nf++
	p, err := h.fwd.Open(nil, nonce(h.nf), c, nil)
	if err != nil {
		return nil, ErrCell
	}
	return p, nil
}

// SealBackward (relais) ajoute sa couche dans le sens arrière.
func (h *Hop) SealBackward(p []byte) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nb++
	return h.bwd.Seal(nil, nonce(h.nb), p, nil)
}

// OpenBackward (client) retire la couche de ce saut dans le sens arrière.
func (h *Hop) OpenBackward(c []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nb++
	p, err := h.bwd.Open(nil, nonce(h.nb), c, nil)
	if err != nil {
		return nil, ErrCell
	}
	return p, nil
}

// --- commandes transportées dans un circuit -------------------------------

const (
	CmdExtend   byte = 1 // client → dernier saut : [clé Ed25519 du suivant 32o][X 32o]
	CmdExtended byte = 2 // dernier saut → client : [réponse 64o]
	CmdBegin    byte = 3 // client → sortie : pseudonyme jetable du circuit
	CmdFrame    byte = 4 // trame wire encodée (requête du client ou réponse de la sortie)
)
