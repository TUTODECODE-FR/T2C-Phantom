package sysint

import (
	"errors"
	"net/url"
	"strings"
)

// Browser : un navigateur détecté sur le poste.
type Browser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Recommended bool   `json:"recommended"` // Firefox : CA et proxy gérés sans réglage manuel
	path        string
}

// Browsers : navigateurs installés, Firefox en tête. L'entrée "default" ouvre
// le navigateur par défaut du système.
func Browsers() []Browser {
	out := []Browser{}
	for _, b := range detectBrowsers() {
		b.Recommended = b.ID == "firefox"
		out = append(out, b)
	}
	return append(out, Browser{ID: "default", Name: "Navigateur par défaut"})
}

// OpenURL ouvre url dans le navigateur id ("default" = navigateur du système).
// Seules les adresses http(s) sont acceptées.
func OpenURL(id, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(raw, "\n\r\x00") {
		return errors.New("adresse refusée")
	}
	if id == "" || id == "default" {
		return openDefault(raw)
	}
	for _, b := range detectBrowsers() {
		if b.ID == id {
			return openWith(b, raw)
		}
	}
	return errors.New("navigateur introuvable")
}
