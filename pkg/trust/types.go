package trust

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Domaines de signature : une signature de catalogue ne peut pas être
// rejouée comme signature de ticket, etc.
const (
	domCatalog  = "t2c/catalog/v1"
	domManifest = "t2c/manifest/v1"
	domHosts    = "t2c/hosts/v1"
	domTicket   = "t2c/ticket/v1"
)

var ErrBadSignature = errors.New("trust: signature invalide")

// SiteEntry : un site publié, pointant vers la racine de son manifeste.
type SiteEntry struct {
	ID        string `json:"id"`
	Alias     string `json:"alias"`
	Title     string `json:"title"`
	Publisher string `json:"publisher,omitempty"`
	Root      string `json:"root"` // racine du manifeste (hex sha256)
	Size      int64  `json:"size"`
}

// Catalog : liste officielle des sites. Version strictement croissante.
type Catalog struct {
	Version uint64      `json:"version"`
	Issued  int64       `json:"issued"`
	Sites   []SiteEntry `json:"sites"`
	Sig     string      `json:"sig"`
	// Heads : têtes de site (hors signature racine, chacune signée par sa clé
	// déléguée). Appliquées par Verify ; Sites devient alors la vue effective.
	Heads []SiteHead `json:"heads,omitempty"`
	// Relays : relais officiels signés par la racine (hors signature du
	// catalogue, voir RelayList). Mis à nil par Verify s'il est invalide.
	Relays *RelayList `json:"relays,omitempty"`
	// Policy : consignes réseau (hors signature du catalogue, voir
	// NetworkPolicy). Mises à nil par Verify si invalides ; expirées mais
	// authentiques, gardées (exclusions et urgence seulement : échec fermé).
	Policy *NetworkPolicy `json:"policy,omitempty"`
	// Versions : avis de version signé par la racine (hors signature du
	// catalogue, voir VersionNotice). Mis à nil par Verify s'il est invalide.
	Versions *VersionNotice `json:"versions,omitempty"`
	// RelayRelease : version du relais à installer, signée par la clé de mise
	// à jour (hors signature du catalogue, voir RelayRelease). Mise à nil par
	// Verify si invalide ou signée par une clé révoquée.
	RelayRelease *RelayRelease `json:"relay_release,omitempty"`
	base         []SiteEntry
}

// HostEntry : un bénévole approuvé. Pas d'IP : seulement l'ID de pair libp2p,
// les adresses se résolvent via la DHT.
type HostEntry struct {
	Peer    string `json:"peer"`
	Site    string `json:"site"`
	Root    string `json:"root"`
	Score   int    `json:"score"`
	Expires int64  `json:"expires"`
}

// HostsList : bénévoles de confiance + pairs bannis + nœuds d'infrastructure.
type HostsList struct {
	Version   uint64      `json:"version"`
	Issued    int64       `json:"issued"`
	Hosts     []HostEntry `json:"hosts"`
	Banned    []string    `json:"banned"`
	Bootstrap []string    `json:"bootstrap"`       // multiaddrs infra (VPS) uniquement
	Deleg     *Delegation `json:"deleg,omitempty"` // signée par une clé de service déléguée
	Sig       string      `json:"sig"`
}

// Ticket : autorisation signée pour qu'un pair héberge une version d'un site.
type Ticket struct {
	Peer    string      `json:"peer"`
	Site    string      `json:"site"`
	Root    string      `json:"root"`
	Expires int64       `json:"expires"`
	Deleg   *Delegation `json:"deleg,omitempty"`
	Sig     string      `json:"sig"`
}

func (t *Ticket) Valid(now time.Time) bool { return now.Unix() < t.Expires }

// --- signature générique -------------------------------------------------

func payload(domain string, v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append([]byte(domain+"\n"), b...), nil
}

func sign(priv ed25519.PrivateKey, domain string, unsigned any) (string, error) {
	p, err := payload(domain, unsigned)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ed25519.Sign(priv, p)), nil
}

func verify(pub ed25519.PublicKey, domain string, unsigned any, sigHex string) error {
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrBadSignature
	}
	p, err := payload(domain, unsigned)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, p, sig) {
		return ErrBadSignature
	}
	return nil
}

// Chaque type signe une copie de lui-même avec Sig vide.

func (c *Catalog) Sign(priv ed25519.PrivateKey) error {
	c.base = nil
	u := *c
	u.Sig, u.Heads, u.Relays, u.Policy, u.Versions, u.RelayRelease = "", nil, nil, nil, nil, nil
	s, err := sign(priv, domCatalog, u)
	c.Sig = s
	return err
}

func (c *Catalog) Verify(pub ed25519.PublicKey) error {
	u := *c
	u.Sig, u.Heads, u.Relays, u.Policy, u.Versions, u.RelayRelease, u.base = "", nil, nil, nil, nil, nil, nil
	u.Sites = c.Base()
	if err := verify(pub, domCatalog, u, c.Sig); err != nil {
		return err
	}
	for _, s := range u.Sites {
		if !isHexHash(s.Root) || s.ID == "" {
			return fmt.Errorf("trust: entrée de catalogue invalide %q", s.ID)
		}
	}
	if c.base == nil {
		c.base = slices.Clone(c.Sites)
	}
	c.Sites = slices.Clone(c.base)
	c.applyHeads(pub)
	if c.Relays != nil && c.Relays.Verify(pub) != nil {
		c.Relays = nil // liste invalide : ignorée, jamais utilisée
	}
	if c.Policy != nil {
		// consignes invalides ou d'une clé révoquée : ignorées ; expirées mais
		// authentiques : gardées (exclusions et urgence seulement, voir live)
		if err := c.Policy.Verify(pub, time.Now()); err != nil && !errors.Is(err, ErrPolicyExpired) || c.Relays.revoked(c.Policy.Signer()) {
			c.Policy = nil
		}
	}
	if c.Versions != nil && c.Versions.Verify(pub) != nil {
		c.Versions = nil // avis invalide : ignoré
	}
	if c.RelayRelease != nil && (c.RelayRelease.Verify(pub, time.Now()) != nil || c.Relays.revoked(c.RelayRelease.Signer())) {
		c.RelayRelease = nil // release invalide ou d'une clé révoquée : ignorée
	}
	return nil
}

// Find renvoie le site par ID ou alias (sans le suffixe .t2c).
func (c *Catalog) Find(idOrAlias string) (*SiteEntry, bool) {
	for i := range c.Sites {
		if c.Sites[i].ID == idOrAlias || c.Sites[i].Alias == idOrAlias {
			return &c.Sites[i], true
		}
	}
	return nil, false
}

func (h *HostsList) Sign(priv ed25519.PrivateKey) error {
	u := *h
	u.Sig = ""
	s, err := sign(priv, domHosts, u)
	h.Sig = s
	return err
}

// SignDelegated : signature par la clé de service, délégation jointe.
func (h *HostsList) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	h.Deleg = d
	return h.Sign(priv)
}

// Verify : root = clé racine. Accepte la racine ou une clé de service déléguée (portée hosts).
func (h *HostsList) Verify(root ed25519.PublicKey) error {
	k, err := signerKey(root, h.Deleg, ScopeHosts)
	if err != nil {
		return err
	}
	u := *h
	u.Sig = ""
	return verify(k, domHosts, u, h.Sig)
}

// IsBanned : pair présent dans la liste des bannis.
func (h *HostsList) IsBanned(peerID string) bool {
	for _, b := range h.Banned {
		if b == peerID {
			return true
		}
	}
	return false
}

// HostScore : score du bénévole pour ce site (0, false s'il n'est pas listé ou expiré).
func (h *HostsList) HostScore(peerID, root string, now time.Time) (int, bool) {
	for _, e := range h.Hosts {
		if e.Peer == peerID && e.Root == root && now.Unix() < e.Expires {
			return e.Score, true
		}
	}
	return 0, false
}

func (t *Ticket) Sign(priv ed25519.PrivateKey) error {
	u := *t
	u.Sig = ""
	s, err := sign(priv, domTicket, u)
	t.Sig = s
	return err
}

func (t *Ticket) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	t.Deleg = d
	return t.Sign(priv)
}

// Verify : root = clé racine. Accepte la racine ou une clé de service déléguée (portée ticket).
func (t *Ticket) Verify(root ed25519.PublicKey) error {
	k, err := signerKey(root, t.Deleg, ScopeTicket)
	if err != nil {
		return err
	}
	u := *t
	u.Sig = ""
	return verify(k, domTicket, u, t.Sig)
}

func isHexHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
