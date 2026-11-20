package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossSiteSafe(t *testing.T) {
	local := map[string]bool{"http://127.0.0.1:9090": true}
	cases := []struct {
		method, origin, site string
		ok                   bool
	}{
		{"GET", "https://evil.example", "cross-site", false}, // lecture de l'état : refusée
		{"GET", "", "cross-site", false},
		{"GET", "", "same-origin", true},
		{"GET", "", "", true},  // outil local (curl)
		{"POST", "", "", true}, // outil local (curl)
		{"POST", "http://127.0.0.1:9090", "same-origin", true},
		{"POST", "https://evil.example", "", false},
		{"POST", "", "cross-site", false},
		{"DELETE", "null", "", false},
		{"POST", "http://localhost.evil.example", "same-site", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, "http://127.0.0.1:9090/api/hosting", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.site != "" {
			r.Header.Set("Sec-Fetch-Site", c.site)
		}
		if got := crossSiteSafe(r, local); got != c.ok {
			t.Fatalf("%s origin=%q site=%q : %v, attendu %v", c.method, c.origin, c.site, got, c.ok)
		}
	}
}

// Fichiers de l'interface : lisibles ; /api/stop : POST seulement ; en-têtes anti-incrustation.
func TestDashboardGuard(t *testing.T) {
	local := map[string]bool{"http://127.0.0.1:9090": true}
	r := httptest.NewRequest("GET", "http://127.0.0.1:9090/app.js", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	if !crossSiteSafe(r, local) {
		t.Fatal("fichier de l'interface refusé")
	}
	d := NewDashboard()
	rec := httptest.NewRecorder()
	d.APIHandler().ServeHTTP(rec, httptest.NewRequest("GET", "http://127.0.0.1:9090/api/stop", nil))
	if rec.Code != 405 {
		t.Fatalf("GET /api/stop : %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	d.APIHandler().ServeHTTP(rec, httptest.NewRequest("POST", "http://127.0.0.1:9090/api/stop", nil))
	if rec.Code != 200 {
		t.Fatalf("POST /api/stop : %d", rec.Code)
	}
}

func TestLocalGuard(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := localGuard(9090, ok)
	for _, c := range []struct {
		host, path, site, origin string
		want                     int
	}{
		{"127.0.0.1:9090", "/api/status", "same-origin", "", 204},
		{"localhost:9090", "/", "", "", 204},
		{"evil.example:9090", "/api/status", "", "", 403}, // DNS rebinding
		{"127.0.0.1:9090", "/api/status", "cross-site", "", 403},
		{"127.0.0.1:9090", "/api/status", "", "https://evil.example", 403},
	} {
		r := httptest.NewRequest("GET", "http://"+c.host+c.path, nil)
		if c.site != "" {
			r.Header.Set("Sec-Fetch-Site", c.site)
		}
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != c.want {
			t.Fatalf("%+v : %d", c, rec.Code)
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Content-Security-Policy") != "frame-ancestors 'none'" {
			t.Fatalf("%+v : en-têtes anti-incrustation absents", c)
		}
	}
}
