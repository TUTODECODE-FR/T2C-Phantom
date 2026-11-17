package onion

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestRelayKeyConversion(t *testing.T) {
	for i := 0; i < 20; i++ {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		xp, err := RelayPublic(pub)
		if err != nil {
			t.Fatal(err)
		}
		xs, _ := RelayPrivate(priv)
		if !bytes.Equal(xs.PublicKey().Bytes(), xp.Bytes()) {
			t.Fatal("clé X25519 dérivée incohérente")
		}
	}
}

func TestHandshakeAndLayers(t *testing.T) {
	var relays []ed25519.PrivateKey
	for i := 0; i < 3; i++ {
		_, k, _ := ed25519.GenerateKey(rand.Reader)
		relays = append(relays, k)
	}
	var cHops, rHops []*Hop
	for _, r := range relays {
		c, _ := NewClient(r.Public().(ed25519.PublicKey))
		reply, rh, err := Respond(r, c.Message())
		if err != nil {
			t.Fatal(err)
		}
		ch, err := c.Finish(reply)
		if err != nil {
			t.Fatal(err)
		}
		cHops, rHops = append(cHops, ch), append(rHops, rh)
	}
	for round := 0; round < 3; round++ {
		msg := []byte("demande anonyme")
		cell := msg
		for i := len(cHops) - 1; i >= 0; i-- {
			cell = cHops[i].SealForward(cell)
		}
		for i, h := range rHops {
			var err error
			cell, err = h.OpenForward(cell)
			if err != nil {
				t.Fatalf("saut %d: %v", i, err)
			}
			if i < len(rHops)-1 && bytes.Contains(cell, msg) {
				t.Fatal("un relais intermédiaire lit la demande")
			}
		}
		if !bytes.Equal(cell, msg) {
			t.Fatal("demande altérée")
		}
		resp := []byte("réponse")
		for i := len(rHops) - 1; i >= 0; i-- {
			resp = rHops[i].SealBackward(resp)
		}
		for _, h := range cHops {
			var err error
			resp, err = h.OpenBackward(resp)
			if err != nil {
				t.Fatal(err)
			}
		}
		if string(resp) != "réponse" {
			t.Fatal("réponse altérée")
		}
	}
}

func TestImpostorRelayRejected(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, impostor, _ := ed25519.GenerateKey(rand.Reader)
	c, _ := NewClient(pub)
	reply, _, err := Respond(impostor, c.Message())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Finish(reply); err == nil {
		t.Fatal("relais imposteur accepté")
	}
	// cellule modifiée en route
	_, r, _ := ed25519.GenerateKey(rand.Reader)
	c2, _ := NewClient(r.Public().(ed25519.PublicKey))
	rep, rh, _ := Respond(r, c2.Message())
	ch, _ := c2.Finish(rep)
	cell := ch.SealForward([]byte("x"))
	cell[0] ^= 1
	if _, err := rh.OpenForward(cell); err == nil {
		t.Fatal("cellule altérée acceptée")
	}
}

func TestPadding(t *testing.T) {
	sizes := map[int]bool{}
	for _, n := range []int{1, 40, 900, 1019, 1020, 5000, 60000, 262_300} {
		p := bytes.Repeat([]byte{CmdFrame}, n)
		c, ok := Pad(p)
		if !ok {
			t.Fatalf("%d octets non enveloppés", n)
		}
		sizes[len(c)] = true
		got, ok := Unpad(c)
		if !ok || !bytes.Equal(got, p) {
			t.Fatalf("aller-retour %d octets", n)
		}
	}
	if len(sizes) > 4 {
		t.Fatalf("trop de tailles visibles: %v", sizes)
	}
	// une recherche et une cellule leurre ont la même taille
	small, _ := Pad([]byte{CmdFrame, 1, 2, 3})
	if len(small) != len(Drop()) {
		t.Fatal("cellule leurre discernable par sa taille")
	}
	if _, ok := Pad(make([]byte, MaxCell)); ok {
		t.Fatal("commande trop grande acceptée")
	}
	for _, bad := range [][]byte{nil, {CmdPadded}, {CmdPadded, 0, 0, 0, 0}, {CmdPadded, 0, 0, 1, 0, 1}} {
		if _, ok := Unpad(bad); ok {
			t.Fatalf("enveloppe invalide acceptée: %v", bad)
		}
	}
}
