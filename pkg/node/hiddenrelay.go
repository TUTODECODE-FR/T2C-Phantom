package node

// Relais caché bénévole intégré à l'application (activé par défaut,
// désactivable dans le bénévolat). Il n'écoute rien et ne publie aucune
// adresse : c'est lui qui se connecte aux relais officiels. Il ne sert qu'au
// MILIEU des circuits des autres membres : il ne voit ni leur adresse (connue du
// relais d'entrée) ni ce qu'ils demandent (relais de sortie), seulement des
// cellules chiffrées de taille fixe. Sa clé est distincte de l'identité de
// l'utilisateur : impossible de relier les deux.

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"t2c-relay/pkg/broker"
	"t2c/pkg/peerid"
)

const (
	defaultRelayKBs  = 128              // ~1 Mbit/s
	hiddenRelayEvery = 30 * time.Second // vérifie choix et liste des relais
)

type hiddenRelay struct {
	b        *broker.Broker
	cancel   context.CancelFunc
	official []string
	kbs      int64
	exit     bool
	entry    string       // relais d'entrée demandé (écoute|adresse), vide sinon
	srv      *http.Server // écoute publique (relais d'entrée), nil sinon
}

// officialMeshRelays : relais officiels joignables (liste signée par la racine).
func (n *Node) officialMeshRelays() []string {
	var out []string
	for _, rk := range n.relayKeys() {
		if strings.HasPrefix(rk.URL, "wss://") || strings.HasPrefix(rk.URL, "ws://127.0.0.1") {
			out = append(out, rk.URL+"#k="+hexKey(rk.Key))
		}
	}
	return out
}

func (n *Node) hiddenRelayLoop(ctx context.Context) {
	defer n.stopHiddenRelay()
	every := hiddenRelayEvery
	if n.cfg.HiddenRelayEvery > 0 {
		every = n.cfg.HiddenRelayEvery
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		n.syncHiddenRelay(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// syncHiddenRelay démarre, arrête ou relance le relais caché selon les choix
// du bénévole et la liste des relais officiels.
func (n *Node) syncHiddenRelay(ctx context.Context) {
	h := n.hostingPrefs()
	official := n.officialMeshRelays()
	want := !n.cfg.NoHiddenRelay && h.relayOn() && len(official) > 0
	n.mu.Lock()
	cur := n.hidden
	if cur != nil {
		// consignes réseau vues par le relais caché, gardées entre deux relances
		// (sinon le plafond oscillerait à chaque redémarrage)
		if p := cur.b.Policy(); p != nil {
			n.relayPolicy = p
		}
	}
	kbs := n.relayPolicy.RelayKBs(h.RelayKBs) // les consignes ne peuvent que baisser le plafond
	n.mu.Unlock()
	entry := ""
	if h.entryOn() {
		entry = h.EntryListen + "|" + h.EntryURL
	}
	if cur != nil && (!want || !slices.Equal(cur.official, official) || cur.kbs != kbs || cur.exit != h.relayExitOn() ||
		cur.entry != entry || entry != "" && cur.srv == nil) {
		n.stopHiddenRelay()
		cur = nil
	}
	if !want || cur != nil {
		return
	}
	id, err := n.relayIdentity()
	if err != nil {
		n.setErr(err)
		return
	}
	b := broker.New(broker.Config{RootKey: n.cfg.RootKey, MintBits: n.cfg.MintBits, MaxPeers: 256, Version: n.cfg.AppVersion,
		GlobalBytesPerSec: float64(kbs) * 1024}, broker.NewMeshOrigin())
	// relais d'entrée : écoute ouverte AVANT l'annonce (l'officiel sonde aussitôt)
	var ln net.Listener
	publicURL := ""
	if entry != "" {
		if ln, err = net.Listen("tcp", h.EntryListen); err != nil {
			n.setErr(fmt.Errorf("relais d'entrée %s: %w", h.EntryListen, err))
		} else {
			publicURL = h.EntryURL
		}
	}
	if err := b.EnableMesh(broker.MeshConfig{Key: id.Priv, Volunteer: true, Mint: id.MintNonce, MintBits: id.MintBits,
		VolunteerExit: h.relayExitOn(), Official: official, Every: n.hiddenEvery(), MaxCircuits: 64,
		PublicURL: publicURL, AllowPrivateProbe: n.cfg.AllowPrivateRelay}); err != nil {
		if ln != nil {
			_ = ln.Close()
		}
		n.setErr(err)
		return
	}
	var srv *http.Server
	if ln != nil {
		mux := http.NewServeMux()
		mux.Handle("GET /v1/ws", b)
		mux.Handle("GET /v1/mesh", b.MeshHandler())
		srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 120 * time.Second,
			MaxHeaderBytes: 16 << 10, ErrorLog: log.New(io.Discard, "", 0)} // erreurs HTTP : adresses, jetées
		go func() { _ = srv.Serve(ln) }()
	}
	rctx, cancel := context.WithCancel(ctx)
	go b.RunMesh(rctx)
	n.mu.Lock()
	n.hidden = &hiddenRelay{b: b, cancel: cancel, official: official, kbs: kbs, exit: h.relayExitOn(), entry: entry, srv: srv}
	n.mu.Unlock()
}

func (n *Node) hiddenEvery() time.Duration {
	if n.cfg.HiddenRelayEvery > 0 {
		return n.cfg.HiddenRelayEvery
	}
	return 15 * time.Second
}

func (n *Node) stopHiddenRelay() {
	n.mu.Lock()
	cur := n.hidden
	n.hidden = nil
	n.mu.Unlock()
	if cur != nil {
		cur.cancel()
		if cur.srv != nil {
			_ = cur.srv.Close()
		}
	}
}

// hiddenRelayLinks : relais officiels auxquels le relais caché est relié.
func (n *Node) hiddenRelayLinks() int {
	n.mu.Lock()
	cur := n.hidden
	n.mu.Unlock()
	if cur == nil {
		return 0
	}
	return cur.b.MeshLinks()
}

func hexKey(k []byte) string { return hex.EncodeToString(k) }

// relayIdentity : clé du relais caché, créée une seule fois (preuve de travail),
// distincte de l'identité de l'utilisateur.
func (n *Node) relayIdentity() (*peerid.Identity, error) {
	n.relayIDMu.Lock()
	defer n.relayIDMu.Unlock()
	if n.relayID != nil {
		return n.relayID, nil
	}
	bits := n.cfg.RelayMintBits
	if bits == 0 {
		bits = broker.DefaultVolunteerBits
	}
	id, err := peerid.LoadOrCreate(filepath.Join(n.cfg.DataDir, "relay-identity.json"), bits)
	if err != nil {
		return nil, err
	}
	n.relayID = id
	return id, nil
}

// HiddenRelayKey : clé publique du relais caché (nil s'il n'a jamais tourné).
func (n *Node) HiddenRelayKey() ed25519.PublicKey { return n.hiddenRelayKey() }

// hiddenRelayKey : clé du relais caché s'il existe déjà (pour l'exclure de nos propres circuits).
func (n *Node) hiddenRelayKey() ed25519.PublicKey {
	n.relayIDMu.Lock()
	defer n.relayIDMu.Unlock()
	if n.relayID == nil {
		return nil
	}
	return n.relayID.Pub
}
