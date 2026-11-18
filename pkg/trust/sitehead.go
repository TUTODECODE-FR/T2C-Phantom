package trust

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
)

// Tête de site : publication continue d'UN site par une clé dédiée (conteneur
// qui fabrique repo.t2c, cours.t2c…), sans sortir la clé racine. La racine
// délègue la portée « site:<id> » ; la clé ne peut rien signer d'autre.
// Le site doit déjà figurer dans le catalogue signé par la racine : la tête ne
// fait que remplacer sa racine de contenu par une version plus récente.
const domSiteHead = "t2c/sitehead/v1"

// ScopeSite : portée de délégation pour publier le site id.
func ScopeSite(id string) string { return "site:" + id }

type SiteHead struct {
	Site   string      `json:"site"` // ID du site (pas l'alias)
	Root   string      `json:"root"`
	Size   int64       `json:"size"`
	Issued int64       `json:"issued"` // strictement croissant (anti-retour arrière)
	Deleg  *Delegation `json:"deleg,omitempty"`
	Sig    string      `json:"sig"`
}

func (h *SiteHead) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	h.Deleg = d
	u := *h
	u.Sig = ""
	s, err := sign(priv, domSiteHead, u)
	h.Sig = s
	return err
}

func (h *SiteHead) Verify(root ed25519.PublicKey) error {
	if h.Deleg == nil || !isHexHash(h.Root) || h.Size <= 0 || h.Size > MaxContentSize {
		return errors.New("trust: tête de site invalide")
	}
	k, err := signerKey(root, h.Deleg, ScopeSite(h.Site))
	if err != nil {
		return err
	}
	u := *h
	u.Sig = ""
	return verify(k, domSiteHead, u, h.Sig)
}

// SignDelegated signe un manifeste avec une clé de portée site:<m.Site>.
func (m *Manifest) SignDelegated(priv ed25519.PrivateKey, d *Delegation) error {
	m.Deleg = d
	return m.Sign(priv)
}

// applyHeads remplace la racine des sites par celle de leur tête valide la plus
// récente. Une tête invalide est ignorée (le site garde sa version signée par
// la racine) : un conteneur compromis ne peut toucher qu'à son propre site.
func (c *Catalog) applyHeads(root ed25519.PublicKey) {
	best := map[string]SiteHead{}
	for _, h := range c.Heads {
		if _, ok := c.findID(h.Site); !ok || h.Verify(root) != nil {
			continue
		}
		if b, ok := best[h.Site]; !ok || h.Issued > b.Issued {
			best[h.Site] = h
		}
	}
	c.Heads = c.Heads[:0:0]
	for i := range c.Sites {
		if h, ok := best[c.Sites[i].ID]; ok {
			c.Sites[i].Root, c.Sites[i].Size = h.Root, h.Size
			c.Heads = append(c.Heads, h)
		}
	}
}

func (c *Catalog) findID(id string) (*SiteEntry, bool) {
	for i := range c.Base() {
		if c.Base()[i].ID == id {
			return &c.Base()[i], true
		}
	}
	return nil, false
}

// Base : sites tels que signés par la racine (avant les têtes).
func (c *Catalog) Base() []SiteEntry {
	if c.base != nil {
		return c.base
	}
	return c.Sites
}

// HeadIssued : date de la tête appliquée à site (0 sans tête).
func (c *Catalog) HeadIssued(site string) int64 {
	for _, h := range c.Heads {
		if h.Site == site {
			return h.Issued
		}
	}
	return 0
}

// MarshalJSON écrit toujours la forme signée (sites de base + têtes) : le
// document relu se revérifie et se réapplique à l'identique.
func (c Catalog) MarshalJSON() ([]byte, error) {
	type raw Catalog
	r := raw(c)
	if c.base != nil {
		r.Sites = c.base
	}
	return json.Marshal(r)
}
