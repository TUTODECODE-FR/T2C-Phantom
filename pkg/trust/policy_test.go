package trust

import (
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestNetworkPolicy(t *testing.T) {
	rootPub, rootPriv := keys(t)
	svcPub, svcPriv := keys(t)
	badPub, _ := keys(t)
	banned := hex.EncodeToString(badPub)
	now := time.Now()

	mk := func() *NetworkPolicy {
		return &NetworkPolicy{Version: 3, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix(),
			BannedRelays: []string{banned}, VolunteerBits: 28, MaxPerFamily: 4, HiddenRelayKBs: 64, VolunteerPercent: 50}
	}

	// délégation de portée policy : acceptée
	d, err := NewDelegation(rootPriv, svcPub, time.Hour, ScopePolicy)
	if err != nil {
		t.Fatal(err)
	}
	p := mk()
	if err := p.SignDelegated(svcPriv, d); err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(rootPub, now); err != nil {
		t.Fatalf("consignes déléguées refusées: %v", err)
	}
	// racine directe : acceptée
	r := mk()
	if r.SignDelegated(rootPriv, nil) != nil || r.Verify(rootPub, now) != nil {
		t.Fatal("consignes racine refusées")
	}

	// délégation sans la portée policy : refusée
	dh, _ := NewDelegation(rootPriv, svcPub, time.Hour, ScopeHosts, ScopeTicket)
	q := mk()
	_ = q.SignDelegated(svcPriv, dh)
	if q.Verify(rootPub, now) == nil {
		t.Fatal("délégation sans portée policy acceptée")
	}
	// altération après signature
	q = mk()
	_ = q.SignDelegated(svcPriv, d)
	q.Emergency = true
	if q.Verify(rootPub, now) == nil {
		t.Fatal("consignes altérées acceptées")
	}
	// expirées, trop longues, clé exclue mal formée
	q = mk()
	_ = q.SignDelegated(svcPriv, d)
	if q.Verify(rootPub, now.Add(2*time.Hour)) == nil {
		t.Fatal("consignes expirées acceptées")
	}
	q = mk()
	q.Expires = now.Add(MaxPolicyTTL + time.Hour).Unix()
	_ = q.SignDelegated(svcPriv, d)
	if q.Verify(rootPub, now) == nil {
		t.Fatal("durée excessive acceptée")
	}
	q = mk()
	q.BannedRelays = []string{"ZZ"}
	_ = q.SignDelegated(svcPriv, d)
	if q.Verify(rootPub, now) == nil {
		t.Fatal("clé exclue mal formée acceptée")
	}

	// application : ne peut que resserrer
	if !p.Banned(banned) || p.Banned(hex.EncodeToString(svcPub)) {
		t.Fatal("Banned")
	}
	if p.MinBits(24) != 28 || p.MinBits(32) != 32 {
		t.Fatal("MinBits")
	}
	if p.FamilyCap(8) != 4 || p.FamilyCap(2) != 2 {
		t.Fatal("FamilyCap")
	}
	if p.RelayKBs(128) != 64 || p.RelayKBs(32) != 32 {
		t.Fatal("RelayKBs")
	}
	if p.VolunteerShare() != 50 {
		t.Fatal("VolunteerShare")
	}
	p.Emergency = true
	if p.VolunteerShare() != 0 {
		t.Fatal("Emergency")
	}
	var none *NetworkPolicy
	if none.Banned(banned) || none.MinBits(24) != 24 || none.RelayKBs(128) != 128 || none.VolunteerShare() != 100 {
		t.Fatal("récepteur nil")
	}
	old := mk()
	old.Expires = now.Add(-time.Second).Unix()
	if old.Banned(banned) || old.VolunteerShare() != 100 {
		t.Fatal("consignes expirées appliquées")
	}

	// transport dans le catalogue, hors signature racine
	cat := &Catalog{Version: 1, Issued: now.Unix()}
	if err := cat.Sign(rootPriv); err != nil {
		t.Fatal(err)
	}
	p.Emergency = false
	_ = p.SignDelegated(svcPriv, d)
	cat.Policy = p
	raw, _ := json.Marshal(cat)
	var got Catalog
	_ = json.Unmarshal(raw, &got)
	if err := got.Verify(rootPub); err != nil || got.Policy == nil {
		t.Fatalf("catalogue avec consignes: %v %v", err, got.Policy)
	}
	got.Policy.VolunteerPercent = 1
	raw, _ = json.Marshal(&got)
	var tampered Catalog
	_ = json.Unmarshal(raw, &tampered)
	if err := tampered.Verify(rootPub); err != nil || tampered.Policy != nil {
		t.Fatal("consignes altérées non écartées du catalogue")
	}
}
