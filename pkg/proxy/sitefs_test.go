package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSiteFS(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("accueil"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".t2c-ok"), []byte("racine"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "img"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "img", "a.png"), []byte("png"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("x"), 0o644)
	h := http.FileServer(siteFS{http.Dir(dir)})
	for path, want := range map[string]int{
		"/":               200,
		"/index.html":     301, // FileServer redirige vers « / »
		"/img/a.png":      200,
		"/img/":           404, // pas de liste de dossier
		"/.t2c-ok":        404,
		"/.git/config":    404,
		"/img/../.t2c-ok": 404,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Fatalf("%s : HTTP %d, attendu %d", path, rec.Code, want)
		}
	}
}
