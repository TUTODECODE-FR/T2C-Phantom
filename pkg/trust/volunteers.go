package trust

import (
	"crypto/ed25519"
	"errors"
	"time"
)

// VolunteerList : relais bénévoles cachés (apps T2C Phantom) connectés à un
// relais officiel, signée par la clé de CE relais (elle-même dans la liste des
// relais signée par la racine). Le serveur n'y touche pas : un serveur piraté
// ne peut pas y glisser de relais.
//
// Ces relais ne servent qu'au MILIEU des circuits : ils ne voient ni l'adresse
// du client (connue du relais d'entrée) ni ce qui est demandé (relais de sortie).
const domVolunteers = "t2c/volunteers/v1"

// MaxVolunteerList : taille maximale d'une liste publiée.
const MaxVolunteerList = 256

type VolunteerList struct {
	Relay      string   `json:"relay"` // clé du relais officiel qui signe (hex)
	Issued     int64    `json:"issued"`
	Volunteers []string `json:"volunteers"` // clés publiques des relais bénévoles (hex)
	Sig        string   `json:"sig"`
}

func (v *VolunteerList) Sign(relay ed25519.PrivateKey) error {
	u := *v
	u.Sig = ""
	s, err := sign(relay, domVolunteers, u)
	v.Sig = s
	return err
}

// Verify : signée par relay, récente (moins de 10 minutes), de taille raisonnable.
func (v *VolunteerList) Verify(relay ed25519.PublicKey, now time.Time) error {
	if len(v.Volunteers) > MaxVolunteerList || now.Unix()-v.Issued > 600 || v.Issued-now.Unix() > 60 {
		return errors.New("trust: liste de bénévoles périmée ou invalide")
	}
	for _, k := range v.Volunteers {
		if _, err := ParsePublicKey(k); err != nil {
			return err
		}
	}
	u := *v
	u.Sig = ""
	return verify(relay, domVolunteers, u, v.Sig)
}
