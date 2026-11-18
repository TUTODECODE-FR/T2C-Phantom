package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"t2c/pkg/trust"
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
	if req.URL.Path == SitesPath {
		s.serveSitesList(w, req)
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
	defer cancel()
	var hfs http.FileSystem
	var e *trust.SiteEntry
	var err error
	if lz, ok := s.Node.(lazySites); ok {
		// chargement à la demande : seulement les morceaux de la page demandée
		hfs, e, err = lz.SiteFS(ctx, domain)
	} else {
		var dir string
		dir, e, err = s.Node.SiteDir(ctx, domain)
		hfs = http.Dir(dir)
	}
	if err != nil {
		http.Error(w, "Site inaccessible : "+err.Error(), http.StatusBadGateway)
		fmt.Println("site", domain, ":", err)
		return
	}
	w.Header().Set("X-T2C-Root", e.Root)
	w.Header().Set("X-T2C-Verified", "manifest-signed")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if st, ok := hfs.(streamer); ok && !hiddenPath(req.URL.Path) {
		if rc, size, ok := st.Stream(req.URL.Path); ok {
			// gros fichier : envoyé au fil de l'eau, taille annoncée (barre de progression)
			defer rc.Close()
			ct := mime.TypeByExtension(path.Ext(req.URL.Path))
			if ct == "" {
				ct = "application/octet-stream"
			}
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
			if req.Method == http.MethodHead {
				return
			}
			_, _ = io.Copy(w, io.LimitReader(rc, size))
			return
		}
	}
	http.FileServer(siteFS{hfs}).ServeHTTP(w, req)
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

// SitesPath : liste des sites du catalogue signé, servie par l'application
// elle-même sur chaque site .t2c (une page comme tutodecode.t2c peut ainsi
// afficher tous les sites, à jour, sans rien demander au réseau en plus).
const SitesPath = "/.well-known/t2c/sites.json"

type streamer interface {
	Stream(name string) (io.ReadCloser, int64, bool)
}

// hiddenPath : un élément du chemin commence par « . » (jamais servi).
func hiddenPath(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return strings.Contains(p, "..")
}

type lazySites interface {
	SiteFS(ctx context.Context, domain string) (http.FileSystem, *trust.SiteEntry, error)
}

type catalogSource interface {
	Catalog(ctx context.Context) (*trust.Catalog, error)
}

type siteInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Title     string `json:"title"`
	Publisher string `json:"publisher,omitempty"`
	Size      int64  `json:"size"`
}

func (s *Server) serveSitesList(w http.ResponseWriter, req *http.Request) {
	src, ok := s.Node.(catalogSource)
	if !ok {
		http.NotFound(w, req)
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()
	cat, err := src.Catalog(ctx)
	if err != nil {
		http.Error(w, "catalogue indisponible", http.StatusBadGateway)
		return
	}
	out := struct {
		Version uint64     `json:"version"`
		Sites   []siteInfo `json:"sites"`
	}{Version: cat.Version, Sites: []siteInfo{}}
	for _, e := range cat.Sites {
		a := e.Alias
		if a == "" {
			a = e.ID
		}
		out.Sites = append(out.Sites, siteInfo{ID: e.ID, Address: a + ".t2c", Title: e.Title, Publisher: e.Publisher, Size: e.Size})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-T2C-Verified", "catalog-signed")
	_ = json.NewEncoder(w).Encode(out)
}
