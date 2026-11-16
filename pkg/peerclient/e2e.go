package peerclient

import (
	"crypto/ecdh"
	"encoding/binary"
	"errors"

	"github.com/flynn/noise"

	"t2c/pkg/peerid"
)

// Chiffrement de bout en bout entre deux pairs (Noise IK, X25519/ChaChaPoly/BLAKE2s).
// Le VPS relaie ces messages sans pouvoir les lire ni les modifier.

var suite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

const partSize = 60000 // < 65535 (taille max d'un message Noise)

func prologue(initiator, responder string) []byte {
	return []byte("t2c/e2e/v1\n" + initiator + "\n" + responder)
}

func dhKey(k *ecdh.PrivateKey) noise.DHKey {
	return noise.DHKey{Private: k.Bytes(), Public: k.PublicKey().Bytes()}
}

func newInitiator(id *peerid.Identity, responderID string, responderNoise []byte) (*noise.HandshakeState, error) {
	return noise.NewHandshakeState(noise.Config{
		CipherSuite:   suite,
		Pattern:       noise.HandshakeIK,
		Initiator:     true,
		StaticKeypair: dhKey(id.Noise),
		PeerStatic:    responderNoise,
		Prologue:      prologue(id.ID(), responderID),
	})
}

func newResponder(id *peerid.Identity, initiatorID string) (*noise.HandshakeState, error) {
	return noise.NewHandshakeState(noise.Config{
		CipherSuite:   suite,
		Pattern:       noise.HandshakeIK,
		Initiator:     false,
		StaticKeypair: dhKey(id.Noise),
		Prologue:      prologue(initiatorID, id.ID()),
	})
}

// Réponse en clair : [statut 1o][longueur 4o][données], découpée en parties.
const (
	stOK  byte = 0
	stErr byte = 1
)

func encodeResponse(status byte, data []byte) []byte {
	out := make([]byte, 5+len(data))
	out[0] = status
	binary.BigEndian.PutUint32(out[1:5], uint32(len(data)))
	copy(out[5:], data)
	return out
}

// responseAssembler reconstitue la réponse à partir des parties déchiffrées.
type responseAssembler struct {
	buf  []byte
	want int
}

func (r *responseAssembler) add(p []byte, max int) (done bool, err error) {
	r.buf = append(r.buf, p...)
	if r.want == 0 && len(r.buf) >= 5 {
		r.want = 5 + int(binary.BigEndian.Uint32(r.buf[1:5]))
		if r.want-5 > max {
			return false, errors.New("réponse trop grande")
		}
	}
	if r.want > 0 && len(r.buf) > r.want {
		return false, errors.New("réponse trop longue")
	}
	return r.want > 0 && len(r.buf) == r.want, nil
}

func (r *responseAssembler) result() (byte, []byte) { return r.buf[0], r.buf[5:] }
