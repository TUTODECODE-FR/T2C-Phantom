package proxy

import (
	"net/http"
	"net/url"
	"strings"
)

// SiteCSP : politique de contenu imposée à TOUTES les réponses .t2c (HTTP et
// HTTPS). Un site ne peut charger, contacter, encadrer ou viser par un
// formulaire que lui-même ou un autre site .t2c : jamais le web ouvert, qui
// recevrait l'adresse IP réelle du membre (le PAC envoie en direct tout ce
// qui n'est pas .t2c). « *.t2c » sans schéma = même schéma que la page.
// Les scripts en ligne restent permis (sites actuels) : ils ne peuvent pas
// sortir du .t2c par fetch/XHR/WebSocket (connect-src). Limites connues,
// hors de portée d'une CSP : WebRTC (STUN direct) et la navigation de la page
// entière vers une adresse du web ouvert (voir SECURITY.md).
const SiteCSP = "default-src 'self' *.t2c; " +
	"script-src 'self' 'unsafe-inline' *.t2c; " +
	"style-src 'self' 'unsafe-inline' *.t2c; " +
	"img-src 'self' data: blob: *.t2c; " +
	"font-src 'self' data: *.t2c; " +
	"media-src 'self' data: blob: *.t2c; " +
	"connect-src 'self' *.t2c; " +
	"frame-src 'self' *.t2c; " +
	"worker-src 'self'; " +
	"manifest-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"form-action 'self' *.t2c; " +
	"frame-ancestors 'self' *.t2c"

// setSiteHeaders : en-têtes de sécurité de chaque réponse .t2c (aucun
// référent envoyé, à personne).
func setSiteHeaders(h http.Header) {
	h.Set("Content-Security-Policy", SiteCSP)
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), display-capture=()")
}

// isT2CURL : adresse (Origin ou Referer) d'un site .t2c.
func isT2CURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return strings.HasSuffix(h, ".t2c") && len(h) > len(".t2c")
}

// crossSiteProbe : la requête vient d'une page qui n'est pas un site .t2c.
// Refusée AVANT toute résolution ou tout accès réseau : une page du web
// ouvert ne doit pas pouvoir savoir si le visiteur est membre (le proxy
// répond), deviner son historique au temps de réponse (page en cache ou
// non), ni faire travailler le réseau en son nom.
//
//   - Origin ou Referer présent et hors .t2c (« null » compris) : refus ;
//   - Sec-Fetch-Site « cross-site » sans Origin/Referer .t2c : refus, sauf
//     navigation de la page entière (lien suivi, adresse tapée), sans effet
//     mesurable par la page d'origine ; c'est aussi le cas d'un lien entre
//     deux sites .t2c, qui n'envoient pas de référent à une autre origine.
//
// En http://, le navigateur n'envoie pas Sec-Fetch-* : seuls Origin et
// Referer servent, et une page peut supprimer son propre référent ; la
// protection n'est donc complète qu'en https:// (voir SECURITY.md).
func crossSiteProbe(r *http.Request) bool {
	t2cOrigin := false
	if o := r.Header.Get("Origin"); o != "" {
		if !isT2CURL(o) {
			return true
		}
		t2cOrigin = true
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if !isT2CURL(ref) {
			return true
		}
		t2cOrigin = true
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" && !t2cOrigin {
		return !(r.Header.Get("Sec-Fetch-Mode") == "navigate" && r.Header.Get("Sec-Fetch-Dest") == "document")
	}
	return false
}

// refuseProbe : 403 immédiat. Pour une navigation (lien du web ouvert vers un
// .t2c), une page locale propose de continuer : le clic part alors de la
// page .t2c elle-même (même origine), donc par un geste du membre.
func refuseProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("Sec-Fetch-Dest") == "document" || (r.Header.Get("Sec-Fetch-Dest") == "" && r.Header.Get("Upgrade-Insecure-Requests") == "1") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		// chemin local seulement (« //hôte » serait un lien vers le web ouvert)
		target := (&url.URL{Path: "/" + strings.TrimLeft(r.URL.Path, "/"), RawQuery: r.URL.RawQuery}).String()
		_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>T2C Phantom</title>` +
			`<p>Lien venu d'un site hors <code>.t2c</code> : T2C Phantom ne l'ouvre pas tout seul ` +
			`(un site du web ouvert pourrait sinon savoir que tu es membre).</p>` +
			`<p><a href="` + htmlEscape(target) + `">Ouvrir quand même cette page .t2c</a></p>`))
		return
	}
	http.Error(w, "Proxy T2C : requête venue d'un site hors .t2c refusée", http.StatusForbidden)
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(s)
}
