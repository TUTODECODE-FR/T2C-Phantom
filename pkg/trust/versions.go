package trust

import (
	"crypto/ed25519"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// VersionNotice : avis de version de l'application, signé par la clé racine
// sur le poste de l'éditeur (jamais par une clé de serveur, aucune délégation).
//   - Latest : dernière version publiée (bannière discrète) ;
//   - Important : mise à jour importante (bannière visible) ;
//   - Minimum : en dessous, les relais refusent l'application.
//
// Transporté dans le catalogue (champ « versions »), hors de sa signature,
// comme RelayList.
const domVersions = "t2c/versions/v1"

const (
	MaxVersionNotes = 500 // caractères
	maxVersionURL   = 512
)

// Niveaux d'une version face à l'avis (VersionNotice.Level).
const (
	LevelOK        = "ok"
	LevelAvailable = "available"
	LevelImportant = "important"
	LevelObsolete  = "obsolete"
)

// DevVersion : version des builds hors release (jamais de bannière ; refusée
// par les relais dès qu'un minimum est fixé, sauf relais de test).
const DevVersion = "dev"

type VersionNotice struct {
	Version   uint64 `json:"version"` // strictement croissante (anti-retour arrière)
	Issued    int64  `json:"issued"`
	Latest    string `json:"latest"`
	Important string `json:"important,omitempty"`
	Minimum   string `json:"minimum,omitempty"`
	Notes     string `json:"notes,omitempty"`
	URL       string `json:"url,omitempty"` // page de la release (https uniquement)
	Sig       string `json:"sig"`
}

func (v *VersionNotice) Sign(root ed25519.PrivateKey) error {
	u := *v
	u.Sig = ""
	s, err := sign(root, domVersions, u)
	v.Sig = s
	return err
}

// Check : forme et invariants (Minimum ≤ Important ≤ Latest), sans signature.
func (v *VersionNotice) Check() error {
	bad := errors.New("trust: avis de version invalide")
	if v.Version == 0 || v.Issued <= 0 {
		return bad
	}
	latest, err := ParseSemver(v.Latest)
	if err != nil {
		return bad
	}
	top := latest
	for _, s := range []string{v.Important, v.Minimum} { // décroissant
		if s == "" {
			continue
		}
		x, err := ParseSemver(s)
		if err != nil || x.Compare(top) > 0 {
			return bad
		}
		top = x
	}
	if !utf8.ValidString(v.Notes) || utf8.RuneCountInString(v.Notes) > MaxVersionNotes {
		return bad
	}
	if v.URL != "" {
		u, err := url.Parse(v.URL)
		if err != nil || len(v.URL) > maxVersionURL || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return bad
		}
	}
	return nil
}

// Verify : forme puis signature de la clé racine elle-même.
func (v *VersionNotice) Verify(root ed25519.PublicKey) error {
	if err := v.Check(); err != nil {
		return err
	}
	u := *v
	u.Sig = ""
	return verify(root, domVersions, u, v.Sig)
}

// Level : situation de la version cur de l'application. Une version « dev »
// ou illisible n'affiche jamais rien (LevelOK). v nil = aucun avis.
func (v *VersionNotice) Level(cur string) string {
	if v == nil {
		return LevelOK
	}
	c, err := ParseSemver(cur)
	if err != nil {
		return LevelOK
	}
	below := func(s string) bool {
		x, err := ParseSemver(s)
		return s != "" && err == nil && c.Compare(x) < 0
	}
	switch {
	case below(v.Minimum):
		return LevelObsolete
	case below(v.Important):
		return LevelImportant
	case below(v.Latest):
		return LevelAvailable
	}
	return LevelOK
}

// Refuses (relais) : vrai si la version annoncée cur est sous le minimum.
// Sans minimum, rien n'est refusé. Avec un minimum :
//   - champ absent (applications d'avant l'avis de version) : refusé ;
//   - « dev » : refusé, sauf allowDev (relais de test) ;
//   - version illisible : refusée.
func (v *VersionNotice) Refuses(cur string, allowDev bool) bool {
	if v == nil || v.Minimum == "" {
		return false
	}
	if cur == DevVersion {
		return !allowDev
	}
	c, err := ParseSemver(cur)
	if err != nil {
		return true
	}
	m, err := ParseSemver(v.Minimum)
	return err == nil && c.Compare(m) < 0
}

// Semver : version « MAJEUR.MINEUR.CORRECTIF » (préfixe « v » toléré).
type Semver [3]uint32

// ParseSemver : strict — trois nombres décimaux sans zéro initial, rien d'autre
// (ni suffixe « -rc1 », ni espace, ni signe).
func ParseSemver(s string) (Semver, error) {
	var v Semver
	bad := errors.New("trust: version invalide")
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(s) > 32 || len(parts) != 3 {
		return v, bad
	}
	for i, p := range parts {
		if p == "" || len(p) > 9 || (len(p) > 1 && p[0] == '0') {
			return v, bad
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return v, bad
			}
		}
		n, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return v, bad
		}
		v[i] = uint32(n)
	}
	return v, nil
}

// Compare : -1, 0 ou 1.
func (a Semver) Compare(b Semver) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

func (a Semver) String() string {
	return strconv.FormatUint(uint64(a[0]), 10) + "." + strconv.FormatUint(uint64(a[1]), 10) + "." + strconv.FormatUint(uint64(a[2]), 10)
}

// CompareSemver : compare deux versions textuelles (erreur si l'une est illisible).
func CompareSemver(a, b string) (int, error) {
	x, err := ParseSemver(a)
	if err != nil {
		return 0, err
	}
	y, err := ParseSemver(b)
	if err != nil {
		return 0, err
	}
	return x.Compare(y), nil
}
