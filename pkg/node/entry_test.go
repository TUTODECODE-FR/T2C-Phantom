package node

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"t2c/pkg/peerclient"
	"t2c/pkg/trust"
)

// entryWorld : vingt bénévoles publics anciens (familles distinctes) dans
// l'annuaire appris, plus un officiel et une adresse invalide (jamais choisis).
func entryWorld(t *testing.T, since time.Time) (*Node, []peerclient.RelayKey, []entryCand) {
	t.Helper()
	n := &Node{cfg: Config{DataDir: t.TempDir()}}
	off, _, _ := ed25519.GenerateKey(nil)
	keys := []peerclient.RelayKey{{URL: "wss://off.example/v1/ws", Key: off}}
	var es []entryCand
	for i := 0; i < 20; i++ {
		k, _, _ := ed25519.GenerateKey(nil)
		es = append(es, entryCand{Key: hex.EncodeToString(k), URL: "ws://198.51.100.1:8443/v1/ws", Family: fmt.Sprint("f", i), Since: since.Unix()})
	}
	all := append(append([]entryCand{}, es...),
		entryCand{Key: hex.EncodeToString(off), URL: "ws://x/v1/ws", Since: since.Unix()}, // officiel : jamais
		entryCand{Key: es[0].Key[:62] + "00", URL: "http://x", Since: since.Unix()})       // adresse invalide
	b, _ := json.Marshal(entriesDoc{At: time.Now().Unix(), Share: 100, Entries: all})
	if err := os.WriteFile(n.entriesPath(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return n, keys, es
}

// Choix d'une entrée : toujours la même garde (jamais un nouveau bénévole à
// chaque connexion), jamais un officiel, une adresse invalide, un exclu ; rien
// en urgence ni pour un membre qui héberge.
func TestPickEntry(t *testing.T) {
	n, keys, es := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	first := n.pickEntry(keys)
	if first == nil {
		t.Fatal("aucune garde choisie parmi vingt bénévoles anciens")
	}
	for i := 0; i < 50; i++ {
		e := n.pickEntry(keys)
		if e == nil || e.Key != first.Key {
			t.Fatalf("entrée changée sans raison : %+v", e)
		}
	}
	if gd := n.loadGuards(); len(gd.Guards) != 2 || gd.Guards[0].Family == gd.Guards[1].Family {
		t.Fatalf("garde principale + secours de familles distinctes attendues : %+v", gd.Guards)
	}
	// consignes : exclus, puis urgence
	var banned []string
	for _, e := range es {
		banned = append(banned, e.Key)
	}
	n.catalog = &trust.Catalog{Policy: &trust.NetworkPolicy{Issued: 1, BannedRelays: banned, Expires: time.Now().Add(time.Hour).Unix()}}
	if e := n.pickEntry(keys); e != nil {
		t.Fatalf("bénévole exclu choisi : %s", e.Key)
	}
	n2, keys2, _ := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	n2.catalog = &trust.Catalog{Policy: &trust.NetworkPolicy{Issued: 1, Emergency: true, Expires: time.Now().Add(time.Hour).Unix()}}
	if n2.pickEntry(keys2) != nil {
		t.Fatal("mode urgence : entrée bénévole choisie")
	}
	// un membre qui héberge passe toujours par un officiel
	n3, keys3, _ := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	n3.hosting = Hosting{All: true}
	if n3.pickEntry(keys3) != nil {
		t.Fatal("hébergeur entré par un bénévole")
	}
	// trop récents (moins de 30 jours) : jamais gardes
	n4, keys4, _ := entryWorld(t, time.Now().Add(-10*24*time.Hour))
	if n4.pickEntry(keys4) != nil {
		t.Fatal("bénévole récent choisi comme garde")
	}
	n4.cfg.GuardMinAge = 5 * 24 * time.Hour
	if n4.pickEntry(keys4) == nil {
		t.Fatal("GuardMinAge réglable ignoré")
	}
}

// Gardes : enregistrées (même garde après redémarrage), secours en cas
// d'échec passager, remplacées après 3 échecs en une heure ou à expiration.
func TestEntryGuardsPersist(t *testing.T) {
	n, keys, _ := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	g := n.pickEntry(keys)
	if g == nil {
		t.Fatal("aucune garde")
	}
	restarted := &Node{cfg: n.cfg} // même dossier de données
	if e := restarted.pickEntry(keys); e == nil || e.Key != g.Key {
		t.Fatal("garde perdue au redémarrage")
	}
	backup := restarted.loadGuards().Guards[1].Key
	restarted.entryFailed(g.Key)
	if e := restarted.pickEntry(keys); e == nil || e.Key != backup {
		t.Fatal("échec passager : le secours doit servir")
	}
	restarted.entryOK(backup)
	restarted.entryFailed(g.Key)
	restarted.entryFailed(g.Key)
	gd := restarted.loadGuards()
	if len(gd.Guards[0].Fails) != 3 {
		t.Fatalf("échecs comptés : %+v", gd.Guards[0])
	}
	restarted.pickEntry(keys)
	gd = restarted.loadGuards()
	for _, x := range gd.Guards {
		if x.Key == g.Key {
			t.Fatal("garde gardée après 3 échecs en une heure")
		}
	}
	if gd.Guards[0].Key != backup || len(gd.Guards) != 2 {
		t.Fatalf("le secours doit devenir principal, un nouveau secours tiré : %+v", gd.Guards)
	}
	// expiration (60 jours)
	gd.Guards[0].Chosen = time.Now().Add(-GuardLife - time.Hour).Unix()
	restarted.saveGuards(gd)
	if e := restarted.pickEntry(keys); e == nil || e.Key == backup {
		t.Fatal("garde expirée encore utilisée")
	}
}

// Mode Maximal, ou entrée bénévole refusée : toujours un officiel.
func TestMaximalNoVolunteerEntry(t *testing.T) {
	n, keys, _ := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	n.hosting = Hosting{AnonMode: peerclient.ModeMaximal}
	for i := 0; i < 20; i++ {
		if n.pickEntry(keys) != nil {
			t.Fatal("mode Maximal : entrée bénévole choisie")
		}
	}
	off := false
	n.hosting = Hosting{AnonMode: peerclient.ModePartage, EntryVolunteers: &off}
	if n.pickEntry(keys) != nil {
		t.Fatal("entrée bénévole refusée par le membre, choisie quand même")
	}
	n.hosting = Hosting{AnonMode: peerclient.ModePartage}
	if n.pickEntry(keys) == nil {
		t.Fatal("mode Partagé : entrée par la garde attendue par défaut")
	}
}

// EntryShare : tirage « entrée bénévole ou officielle » d'abord, selon les consignes.
func TestEntryShareDraw(t *testing.T) {
	n, keys, _ := entryWorld(t, time.Now().Add(-60*24*time.Hour))
	n.catalog = &trust.Catalog{Policy: &trust.NetworkPolicy{Issued: 1, EntryPercent: 25, Expires: time.Now().Add(time.Hour).Unix()}}
	got := 0
	for i := 0; i < 400; i++ {
		if n.pickEntry(keys) != nil {
			got++
		}
	}
	if got < 50 || got > 160 {
		t.Fatalf("EntryPercent 25 : %d entrées bénévoles sur 400", got)
	}
	// consigne expirée : la part revient à 100 % (seules exclusions et urgence restent)
	n.catalog = &trust.Catalog{Policy: &trust.NetworkPolicy{Issued: 2, EntryPercent: 1, Expires: time.Now().Add(-time.Hour).Unix()}}
	for i := 0; i < 20; i++ {
		if n.pickEntry(keys) == nil {
			t.Fatal("consigne expirée : part d'entrée encore appliquée")
		}
	}
}

// Une identité neuve à chaque connexion par un bénévole.
func TestEntryIdentityFresh(t *testing.T) {
	n := &Node{cfg: Config{MintBits: 8}}
	a, err1 := n.entryIdentity()
	b, err2 := n.entryIdentity()
	if err1 != nil || err2 != nil || a.ID() == b.ID() {
		t.Fatal("identité d'entrée réutilisée")
	}
}
