package wire

import (
	"bytes"
	"testing"
)

// FuzzDecode : toute trame reçue d'Internet. Jamais de panique ; une trame
// acceptée se ré-encode et se relit à l'identique (type et données).
func FuzzDecode(f *testing.F) {
	for _, fr := range []*Frame{
		{T: THello, H: Header{Pub: make([]byte, 32), Nonce: make([]byte, 32), Kind: "volunteer"}},
		{T: TData, H: Header{Q: 7, Kind: KindCatalog}, Data: []byte("{}")},
		{T: TCircCell, H: Header{Circ: 3}, Data: bytes.Repeat([]byte{1}, 512)},
	} {
		b, _ := Encode(fr)
		f.Add(b)
	}
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{byte(TCircDestroy), 0xff, 0xff})
	f.Fuzz(func(t *testing.T, b []byte) {
		fr, err := Decode(b)
		if err != nil {
			return
		}
		if len(fr.Data) > MaxPayload {
			t.Fatal("charge trop grande acceptée")
		}
		b2, err := Encode(fr)
		if err != nil {
			return // en-tête trop grand une fois ré-encodé : refusé à l'envoi
		}
		fr2, err := Decode(b2)
		if err != nil || fr2.T != fr.T || !bytes.Equal(fr2.Data, fr.Data) {
			t.Fatalf("aller-retour incohérent: %v", err)
		}
	})
}
