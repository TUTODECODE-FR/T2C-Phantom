package trust

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"time"
)

// VolunteerList : relais bénévoles cachés (apps T2C Phantom) connectés à un
// relais officiel, signée par la clé de CE relais (elle-même dans la liste des
// relais signée par la racine). Le serveur n'y touche pas : un serveur piraté
// ne peut pas y glisser de relais.
//
// Un relais bénévole au milieu d'un circuit ne voit ni l'adresse du membre ni
// ce qu'il demande ; en sortie (s'il l'accepte), il voit les demandes sous un
// pseudonyme jetable, jamais l'adresse (connue du seul relais d'entrée).
const domVolunteers = "t2c/volunteers/v1"

// MaxVolunteerList : taille maximale d'une liste publiée.
const MaxVolunteerList = 256

type VolunteerList struct {
	Relay      string           `json:"relay"` // clé du relais officiel qui signe (hex)
	Issued     int64            `json:"issued"`
	Volunteers []VolunteerEntry `json:"volunteers"`
	Sig        string           `json:"sig"`
}

// VolunteerEntry : un relais bénévole. Family : empreinte salée de son réseau
// (deux relais de même famille ne sont jamais dans un même circuit), jamais
// son adresse. Since : première connexion (montée en charge progressive).
type VolunteerEntry struct {
	Key    string `json:"key"`
	Family string `json:"family"`
	Since  int64  `json:"since"`
	Exit   bool   `json:"exit,omitempty"`
	// URL : adresse publique (ws:// ou wss://hôte/v1/ws) d'un bénévole qui a
	// choisi d'être aussi relais d'ENTRÉE ; publiée seulement après une sonde
	// réussie du relais officiel (l'adresse répond bien avec cette clé).
	URL string `json:"url,omitempty"`
}

// MaxVolunteerURL : longueur maximale d'une adresse publique de bénévole.
const MaxVolunteerURL = 200

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
	for _, e := range v.Volunteers {
		if _, err := ParsePublicKey(e.Key); err != nil {
			return err
		}
		if len(e.Family) > 32 || e.Since > now.Unix()+60 || len(e.URL) > MaxVolunteerURL ||
			e.URL != "" && !strings.HasPrefix(e.URL, "ws://") && !strings.HasPrefix(e.URL, "wss://") {
			return errors.New("trust: entrée de bénévole invalide")
		}
	}
	u := *v
	u.Sig = ""
	return verify(relay, domVolunteers, u, v.Sig)
}
