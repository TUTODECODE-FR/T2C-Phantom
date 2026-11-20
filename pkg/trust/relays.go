package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strings"
)

// RelayList : relais officiels du réseau, signés par la clé racine sur le poste
// de l'éditeur (jamais par une clé de serveur). L'application et les relais
// n'utilisent QUE ces relais pour les circuits anonymes et la fédération : un
// serveur piraté ne peut pas y glisser des relais espions.
//
// Transportée dans le catalogue (champ « relays »), hors de sa signature
// racine pour rester compatible avec les anciennes applications, mais signée
// elle-même par la racine.
const domRelays = "t2c/relays/v1"

// MaxRelays : taille maximale de la liste.
const MaxRelays = 64

type RelayList struct {
	Version uint64   `json:"version"` // strictement croissante (anti-retour arrière)
	Issued  int64    `json:"issued"`
	Relays  []string `json:"relays"` // « wss://hôte/v1/ws#k=<clé> » ou « hidden:nom#k=<clé> »
	// Revoked : clés de service (hex) révoquées par la racine (serveur piraté) :
	// leurs consignes réseau sont refusées immédiatement, sans attendre
	// l'expiration de leur délégation.
	Revoked []string `json:"revoked,omitempty"`
	Sig     string   `json:"sig"`
}

// RelayEntryKey : clé épinglée d'une entrée « …#k=<hex> » (false sans clé
// valide : l'entrée doit alors être ignorée).
func RelayEntryKey(e string) (ed25519.PublicKey, bool) {
	_, frag, ok := strings.Cut(e, "#")
	if !ok || !strings.HasPrefix(frag, "k=") {
		return nil, false
	}
	k, err := ParsePublicKey(strings.TrimPrefix(frag, "k="))
	return k, err == nil
}

func (r *RelayList) Sign(root ed25519.PrivateKey) error {
	u := *r
	u.Sig = ""
	s, err := sign(root, domRelays, u)
	r.Sig = s
	return err
}

// Verify : signature de la clé racine elle-même (aucune délégation acceptée).
func (r *RelayList) Verify(root ed25519.PublicKey) error {
	if r.Version == 0 || len(r.Relays) == 0 || len(r.Relays) > MaxRelays {
		return errors.New("trust: liste de relais invalide")
	}
	for _, e := range r.Relays {
		// ws:// n'est accepté par l'application que vers 127.0.0.1 (tests, banc local)
		if len(e) > 512 || !(strings.HasPrefix(e, "wss://") || strings.HasPrefix(e, "ws://") || strings.HasPrefix(e, "hidden:")) {
			return errors.New("trust: entrée de relais invalide")
		}
	}
	if len(r.Revoked) > MaxRelays {
		return errors.New("trust: trop de clés révoquées")
	}
	for _, k := range r.Revoked {
		if b, err := hex.DecodeString(k); err != nil || len(b) != ed25519.PublicKeySize || k != hex.EncodeToString(b) {
			return errors.New("trust: clé révoquée invalide")
		}
	}
	u := *r
	u.Sig = ""
	return verify(root, domRelays, u, r.Sig)
}
