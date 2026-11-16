package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// onceListener livre une seule connexion puis se ferme (Serve quitte sans tuer le handler).
type onceListener struct {
	addr net.Addr
	ch   chan net.Conn
	once sync.Once
}

func (l *onceListener) Accept() (net.Conn, error) {
	c, ok := <-l.ch
	if !ok {
		return nil, net.ErrClosed
	}
	l.Close()
	return c, nil
}
func (l *onceListener) Close() error {
	l.once.Do(func() { close(l.ch) })
	return nil
}
func (l *onceListener) Addr() net.Addr { return l.addr }

// handleConnectHTTPS : HTTPS sur *.t2c, uniquement pour les sites du catalogue
// signé (la CA locale ne signe jamais un nom hors catalogue).
func (s *Server) handleConnectHTTPS(w http.ResponseWriter, req *http.Request) {
	domain := strings.ToLower(strings.Split(req.Host, ":")[0])
	// Firefox envoie TOUT son HTTPS au proxy (télémétrie…) : on ne touche qu'aux .t2c.
	if !strings.HasSuffix(domain, ".t2c") {
		http.Error(w, "Proxy T2C: uniquement HTTPS *.t2c", http.StatusForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	_, err := s.Node.Resolve(ctx, domain)
	cancel()
	if err != nil {
		signaled, throttled := GlobalAbuse.RecordSpoof(req, domain)
		msg := "HTTPS refusé : " + err.Error()
		if signaled {
			msg += " · SIGNALÉ (répétition)"
		}
		if throttled {
			msg += " · CONNECT throttlé"
		}
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack impossible", http.StatusInternalServerError)
		return
	}
	tlsCfg, err := s.CA.LeafTLS(domain)
	if err != nil {
		http.Error(w, "certificat local: "+err.Error(), http.StatusInternalServerError)
		return
	}
	clientConn, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	_, _ = bufrw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = bufrw.Flush()

	tlsConn := tls.Server(clientConn, tlsCfg)
	ln := &onceListener{addr: clientConn.LocalAddr(), ch: make(chan net.Conn, 1)}
	ln.ch <- tlsConn
	srv := &http.Server{
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			s.serveSite(rw, r, domain)
		}),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Minute,
		ErrorLog:          nil,
	}
	if err := srv.Serve(ln); err != nil && err != net.ErrClosed && err != http.ErrServerClosed {
		fmt.Println("HTTPS", domain, ":", err)
	}
}
