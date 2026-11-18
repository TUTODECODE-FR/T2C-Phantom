package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"slices"
	"time"
)

// Délégation : la clé racine reste hors ligne (poste de l'éditeur). Elle signe
// une délégation courte qui autorise une clé de service (sur le serveur) à
// signer UNIQUEMENT ce que ses portées nomment : liste des bénévoles, tickets,
// ou un site précis (« site:<id> », voir SiteHead). Le catalogue exige
// toujours la clé racine : une clé déléguée ne peut ni créer de site ni
// toucher à un site qui ne lui est pas confié.
const domDelegation = "t2c/delegation/v1"

const (
	ScopeHosts  = "hosts"
	ScopeTicket = "ticket"
)

// MaxDelegation : durée maximale acceptée d'une délégation.
const MaxDelegation = 180 * 24 * time.Hour

type Delegation struct {
	Key     string   `json:"key"` // clé publique de service (hex)
	Scopes  []string `json:"scopes"`
	Issued  int64    `json:"issued"`
	Expires int64    `json:"expires"`
	Sig     string   `json:"sig"`
}

var ErrDelegation = errors.New("trust: délégation invalide ou expirée")

// NewDelegation prépare une délégation signée par la clé racine.
func NewDelegation(root ed25519.PrivateKey, service ed25519.PublicKey, ttl time.Duration, scopes ...string) (*Delegation, error) {
	if ttl <= 0 || ttl > MaxDelegation {
		return nil, errors.New("trust: durée de délégation hors limites")
	}
	now := time.Now()
	d := &Delegation{Key: hex.EncodeToString(service), Scopes: scopes, Issued: now.Unix(), Expires: now.Add(ttl).Unix()}
	u := *d
	u.Sig = ""
	s, err := sign(root, domDelegation, u)
	d.Sig = s
	return d, err
}

// ServiceKey vérifie la délégation (signée par root, portée, validité) et
// renvoie la clé de service autorisée.
func (d *Delegation) ServiceKey(root ed25519.PublicKey, scope string, now time.Time) (ed25519.PublicKey, error) {
	if d == nil || !slices.Contains(d.Scopes, scope) || now.Unix() >= d.Expires ||
		d.Expires-d.Issued > int64(MaxDelegation/time.Second) {
		return nil, ErrDelegation
	}
	u := *d
	u.Sig = ""
	if err := verify(root, domDelegation, u, d.Sig); err != nil {
		return nil, ErrDelegation
	}
	k, err := ParsePublicKey(d.Key)
	if err != nil {
		return nil, ErrDelegation
	}
	return k, nil
}

// signerKey : clé qui doit avoir signé un document de portée scope.
// Sans délégation : la racine elle-même.
func signerKey(root ed25519.PublicKey, d *Delegation, scope string) (ed25519.PublicKey, error) {
	if d == nil {
		return root, nil
	}
	return d.ServiceKey(root, scope, time.Now())
}
