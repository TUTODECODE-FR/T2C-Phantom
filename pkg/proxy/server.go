package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"t2c/pkg/node"
	"t2c/pkg/proxy/assohttps"
	"t2c/pkg/trust"
)

// LocalCAPEMPath : CA locale à installer dans le navigateur (/api/asso-ca.crt).
func LocalCAPEMPath() string {
	return filepath.Join(assohttps.Dir(), "asso-ca.crt")
}

// Sites : ce dont le proxy a besoin (implémenté par *node.Node).
type Sites interface {
	SiteDir(ctx context.Context, domain string) (string, *trust.SiteEntry, error)
	Resolve(ctx context.Context, domain string) (*trust.SiteEntry, error)
}

var _ Sites = (*node.Node)(nil)

// Server : proxy HTTP local (127.0.0.1) qui sert les sites .t2c depuis le réseau.
type Server struct {
	Node Sites
	CA   *assohttps.CA
}

// StartProxyServer démarre le proxy sur listenAddr (bloquant).
func StartProxyServer(n Sites, listenAddr string) error {
	s := &Server{Node: n}
	ca, err := assohttps.LoadOrCreate()
	if err != nil {
		fmt.Println("CA locale .t2c:", err)
	} else {
		s.CA = ca
		fmt.Println("🔐 CA locale:", ca.CertPEMPath())
		if os.Getenv("T2C_NO_BROWSER_SETUP") != "" {
			// tests / démo : ne jamais toucher au navigateur ni au Bureau de l'utilisateur
		} else if home, herr := os.UserHomeDir(); herr == nil {
			ca.EnsureTrusted()
			dest := filepath.Join(home, "Desktop", "tutodecode-ca.crt")
			if err := ca.ExportInstallFile(dest); err == nil {
				fmt.Println("📄 CA à installer:", dest)
			}
		}
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Minute, // premier téléchargement d'un site
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("🚀 Proxy T2C sur http://%s\n", listenAddr)
	return srv.ListenAndServe()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if ok, reason := GlobalAbuse.ObserveRequest(req); !ok {
		http.Error(w, "Proxy T2C: "+reason, http.StatusTooManyRequests)
		return
	}
	if req.Method == http.MethodConnect {
		if s.refuseWrongCert(w) {
			return
		}
		s.handleConnectHTTPS(w, req)
		return
	}
	domain := strings.ToLower(strings.Split(req.Host, ":")[0])
	if strings.HasSuffix(domain, ".tdc") {
		http.Error(w, "Proxy T2C: .tdc désigne un cours, pas un site. Utilise .t2c", http.StatusForbidden)
		return
	}
	if !strings.HasSuffix(domain, ".t2c") {
		http.Error(w, "Proxy T2C: seuls les domaines .t2c sont supportés", http.StatusForbidden)
		return
	}
	s.serveSite(w, req, domain)
}
