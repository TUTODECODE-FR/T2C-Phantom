package node

// Relais d'entrée bénévoles (phase C1), côté membre.
//
// L'annuaire signé par un relais officiel donne les bénévoles qui ont CHOISI
// d'être aussi relais d'entrée (adresse publique vérifiée par l'officiel). Un
// membre qui n'héberge rien peut y entrer, en mode anonyme seulement : le
// bénévole voit son adresse IP mais ne lit rien (cellules chiffrées pour les
// sauts suivants). Le trafic ne converge plus vers les seuls relais officiels.
//
// Le membre s'y présente sous une identité jetable neuve à chaque connexion (en
// mémoire, jamais écrite), distincte de son identité habituelle. Il n'entre que
// par sa garde (voir pickEntry) : un seul bénévole voit son adresse, longtemps.

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"t2c/pkg/peerclient"
	"t2c/pkg/peerid"
	"t2c/pkg/trust"
)

const (
	entriesTTL = 24 * time.Hour // annuaire appris trop vieux : ignoré
	maxEntries = 64
)

type entryCand struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Family string `json:"family"`
	Since  int64  `json:"since"`
}

// entriesDoc : <data>/entries.json, appris d'un annuaire signé vérifié.
type entriesDoc struct {
	At      int64       `json:"at"`
	Share   int         `json:"share"` // part des bénévoles selon les consignes vues alors
	Entries []entryCand `json:"entries"`
}

func (n *Node) entriesPath() string { return filepath.Join(n.cfg.DataDir, "entries.json") }

func volunteerURL(u string) bool {
	return len(u) <= trust.MaxVolunteerURL && (strings.HasPrefix(u, "ws://") || strings.HasPrefix(u, "wss://"))
}

// learnEntries retient les relais d'entrée bénévoles de l'annuaire signé.
func (n *Node) learnEntries(ctx context.Context, c *peerclient.Client, cat *trust.Catalog) {
	if n.cfg.NoAnonymity {
		return
	}
	vctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	doc := entriesDoc{At: time.Now().Unix(), Share: cat.Policy.EntryShare(), Entries: []entryCand{}}
	for _, v := range c.PublicVolunteers(vctx) {
		if len(doc.Entries) < maxEntries && volunteerURL(v.URL) {
			doc.Entries = append(doc.Entries, entryCand{Key: hexKey(v.Key), URL: v.URL, Family: v.Family, Since: v.Since.Unix()})
		}
	}
	b, _ := json.Marshal(doc)
	_ = os.WriteFile(n.entriesPath(), b, 0o600)
}

// Gardes d'entrée (comme les « guards » de Tor) : un bénévole qui voit
// l'adresse IP du membre la voit à chaque connexion ; tirer une nouvelle entrée
// à chaque fois finirait par la montrer à un bénévole espion. On garde donc UN
// relais d'entrée (et un secours) pendant GuardLife, et on n'en change qu'en
// cas d'échecs répétés, d'exclusion par les consignes ou d'expiration. Seuls
// les bénévoles listés depuis longtemps (GuardMinAge, ancienneté tenue par
// les relais officiels) peuvent devenir gardes.
const (
	GuardLife          = 60 * 24 * time.Hour // durée de vie d'une garde
	DefaultGuardMinAge = 30 * 24 * time.Hour // ancienneté minimale d'un bénévole pour devenir garde
	guardFailMax       = 3                   // échecs de construction en guardFailWindow : garde abandonnée
	guardFailWindow    = time.Hour
	guardDownMax       = 24 * time.Hour // injoignable depuis plus longtemps : abandonnée
)

type guardRec struct {
	Key       string  `json:"key"`
	URL       string  `json:"url"`
	Family    string  `json:"family"`
	Chosen    int64   `json:"chosen"`
	Fails     []int64 `json:"fails,omitempty"`      // échecs récents (Unix)
	FirstFail int64   `json:"first_fail,omitempty"` // premier échec depuis le dernier succès
}

// guardsDoc : <data>/guards.json (garde principale, puis secours).
type guardsDoc struct {
	Guards []guardRec `json:"guards"`
}

func (n *Node) guardsPath() string { return filepath.Join(n.cfg.DataDir, "guards.json") }

func (n *Node) guardMinAge() time.Duration {
	switch {
	case n.cfg.GuardMinAge < 0:
		return 0
	case n.cfg.GuardMinAge == 0:
		return DefaultGuardMinAge
	}
	return n.cfg.GuardMinAge
}

func (n *Node) loadGuards() guardsDoc {
	var d guardsDoc
	if b, err := os.ReadFile(n.guardsPath()); err == nil {
		_ = json.Unmarshal(b, &d)
	}
	return d
}

func (n *Node) saveGuards(d guardsDoc) {
	b, _ := json.Marshal(d)
	_ = os.WriteFile(n.guardsPath(), b, 0o600)
}

// currentPolicy : consignes en vigueur (la dernière valide est gardée, voir
// trust.PolicyKeeper). À appeler sous n.mu.
func (n *Node) currentPolicy() *trust.NetworkPolicy {
	var p *trust.NetworkPolicy
	if n.catalog != nil {
		p = n.catalog.Policy
	}
	return n.policyKeep.Keep(p)
}

// volunteerEntryAllowed : le membre peut-il entrer par un bénévole ? Jamais
// pour un membre qui héberge (ses tickets passent par les officiels), jamais
// hors mode anonyme, jamais en mode Maximal, jamais s'il l'a refusé.
func (n *Node) volunteerEntryAllowed() bool {
	h := n.hostingPrefs()
	return !n.cfg.NoAnonymity && !h.active() && h.entryVolunteersOn()
}

// pickEntry : garde d'entrée bénévole à utiliser pour cette connexion, ou nil
// pour un relais officiel. Les consignes décident d'abord, au hasard, entre
// entrée bénévole (EntryShare %) et officielle ; puis la garde principale
// sert, ou le secours si elle est momentanément en échec.
func (n *Node) pickEntry(keys []peerclient.RelayKey) *entryCand {
	if len(keys) == 0 || !n.volunteerEntryAllowed() {
		return nil
	}
	var doc entriesDoc
	b, err := os.ReadFile(n.entriesPath())
	if err != nil || json.Unmarshal(b, &doc) != nil || time.Since(time.Unix(doc.At, 0)) > entriesTTL {
		return nil
	}
	n.mu.Lock()
	pol := n.currentPolicy()
	n.mu.Unlock()
	if pol.VolunteerShare() == 0 || pol.EntryShare() == 0 {
		return nil
	}
	if share := pol.EntryShare(); share < 100 && mrand.IntN(100) >= share {
		return nil // consignes : cette connexion entre par un officiel
	}
	n.guardMu.Lock()
	defer n.guardMu.Unlock()
	gd := n.refreshGuards(doc, pol, keys)
	now := time.Now()
	listed := map[string]entryCand{}
	for _, e := range doc.Entries {
		listed[e.Key] = e
	}
	for _, g := range gd.Guards {
		e, ok := listed[g.Key]
		if !ok || g.cooling(now) {
			continue // absente de l'annuaire ou en échec récent : le secours, sinon un officiel
		}
		return &e
	}
	return nil
}

// cooling : garde en échec il y a moins de 2 minutes (on essaie le secours).
func (g guardRec) cooling(now time.Time) bool {
	return len(g.Fails) > 0 && now.Sub(time.Unix(g.Fails[len(g.Fails)-1], 0)) < 2*time.Minute
}

// refreshGuards : retire les gardes expirées, exclues ou trop souvent en échec,
// puis complète jusqu'à 2 (principale + secours) parmi les bénévoles éligibles.
// Enregistré dans guards.json. À appeler sous n.guardMu.
func (n *Node) refreshGuards(doc entriesDoc, pol *trust.NetworkPolicy, keys []peerclient.RelayKey) guardsDoc {
	gd := n.loadGuards()
	now := time.Now()
	own := n.hiddenRelayKey()
	changed := false
	var kept []guardRec
	dropped := map[string]bool{} // retirées à l'instant : pas re-tirées au sort dans la foulée
	for _, g := range gd.Guards {
		recent := g.Fails[:0:0]
		for _, f := range g.Fails {
			if now.Sub(time.Unix(f, 0)) < guardFailWindow {
				recent = append(recent, f)
			}
		}
		if len(recent) != len(g.Fails) {
			g.Fails, changed = recent, true
		}
		pk, err := trust.ParsePublicKey(g.Key)
		drop := err != nil || pol.Banned(g.Key) || now.Sub(time.Unix(g.Chosen, 0)) > GuardLife || g.Chosen > now.Unix()+60 ||
			len(g.Fails) >= guardFailMax || g.FirstFail > 0 && now.Sub(time.Unix(g.FirstFail, 0)) > guardDownMax ||
			own.Equal(pk) || officialKey(pk, keys) || len(kept) >= 2
		if drop {
			dropped[g.Key], changed = true, true
			continue
		}
		kept = append(kept, g)
	}
	minAge := n.guardMinAge()
	for len(kept) < 2 {
		var cand []entryCand
		for _, e := range doc.Entries {
			pk, err := trust.ParsePublicKey(e.Key)
			if err != nil || dropped[e.Key] || !volunteerURL(e.URL) || pol.Banned(e.Key) || own.Equal(pk) || officialKey(pk, keys) ||
				now.Sub(time.Unix(e.Since, 0)) < minAge {
				continue
			}
			dup := false
			for _, g := range kept { // jamais deux gardes d'une même famille réseau
				dup = dup || g.Key == e.Key || e.Family != "" && g.Family == e.Family
			}
			if !dup {
				cand = append(cand, e)
			}
		}
		if len(cand) == 0 {
			break
		}
		e := cand[mrand.IntN(len(cand))]
		kept = append(kept, guardRec{Key: e.Key, URL: e.URL, Family: e.Family, Chosen: now.Unix()})
		changed = true
	}
	gd.Guards = kept
	if changed {
		n.saveGuards(gd)
	}
	return gd
}

func officialKey(k ed25519.PublicKey, keys []peerclient.RelayKey) bool {
	for _, r := range keys {
		if r.Key.Equal(k) {
			return true
		}
	}
	return false
}

// entryFailed : échec de connexion ou de construction de circuit par la garde
// key (au bout de guardFailMax en une heure, elle est remplacée).
func (n *Node) entryFailed(key string) {
	n.guardMu.Lock()
	defer n.guardMu.Unlock()
	gd := n.loadGuards()
	now := time.Now().Unix()
	for i := range gd.Guards {
		if gd.Guards[i].Key == key {
			gd.Guards[i].Fails = append(gd.Guards[i].Fails, now)
			if gd.Guards[i].FirstFail == 0 {
				gd.Guards[i].FirstFail = now
			}
		}
	}
	n.saveGuards(gd)
}

// entryOK : connexion réussie par la garde key.
func (n *Node) entryOK(key string) {
	n.guardMu.Lock()
	defer n.guardMu.Unlock()
	gd := n.loadGuards()
	for i := range gd.Guards {
		if gd.Guards[i].Key == key {
			gd.Guards[i].Fails, gd.Guards[i].FirstFail = nil, 0
		}
	}
	n.saveGuards(gd)
}

// entryIdentity : identité jetable NEUVE pour chaque connexion à un relais
// d'entrée bénévole (jamais enregistrée) : la garde ne peut pas relier entre
// elles les connexions du membre par une clé, ni à son identité d'hébergeur.
func (n *Node) entryIdentity() (*peerid.Identity, error) {
	return peerid.New(n.cfg.MintBits)
}
