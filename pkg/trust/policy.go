package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"
)

// NetworkPolicy : consignes réseau, signées sur le poste de l'éditeur par la
// « clé de consignes » (délégation DÉDIÉE, portée « policy » seule) ou par la
// racine. Le serveur maison ne fait que les PROPOSER (pilote automatique) et
// servir la dernière version signée : un intrus sur le serveur ne peut ni
// exclure, ni déclencher l'urgence, ni changer les parts. Appliquées par les
// relais officiels, les relais bénévoles et les applications.
//
// Règle de sécurité : une consigne ne peut que RESSERRER (plafonds plus bas,
// preuve de travail plus dure, relais exclus, moins de bénévoles). Elle ne
// peut jamais forcer un bénévole à faire sortie ni relever un plafond local :
// une clé de service volée ne peut que ralentir le réseau, pas désanonymiser.
// Elle peut toutefois exclure des bénévoles pour orienter le trafic vers les
// autres : une consigne DÉLÉGUÉE exclut donc au plus MaxDelegatedBans relais ;
// au-delà, seule la racine (hors ligne) peut signer.
//
// Elle expire (MaxPolicyTTL au plus) : si le serveur maison tombe, plafonds et
// pourcentages reviennent aux réglages par défaut, mais les relais exclus et le
// mode urgence de la DERNIÈRE consigne valide restent appliqués (mode dégradé,
// échec fermé) jusqu'à l'arrivée d'une consigne plus récente : laisser expirer
// les consignes ne doit jamais réintégrer un relais espion.
//
// Transportée dans le catalogue (champ « policy »), hors de sa signature,
// comme RelayList.
const domPolicy = "t2c/policy/v1"

// ScopePolicy : portée de délégation pour signer les consignes réseau.
const ScopePolicy = "policy"

const (
	MaxPolicyTTL  = 30 * 24 * time.Hour
	MaxPolicyBans = 1024
	// MaxDelegatedBans : exclusions max d'une consigne signée par une clé de
	// service (au-delà : racine seulement).
	MaxDelegatedBans = 64
	maxPolicyBits    = 40
)

type NetworkPolicy struct {
	Version uint64 `json:"version"` // strictement croissante (anti-retour arrière)
	Issued  int64  `json:"issued"`
	Expires int64  `json:"expires"`
	// BannedRelays : clés publiques (hex) de relais exclus du réseau.
	BannedRelays []string `json:"banned_relays,omitempty"`
	// VolunteerBits : preuve de travail minimale des relais bénévoles (0 = inchangé).
	VolunteerBits int `json:"volunteer_bits,omitempty"`
	// MaxPerFamily : bénévoles max par famille réseau (0 = inchangé).
	MaxPerFamily int `json:"max_per_family,omitempty"`
	// HiddenRelayKBs : plafond du relais caché des applications, Ko/s (0 = inchangé).
	HiddenRelayKBs int64 `json:"hidden_relay_kbs,omitempty"`
	// VolunteerPercent : part des circuits qui passent par des bénévoles,
	// 1..100 (0 = inchangé, c.-à-d. 100).
	VolunteerPercent int `json:"volunteer_percent,omitempty"`
	// EntryPercent : part des connexions des membres qui peuvent ENTRER par un
	// relais bénévole public (qui voit alors leur adresse IP), 1..100
	// (0 = inchangé, c.-à-d. 100). Réglage distinct de VolunteerPercent.
	EntryPercent int `json:"entry_percent,omitempty"`
	// Emergency : tout passe par les relais officiels (aucun bénévole).
	Emergency bool        `json:"emergency,omitempty"`
	Deleg     *Delegation `json:"deleg,omitempty"`
	Sig       string      `json:"sig"`
}

func (p *NetworkPolicy) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	p.Deleg = d
	u := *p
	u.Sig = ""
	s, err := sign(priv, domPolicy, u)
	p.Sig = s
	return err
}

// Verify : forme, validité dans le temps et signature (racine ou délégation
// de portée « policy »).
func (p *NetworkPolicy) Verify(root ed25519.PublicKey, now time.Time) error {
	bad := errors.New("trust: consignes réseau invalides")
	if p.Version == 0 || p.Expires <= p.Issued || time.Duration(p.Expires-p.Issued)*time.Second > MaxPolicyTTL {
		return bad
	}
	if p.Issued > now.Unix()+60 {
		return errors.New("trust: consignes réseau futures")
	}
	if len(p.BannedRelays) > MaxPolicyBans || p.VolunteerBits < 0 || p.VolunteerBits > maxPolicyBits ||
		p.MaxPerFamily < 0 || p.HiddenRelayKBs < 0 || p.VolunteerPercent < 0 || p.VolunteerPercent > 100 ||
		p.EntryPercent < 0 || p.EntryPercent > 100 {
		return bad
	}
	if p.Deleg != nil && len(p.BannedRelays) > MaxDelegatedBans {
		return errors.New("trust: trop d'exclusions pour une clé de service (racine requise)")
	}
	for _, k := range p.BannedRelays {
		if b, err := hex.DecodeString(k); err != nil || len(b) != ed25519.PublicKeySize || k != hex.EncodeToString(b) {
			return bad
		}
	}
	// délégation dédiée : une délégation de service (hosts, tickets…) qui
	// porterait aussi « policy » (anciennes délégations du serveur) est refusée.
	if p.Deleg != nil && (len(p.Deleg.Scopes) != 1 || p.Deleg.Scopes[0] != ScopePolicy) {
		return ErrPolicyScope
	}
	k, err := signerKey(root, p.Deleg, ScopePolicy)
	if err != nil {
		return err
	}
	u := *p
	u.Sig = ""
	if err := verify(k, domPolicy, u, p.Sig); err != nil {
		return err
	}
	if now.Unix() >= p.Expires { // signature valable : l'appelant peut la garder en mode dégradé
		return ErrPolicyExpired
	}
	return nil
}

// ErrPolicyScope : consignes signées par une délégation qui n'est pas dédiée
// aux consignes (par exemple la clé de service du serveur).
var ErrPolicyScope = errors.New("trust: consignes signées hors d'une délégation dédiée « policy »")

// ErrPolicyExpired : consignes authentiques mais expirées. Seuls leurs relais
// exclus et leur mode urgence restent appliqués (voir live).
var ErrPolicyExpired = errors.New("trust: consignes réseau expirées")

// PolicyKeeper garde la dernière consigne valide vue (relais, applications) :
// si le catalogue n'en porte plus (serveur maison arrêté), ses exclusions et
// son mode urgence restent en vigueur. Une consigne plus ancienne (rejouée)
// ne la remplace jamais.
type PolicyKeeper struct {
	mu sync.Mutex
	p  *NetworkPolicy
}

// Keep : retient p s'il est au moins aussi récent, renvoie la consigne en vigueur.
func (k *PolicyKeeper) Keep(p *NetworkPolicy) *NetworkPolicy {
	k.mu.Lock()
	defer k.mu.Unlock()
	if p != nil && (k.p == nil || p.Issued >= k.p.Issued) {
		k.p = p
	}
	return k.p
}

// Signer : clé qui a signé (hex de la clé de service, ou « root »). Sert à
// l'anti-retour par signataire : une clé révoquée qui aurait publié une
// version énorme ne bloque pas les consignes de la nouvelle clé.
func (p *NetworkPolicy) Signer() string { return delegSigner(p.Deleg) }

// revoked : la clé de service est révoquée par la liste racine.
func (r *RelayList) revoked(key string) bool {
	return r != nil && key != "root" && slices.Contains(r.Revoked, key)
}

// Revokes : version exportée de revoked (relais, serveur).
func (r *RelayList) Revokes(key string) bool { return r.revoked(key) }

// live : consignes encore valides (une copie en cache peut avoir expiré).
// Toutes les méthodes ci-dessous acceptent un récepteur nil (aucune consigne).
// Plafonds et pourcentages ne s'appliquent que tant qu'elles sont valides ;
// exclusions et mode urgence restent appliqués après expiration (échec fermé).
func (p *NetworkPolicy) live() bool {
	return p != nil && time.Now().Unix() < p.Expires
}

// Banned : la clé (hex minuscule) est exclue du réseau (même consigne expirée).
func (p *NetworkPolicy) Banned(keyHex string) bool {
	return p != nil && slices.Contains(p.BannedRelays, keyHex)
}

// Urgent : mode urgence (même consigne expirée) : tout par les officiels.
func (p *NetworkPolicy) Urgent() bool { return p != nil && p.Emergency }

// MinBits : preuve de travail exigée = la plus dure des deux.
func (p *NetworkPolicy) MinBits(local int) int {
	if p.live() && p.VolunteerBits > local {
		return p.VolunteerBits
	}
	return local
}

// FamilyCap : plafond par famille = le plus bas des deux (0 = aucun).
func (p *NetworkPolicy) FamilyCap(local int) int {
	if p.live() && p.MaxPerFamily > 0 && (local <= 0 || p.MaxPerFamily < local) {
		return p.MaxPerFamily
	}
	return local
}

// RelayKBs : plafond du relais caché = le plus bas des deux.
func (p *NetworkPolicy) RelayKBs(local int64) int64 {
	if p.live() && p.HiddenRelayKBs > 0 && (local <= 0 || p.HiddenRelayKBs < local) {
		return p.HiddenRelayKBs
	}
	return local
}

// VolunteerShare : pourcentage des circuits autorisés à passer par des bénévoles.
func (p *NetworkPolicy) VolunteerShare() int {
	switch {
	case p.Urgent():
		return 0
	case p.live() && p.VolunteerPercent > 0:
		return p.VolunteerPercent
	}
	return 100
}

// EntryShare : pourcentage des connexions autorisées à entrer par un relais
// bénévole public (tirage « entrée bénévole ou officielle » fait d'abord).
func (p *NetworkPolicy) EntryShare() int {
	switch {
	case p.Urgent():
		return 0
	case p.live() && p.EntryPercent > 0:
		return p.EntryPercent
	}
	return 100
}
