package api

import (
	"context"
	"crypto/x509"
	"errors"

	"t2c/pkg/trust"
)

// nodeRef : le proxy démarre une fois ; il suit le nœud courant du tableau de bord
// (arrêt / redémarrage de la connexion sans relancer le proxy).
type nodeRef struct{ d *Dashboard }

var errStopped = errors.New("T2C Phantom est déconnecté — relance la connexion")

func (r *nodeRef) SiteDir(ctx context.Context, domain string) (string, *trust.SiteEntry, error) {
	r.d.mu.Lock()
	n := r.d.node
	r.d.mu.Unlock()
	if n == nil {
		return "", nil, errStopped
	}
	return n.SiteDir(ctx, domain)
}

func (r *nodeRef) Resolve(ctx context.Context, domain string) (*trust.SiteEntry, error) {
	r.d.mu.Lock()
	n := r.d.node
	r.d.mu.Unlock()
	if n == nil {
		return nil, errStopped
	}
	return n.Resolve(ctx, domain)
}

func x509Pool(pemBytes []byte) (*x509.CertPool, bool) {
	p := x509.NewCertPool()
	return p, p.AppendCertsFromPEM(pemBytes)
}
