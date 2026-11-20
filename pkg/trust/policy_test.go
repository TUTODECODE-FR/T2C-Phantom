package trust

import (
	"crypto/ed25519"
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
	// ancienne délégation du serveur (hosts, tickets ET policy) : refusée,
	// seule une délégation dédiée aux consignes peut les signer
	dm, _ := NewDelegation(rootPriv, svcPub, time.Hour, ScopeHosts, ScopeTicket, ScopePolicy)
	q = mk()
	_ = q.SignDelegated(svcPriv, dm)
	if err := q.Verify(rootPub, now); err != ErrPolicyScope {
		t.Fatalf("délégation mixte acceptée pour les consignes: %v", err)
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
	if !old.Banned(banned) || old.VolunteerShare() != 100 || old.MinBits(24) != 24 || old.RelayKBs(128) != 128 {
		t.Fatal("consignes expirées : exclusions gardées, plafonds et parts rendus")
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

// Serveur piraté : la racine révoque sa clé de service ; ses consignes sont
// écartées tout de suite, et l'anti-retour est tenu par signataire.
func TestPolicyRevocation(t *testing.T) {
	rootPub, rootPriv := keys(t)
	svcPub, svcPriv := keys(t)
	now := time.Now()
	d, _ := NewDelegation(rootPriv, svcPub, time.Hour, ScopePolicy)
	p := &NetworkPolicy{Version: 1 << 60, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix(), Emergency: true}
	_ = p.SignDelegated(svcPriv, d)
	if p.Signer() != hex.EncodeToString(svcPub) {
		t.Fatal("Signer")
	}
	mk := func(revoked ...string) *Catalog {
		cat := &Catalog{Version: 1, Issued: now.Unix()}
		_ = cat.Sign(rootPriv)
		rl := &RelayList{Version: 1, Issued: now.Unix(), Relays: []string{"wss://a/v1/ws#k=00"}, Revoked: revoked}
		_ = rl.Sign(rootPriv)
		cat.Relays, cat.Policy = rl, p
		raw, _ := json.Marshal(cat)
		var got Catalog
		_ = json.Unmarshal(raw, &got)
		if err := got.Verify(rootPub); err != nil || got.Relays == nil {
			t.Fatalf("catalogue: %v", err)
		}
		return &got
	}
	if mk().Policy == nil {
		t.Fatal("consignes non révoquées écartées")
	}
	if mk(hex.EncodeToString(svcPub)).Policy != nil {
		t.Fatal("consignes d'une clé révoquée acceptées")
	}
	bad := &RelayList{Version: 1, Relays: []string{"wss://a"}, Revoked: []string{"xx"}}
	_ = bad.Sign(rootPriv)
	if bad.Verify(rootPub) == nil {
		t.Fatal("clé révoquée mal formée acceptée")
	}
}

// Échec fermé : une consigne expirée garde ses exclusions et son mode urgence
// (dans le catalogue comme dans PolicyKeeper) ; plafonds et parts reviennent
// aux défauts ; une consigne plus ancienne ne remplace jamais la dernière.
func TestPolicyFailClosed(t *testing.T) {
	rootPub, rootPriv := keys(t)
	badPub, _ := keys(t)
	banned := hex.EncodeToString(badPub)
	now := time.Now()
	p := &NetworkPolicy{Version: 2, Issued: now.Add(-2 * time.Hour).Unix(), Expires: now.Add(-time.Hour).Unix(),
		BannedRelays: []string{banned}, Emergency: true, VolunteerPercent: 10, EntryPercent: 10, VolunteerBits: 30}
	if err := p.SignDelegated(rootPriv, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(rootPub, now); err != ErrPolicyExpired {
		t.Fatalf("consigne expirée authentique : %v", err)
	}
	cat := &Catalog{Version: 1, Issued: now.Unix()}
	if err := cat.Sign(rootPriv); err != nil {
		t.Fatal(err)
	}
	cat.Policy = p
	raw, _ := json.Marshal(cat)
	var got Catalog
	_ = json.Unmarshal(raw, &got)
	if err := got.Verify(rootPub); err != nil || got.Policy == nil {
		t.Fatalf("consigne expirée écartée du catalogue : %v", err)
	}
	gp := got.Policy
	if !gp.Banned(banned) || !gp.Urgent() || gp.VolunteerShare() != 0 || gp.EntryShare() != 0 || gp.MinBits(24) != 24 {
		t.Fatal("mode dégradé : exclusions et urgence attendues, plafonds par défaut")
	}
	gp.Emergency = false // altérée : écartée même expirée
	raw, _ = json.Marshal(&got)
	var bad Catalog
	_ = json.Unmarshal(raw, &bad)
	if bad.Verify(rootPub) != nil || bad.Policy != nil {
		t.Fatal("consigne expirée altérée gardée")
	}
	var k PolicyKeeper
	if k.Keep(p) != p || k.Keep(nil) != p {
		t.Fatal("PolicyKeeper : dernière consigne perdue quand le catalogue n'en porte plus")
	}
	older := *p
	older.Issued--
	older.Emergency = false
	if k.Keep(&older) != p {
		t.Fatal("PolicyKeeper : consigne plus ancienne acceptée (rejeu)")
	}
	newer := &NetworkPolicy{Version: 3, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix()}
	if k.Keep(newer) != newer || newer.Urgent() || newer.VolunteerShare() != 100 {
		t.Fatal("PolicyKeeper : consigne plus récente ignorée")
	}
}

// EntryShare : réglage distinct de VolunteerPercent, ne peut que resserrer.
func TestEntryShare(t *testing.T) {
	now := time.Now()
	var none *NetworkPolicy
	if none.EntryShare() != 100 {
		t.Fatal("défaut : entrées bénévoles permises")
	}
	p := &NetworkPolicy{Version: 1, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix(), EntryPercent: 30, VolunteerPercent: 80}
	if p.EntryShare() != 30 || p.VolunteerShare() != 80 {
		t.Fatal("EntryShare / VolunteerShare")
	}
	p.EntryPercent = 0
	if p.EntryShare() != 100 {
		t.Fatal("EntryPercent 0 = inchangé")
	}
	p.Emergency = true
	if p.EntryShare() != 0 {
		t.Fatal("urgence : aucune entrée bénévole")
	}
	_, priv := keys(t)
	q := &NetworkPolicy{Version: 1, Issued: now.Unix(), Expires: now.Add(time.Hour).Unix(), EntryPercent: 101}
	_ = q.SignDelegated(priv, nil)
	if q.Verify(priv.Public().(ed25519.PublicKey), now) == nil {
		t.Fatal("EntryPercent > 100 accepté")
	}
}
