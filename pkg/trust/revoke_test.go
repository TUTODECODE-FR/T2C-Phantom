package trust

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// La révocation racine d'une clé de service s'applique aussi aux listes de
// bénévoles et aux tickets qu'elle a signés.
func TestRevocationHostsTickets(t *testing.T) {
	rootPub, rootPriv := keys(t)
	svcPub, svcPriv := keys(t)
	d, err := NewDelegation(rootPriv, svcPub, time.Hour, ScopeHosts, ScopeTicket)
	if err != nil {
		t.Fatal(err)
	}
	h := &HostsList{Version: 5, Issued: time.Now().Unix()}
	if err := h.SignDelegated(svcPriv, d); err != nil {
		t.Fatal(err)
	}
	tk := &Ticket{Peer: "p", Site: "s", Root: "r", Expires: time.Now().Add(time.Hour).Unix()}
	if err := tk.SignDelegated(svcPriv, d); err != nil {
		t.Fatal(err)
	}
	clean := &RelayList{Version: 1, Relays: []string{"wss://a/v1/ws"}}
	if h.VerifyNotRevoked(rootPub, nil) != nil || h.VerifyNotRevoked(rootPub, clean) != nil ||
		tk.VerifyNotRevoked(rootPub, clean) != nil {
		t.Fatal("documents valides refusés")
	}
	rl := &RelayList{Version: 2, Relays: []string{"wss://a/v1/ws"}, Revoked: []string{hex.EncodeToString(svcPub)}}
	if err := h.VerifyNotRevoked(rootPub, rl); !errors.Is(err, ErrRevoked) {
		t.Fatalf("liste de bénévoles d'une clé révoquée acceptée : %v", err)
	}
	if err := tk.VerifyNotRevoked(rootPub, rl); !errors.Is(err, ErrRevoked) {
		t.Fatalf("ticket d'une clé révoquée accepté : %v", err)
	}
	// la racine elle-même n'est jamais « révoquée » par cette liste
	hr := &HostsList{Version: 6}
	if hr.Sign(rootPriv) != nil || hr.VerifyNotRevoked(rootPub, rl) != nil || hr.Signer() != "root" {
		t.Fatal("liste racine refusée")
	}
	if h.Signer() != hex.EncodeToString(svcPub) || tk.Signer() != h.Signer() {
		t.Fatal("signataire inexact")
	}
}

// Une consigne déléguée ne peut exclure qu'un nombre borné de relais.
func TestDelegatedBanCap(t *testing.T) {
	rootPub, rootPriv := keys(t)
	svcPub, svcPriv := keys(t)
	d, _ := NewDelegation(rootPriv, svcPub, time.Hour, ScopePolicy)
	now := time.Now()
	mk := func(n int) *NetworkPolicy {
		p := &NetworkPolicy{Version: 1, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix()}
		for i := 0; i < n; i++ {
			pub, _ := keys(t)
			p.BannedRelays = append(p.BannedRelays, hex.EncodeToString(pub))
		}
		return p
	}
	p := mk(MaxDelegatedBans)
	if p.SignDelegated(svcPriv, d) != nil || p.Verify(rootPub, now) != nil {
		t.Fatal("consigne déléguée au plafond refusée")
	}
	p = mk(MaxDelegatedBans + 1)
	if p.SignDelegated(svcPriv, d) != nil || p.Verify(rootPub, now) == nil {
		t.Fatal("consigne déléguée au-delà du plafond acceptée")
	}
	p = mk(MaxDelegatedBans + 1)
	if p.SignDelegated(rootPriv, nil) != nil || p.Verify(rootPub, now) != nil {
		t.Fatal("exclusions massives refusées à la racine")
	}
}

func TestRelayEntryKey(t *testing.T) {
	pub, _ := keys(t)
	k := hex.EncodeToString(pub)
	for e, ok := range map[string]bool{
		"wss://a.example/v1/ws#k=" + k: true,
		"hidden:nom#k=" + k:            true,
		"wss://a.example/v1/ws":        false,
		"wss://a.example/v1/ws#k=zz":   false,
		"wss://a.example/v1/ws#x=" + k: false,
	} {
		if _, got := RelayEntryKey(e); got != ok {
			t.Fatalf("%s : %v, attendu %v", e, got, ok)
		}
	}
}

// T2C_ROOT_PUBKEY ne remplace jamais une clé racine intégrée.
func TestRootHexPrecedence(t *testing.T) {
	for _, c := range []struct{ compiled, env, want string }{
		{"aa", "bb", "aa"},
		{"", "bb", "bb"},
		{" aa ", "", "aa"},
		{"", "", ""},
	} {
		if got := rootHex(c.compiled, c.env); got != c.want {
			t.Fatalf("%+v : %q", c, got)
		}
	}
}
