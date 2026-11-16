package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// serveSite : sert le site .t2c depuis le dossier local (téléchargé chez les
// bénévoles via le relais, chaque morceau vérifié contre le manifeste signé).
func (s *Server) serveSite(w http.ResponseWriter, req *http.Request, domain string) {
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
	http.FileServer(http.Dir(dir)).ServeHTTP(w, req)
}
