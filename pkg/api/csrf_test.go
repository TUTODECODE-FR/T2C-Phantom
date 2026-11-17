package api

import (
	"net/http/httptest"
	"testing"
)

func TestCrossSiteSafe(t *testing.T) {
	local := map[string]bool{"http://127.0.0.1:9090": true}
	cases := []struct {
		method, origin, site string
		ok                   bool
	}{
		{"GET", "https://evil.example", "cross-site", true}, // lecture : sans effet
		{"POST", "", "", true},                              // outil local (curl)
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
