package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"t2c/pkg/trust"
)

// countingSites : compte les accès au réseau (résolution, téléchargement).
type countingSites struct{ calls atomic.Int32 }

func (c *countingSites) SiteDir(context.Context, string) (string, *trust.SiteEntry, error) {
	c.calls.Add(1)
	return "", nil, context.Canceled
}
func (c *countingSites) Resolve(context.Context, string) (*trust.SiteEntry, error) {
	c.calls.Add(1)
	return nil, context.Canceled
}

// Toute réponse .t2c porte la CSP stricte, sans origine tierce.
func TestSiteSecurityHeaders(t *testing.T) {
	s := &Server{Node: fakeSites{&trust.Catalog{Version: 1}}}
	for _, u := range []string{"http://tutodecode.t2c" + SitesPath, "https://tutodecode.t2c/"} {
		rec := httptest.NewRecorder()
		s.serveSite(rec, httptest.NewRequest("GET", u, nil), "tutodecode.t2c")
		h := rec.Header()
		csp := h.Get("Content-Security-Policy")
		if h.Get("Referrer-Policy") != "no-referrer" || h.Get("X-Content-Type-Options") != "nosniff" || csp == "" {
			t.Fatalf("%s : en-têtes manquants %v", u, h)
		}
		for _, d := range []string{"default-src", "connect-src", "img-src", "font-src", "media-src", "frame-src", "form-action", "base-uri", "object-src 'none'"} {
			if !strings.Contains(csp, d) {
				t.Fatalf("CSP sans %s : %s", d, csp)
			}
		}
		// aucune source autre que 'self', *.t2c, data:, blob:, 'none', 'unsafe-inline'
		for _, dir := range strings.Split(csp, ";") {
			for _, src := range strings.Fields(dir)[1:] {
				switch src {
				case "'self'", "*.t2c", "data:", "blob:", "'none'", "'unsafe-inline'":
				default:
					t.Fatalf("source non autorisée %q dans %q", src, dir)
				}
			}
		}
	}
	// réponse d'erreur (site inaccessible) : mêmes en-têtes
	rec := httptest.NewRecorder()
	s.serveSite(rec, httptest.NewRequest("GET", "http://x.t2c/", nil), "x.t2c")
	if rec.Code != http.StatusBadGateway || rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("erreur sans CSP : %d", rec.Code)
	}
}

// Une page hors .t2c qui sonde le proxy reçoit 403 immédiatement, sans
// qu'aucune résolution ni aucun téléchargement ne soit tenté.
func TestCrossSiteProbeRefused(t *testing.T) {
	cases := []struct {
		name string
		hdr  map[string]string
		deny bool
	}{
		{"image depuis le web ouvert (https)", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "no-cors", "Sec-Fetch-Dest": "image"}, true},
		{"fetch depuis le web ouvert", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "cors", "Origin": "https://evil.example"}, true},
		{"iframe depuis le web ouvert", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "iframe"}, true},
		{"image en http avec référent externe", map[string]string{"Referer": "http://evil.example/page"}, true},
		{"origine opaque", map[string]string{"Origin": "null"}, true},
		{"faux .t2c dans le chemin", map[string]string{"Referer": "https://evil.example/x.t2c"}, true},
		{"lien du web ouvert", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document", "Referer": "https://forum.example/"}, true},
		{"adresse tapée", map[string]string{"Sec-Fetch-Site": "none", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}, false},
		{"même site", map[string]string{"Sec-Fetch-Site": "same-origin", "Sec-Fetch-Dest": "image"}, false},
		{"lien entre deux .t2c", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}, false},
		{"fetch CORS depuis un autre .t2c", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "cors", "Origin": "https://repo.t2c"}, false},
		{"outil local (curl)", nil, false},
	}
	for _, c := range cases {
		cs := &countingSites{}
		s := &Server{Node: cs}
		req := httptest.NewRequest("GET", "https://tutodecode.t2c/page.html", nil)
		for k, v := range c.hdr {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		s.serveSite(rec, req, "tutodecode.t2c")
		denied := rec.Code == http.StatusForbidden
		if denied != c.deny {
			t.Fatalf("%s : HTTP %d", c.name, rec.Code)
		}
		if c.deny && cs.calls.Load() != 0 {
			t.Fatalf("%s : le réseau a été sollicité avant le refus", c.name)
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("%s : refus sans CSP", c.name)
		}
	}
	// la page de refus d'un lien ne mène qu'à la même origine
	s := &Server{Node: &countingSites{}}
	req := httptest.NewRequest("GET", "https://tutodecode.t2c//evil.example/", nil)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Referer", "https://evil.example/")
	rec := httptest.NewRecorder()
	s.serveSite(rec, req, "tutodecode.t2c")
	if body := rec.Body.String(); rec.Code != 403 || strings.Contains(body, `href="//`) || !strings.Contains(body, `href="/evil.example/"`) {
		t.Fatalf("page de refus : %d %s", rec.Code, body)
	}
}
