package proxy

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// serveSite : sert le site .t2c depuis le dossier local (téléchargé chez les
// bénévoles via le relais, chaque morceau vérifié contre le manifeste signé).
func (s *Server) serveSite(w http.ResponseWriter, req *http.Request, domain string) {
	// sites statiques : lecture seule
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
	defer cancel()
	dir, e, err := s.Node.SiteDir(ctx, domain)
	if err != nil {
		http.Error(w, "Site inaccessible : "+err.Error(), http.StatusBadGateway)
		fmt.Println("site", domain, ":", err)
		return
	}
	w.Header().Set("X-T2C-Root", e.Root)
	w.Header().Set("X-T2C-Verified", "manifest-signed")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.FileServer(siteFS{http.Dir(dir)}).ServeHTTP(w, req)
}

// siteFS : ne sert ni les fichiers cachés ou internes (« .t2c-ok »…), ni la
// liste d'un dossier sans page d'index.
type siteFS struct{ fs http.FileSystem }

func (f siteFS) Open(name string) (http.File, error) {
	for _, seg := range strings.Split(path.Clean("/"+name), "/") {
		if strings.HasPrefix(seg, ".") {
			return nil, fs.ErrNotExist
		}
	}
	file, err := f.fs.Open(name)
	if err != nil {
		return nil, err
	}
	if st, err := file.Stat(); err == nil && st.IsDir() {
		idx, err := f.fs.Open(strings.TrimSuffix(name, "/") + "/index.html")
		if err != nil {
			file.Close()
			return nil, fs.ErrNotExist
		}
		idx.Close()
	}
	return file, nil
}
