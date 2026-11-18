package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"slices"
	"time"
)

// NetworkPolicy : consignes réseau, émises en direct par le serveur maison
// (clé de service déléguée, portée « policy ») ou par la racine. Appliquées
// par les relais officiels, les relais bénévoles et les applications.
//
// Règle de sécurité : une consigne ne peut que RESSERRER (plafonds plus bas,
// preuve de travail plus dure, relais exclus, moins de bénévoles). Elle ne
// peut jamais forcer un bénévole à faire sortie ni relever un plafond local :
// une clé de service volée ne peut que ralentir le réseau, pas désanonymiser.
//
// Elle expire (MaxPolicyTTL au plus) : si le serveur maison tombe, tout le
// monde revient aux réglages par défaut.
//
// Transportée dans le catalogue (champ « policy »), hors de sa signature,
// comme RelayList.
const domPolicy = "t2c/policy/v1"

// ScopePolicy : portée de délégation pour signer les consignes réseau.
const ScopePolicy = "policy"

const (
	MaxPolicyTTL  = 30 * 24 * time.Hour
	MaxPolicyBans = 1024
	maxPolicyBits = 40
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
	if now.Unix() >= p.Expires || p.Issued > now.Unix()+60 {
		return errors.New("trust: consignes réseau expirées ou futures")
	}
	if len(p.BannedRelays) > MaxPolicyBans || p.VolunteerBits < 0 || p.VolunteerBits > maxPolicyBits ||
		p.MaxPerFamily < 0 || p.HiddenRelayKBs < 0 || p.VolunteerPercent < 0 || p.VolunteerPercent > 100 {
		return bad
	}
	for _, k := range p.BannedRelays {
		if b, err := hex.DecodeString(k); err != nil || len(b) != ed25519.PublicKeySize || k != hex.EncodeToString(b) {
			return bad
		}
	}
	k, err := signerKey(root, p.Deleg, ScopePolicy)
	if err != nil {
		return err
	}
	u := *p
	u.Sig = ""
	return verify(k, domPolicy, u, p.Sig)
}

// live : consignes encore valides (une copie en cache peut avoir expiré).
// Toutes les méthodes ci-dessous acceptent un récepteur nil (aucune consigne).
func (p *NetworkPolicy) live() bool {
	return p != nil && time.Now().Unix() < p.Expires
}

// Banned : la clé (hex minuscule) est exclue du réseau.
func (p *NetworkPolicy) Banned(keyHex string) bool {
	return p.live() && slices.Contains(p.BannedRelays, keyHex)
}

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
	case !p.live():
		return 100
	case p.Emergency:
		return 0
	case p.VolunteerPercent > 0:
		return p.VolunteerPercent
	}
	return 100
}
