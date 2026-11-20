package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

// RelayRelease : version du relais (t2c-relay) à installer, signée par une clé
// de MISE À JOUR dédiée, hors ligne, déléguée par la racine avec la seule
// portée « relay-update » (la racine elle-même est aussi acceptée). Elle ne
// signe rien d'autre : ni catalogue, ni site, ni consignes.
//
// Les binaires eux-mêmes sont des fichiers d'un site publié par la RACINE
// (défaut « relais ») : le relais les télécharge par le chemin habituel
// (manifeste signé, morceaux vérifiés) puis vérifie SHA256 et taille contre
// cet avis. Il faut donc les deux clés pour faire tourner du code nouveau.
//
// Transportée dans le catalogue (champ « relay_release »), hors de sa
// signature, comme VersionNotice ; mise à nil par Catalog.Verify si invalide.
const domRelayRelease = "t2c/relay-release/v1"

// ScopeRelayUpdate : portée de délégation de la clé de mise à jour des relais.
const ScopeRelayUpdate = "relay-update"

const (
	MaxReleaseBinary   = 100 << 20 // octets par binaire
	maxReleaseBins     = 4
	maxReleasePath     = 256
	MaxReleaseNotes    = 500 // caractères
	DefaultReleaseSite = "relais"
)

// ReleasePlatforms : plateformes acceptées (GOOS/GOARCH).
var ReleasePlatforms = []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64"}

// ReleaseBinary : un binaire, fichier Path dans le site Site de la release.
type ReleaseBinary struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type RelayRelease struct {
	Version  string                   `json:"version"` // MAJEUR.MINEUR.CORRECTIF
	Issued   int64                    `json:"issued"`
	Site     string                   `json:"site"`
	Binaries map[string]ReleaseBinary `json:"binaries"` // « linux/amd64 » → binaire
	// Percent : déploiement progressif, part des relais concernés (1..100),
	// choisie de façon stable par clé de relais et version (voir Includes).
	Percent int         `json:"percent"`
	Notes   string      `json:"notes,omitempty"`
	Deleg   *Delegation `json:"deleg,omitempty"`
	Sig     string      `json:"sig"`
}

// SignDelegated : signe avec la clé de mise à jour (d sa délégation) ou la
// racine (d nil).
func (r *RelayRelease) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	r.Deleg = d
	u := *r
	u.Sig = ""
	s, err := sign(priv, domRelayRelease, u)
	r.Sig = s
	return err
}

// Check : forme seule, sans signature.
func (r *RelayRelease) Check(now time.Time) error {
	bad := errors.New("trust: release de relais invalide")
	if _, err := ParseSemver(r.Version); err != nil || r.Issued <= 0 || r.Issued > now.Unix()+60 {
		return bad
	}
	if !validSiteID(r.Site) || r.Percent < 1 || r.Percent > 100 {
		return bad
	}
	if len(r.Binaries) == 0 || len(r.Binaries) > maxReleaseBins {
		return bad
	}
	for p, b := range r.Binaries {
		ok := false
		for _, w := range ReleasePlatforms {
			ok = ok || p == w
		}
		if !ok || b.Size <= 0 || b.Size > MaxReleaseBinary || !isHexHash(b.SHA256) || strings.ToLower(b.SHA256) != b.SHA256 {
			return bad
		}
		if b.Path == "" || len(b.Path) > maxReleasePath || strings.HasPrefix(b.Path, "/") || strings.Contains(b.Path, "\\") ||
			path.Clean(b.Path) != b.Path || b.Path == ".." || strings.HasPrefix(b.Path, "../") || !utf8.ValidString(b.Path) {
			return bad
		}
	}
	if !utf8.ValidString(r.Notes) || utf8.RuneCountInString(r.Notes) > MaxReleaseNotes {
		return bad
	}
	return nil
}

// Verify : forme puis signature (racine, ou clé déléguée « relay-update »
// valide à now).
func (r *RelayRelease) Verify(root ed25519.PublicKey, now time.Time) error {
	if err := r.Check(now); err != nil {
		return err
	}
	k := root
	if r.Deleg != nil {
		var err error
		if k, err = r.Deleg.ServiceKey(root, ScopeRelayUpdate, now); err != nil {
			return err
		}
	}
	u := *r
	u.Sig = ""
	return verify(k, domRelayRelease, u, r.Sig)
}

// Signer : « root » ou la clé de mise à jour (hex), pour la révocation.
func (r *RelayRelease) Signer() string {
	if r.Deleg == nil {
		return "root"
	}
	return r.Deleg.Key
}

// Includes : le relais de clé key fait partie du déploiement progressif.
// sha256(clé ‖ version)[0:2] mod 100 < Percent : stable pour un relais et une
// version, indépendant d'une version à l'autre.
func (r *RelayRelease) Includes(key ed25519.PublicKey) bool {
	h := sha256.Sum256(append(append([]byte{}, key...), r.Version...))
	return int(binary.BigEndian.Uint16(h[:2]))%100 < r.Percent
}

// validSiteID : a-z, 0-9, « - », 1 à 63 caractères (comme l'origine).
func validSiteID(id string) bool {
	if id == "" || len(id) > 63 || id[0] == '-' {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}
