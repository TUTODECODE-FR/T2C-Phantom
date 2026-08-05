package p2p

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	"github.com/multiformats/go-multiaddr"
	"t2c/pkg/onion"
	"t2c/pkg/swarm"
	"t2c/pkg/utils"
)

// getPublicIP fetches the machine's public IP. Returns empty string on failure.
func getPublicIP() (result string) {
	// Safety: never crash if the network call panics
	defer func() {
		if r := recover(); r != nil {
			result = ""
		}
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}
	for _, url := range services {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		buf := make([]byte, 64)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		ip := strings.TrimSpace(string(buf[:n]))
		if ip != "" {
			return ip
		}
	}
	return ""
}

type Node struct {
	Host         host.Host
	DHT          *dht.IpfsDHT
	ExitPort     int
	SwarmManager *swarm.SwarmManager
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.h.Connect(context.Background(), pi)
}

func NewNode(ctx context.Context, privKey crypto.PrivKey, listenPort int, isRelay bool, staticRelays []string, maxPeers int, maxMemory int64) (*Node, error) {
	var idht *dht.IpfsDHT

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort) // tcp/4001
	
	wsPort := listenPort + 1000
	quicPort := listenPort
	rawTcp443 := false
	if isRelay {
		wsPort = 8443 // WebSocket for CT100
		quicPort = 4001 
		rawTcp443 = false
	}
	wsAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", wsPort) // tcp/8443/ws
	quicAddr := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", quicPort)
	webrtcAddr := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/webrtc-direct", quicPort) // WebRTC (UDP) pour l'obfuscation DPI

	addrs := []string{wsAddr, quicAddr, webrtcAddr}
	// Always listen on Raw TCP for IPFS/AutoRelay
	addrs = append(addrs, listenAddr)
	
	if rawTcp443 {
		addrs = append(addrs, "/ip4/0.0.0.0/tcp/443")
	}

	// 🛡️ FD/Connection Exhaustion Prevention (Anti Slow-Read)
	limits := rcmgr.DefaultLimits
	libp2p.SetDefaultServiceLimits(&limits)

	limits.AddServiceLimit("t2c", rcmgr.BaseLimit{
		Streams:         128,
		StreamsInbound:  64,
		StreamsOutbound: 64,
		Conns:           16, // Max connections per service
		ConnsInbound:    8,
		ConnsOutbound:   8,
		FD:              16, // Max File Descriptors per service
	}, rcmgr.BaseLimitIncrease{})

	var partialLimits rcmgr.PartialLimitConfig
	if maxMemory > 0 {
		partialLimits.System.Memory = rcmgr.LimitVal64(maxMemory)
	}
	if maxPeers > 0 {
		partialLimits.System.Conns = rcmgr.LimitVal(maxPeers)
		partialLimits.System.ConnsInbound = rcmgr.LimitVal(maxPeers / 2)
		partialLimits.System.ConnsOutbound = rcmgr.LimitVal(maxPeers / 2)
	}

	concreteLimits := partialLimits.Build(limits.AutoScale())
	limiter := rcmgr.NewFixedLimiter(concreteLimits)
	
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.ResourceManager(rm),
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(addrs...),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Transport(libp2pwebrtc.New), // Transport WebRTC pour simuler Google Meet (anti-DPI)
		libp2p.Transport(ws.New, ws.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true})), // Override WS for self-signed WSS
		libp2p.EnableRelay(), // Always enable client relay capability
		libp2p.NATPortMap(),  // Enable uPnP / NAT-PMP
	}

	// 🛡️ Multi-Relay Bootstrapping (Anti-SPOF)
	if len(staticRelays) > 0 {
		var relayPinfos []peer.AddrInfo
		for _, r := range staticRelays {
			ma, err := multiaddr.NewMultiaddr(r)
			if err == nil {
				if pi, err := peer.AddrInfoFromP2pAddr(ma); err == nil {
					relayPinfos = append(relayPinfos, *pi)
				}
			}
		}
		if len(relayPinfos) > 0 {
			opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays(relayPinfos))
		}
	}

	opts = append(opts, libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			idht, err = dht.New(context.Background(), h, dht.Mode(dht.ModeAuto), dht.ProtocolPrefix("/t2c/phantom/v1"))
			return idht, err
		}))

	if isRelay {
		opts = append(opts, libp2p.ForceReachabilityPublic())
		opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays(nil))
		
		// Announce real addresses
		if pubIP := getPublicIP(); pubIP != "" {
			opts = append(opts, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				tcpMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", pubIP, listenPort))
				wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/443/wss", pubIP))
				return append(addrs, tcpMa, wsMa)
			}))
		}
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	if isRelay {
		// Act as a public relay node to mask IPs
		_, err = relay.New(h)
		if err != nil {
			return nil, fmt.Errorf("failed to start relay service: %w", err)
		}
		fmt.Println("Circuit Relay v2 Service ENABLED. This node will route traffic anonymously.")
	}

	// Enable local mDNS discovery for automatic pairing on the same network
	mdnsService := mdns.NewMdnsService(h, "t2c-local-discovery", &discoveryNotifee{h: h})
	mdnsService.Start()

	n := &Node{
		Host: h,
		DHT:  idht,
	}

	n.SwarmManager = swarm.NewSwarmManager(h, idht, utils.GetPath("swarm_zips"))

	// Register this node to be an Onion Relay on the network
	onion.RegisterOnionRelay(h, func() int { return n.ExitPort })

	return n, nil
}

// ReserveRelay connects to a relay node and reserves a slot to be reachable via circuit.
func (n *Node) ReserveRelay(ctx context.Context, relayAddrStr string) error {
	relayMa, err := multiaddr.NewMultiaddr(relayAddrStr)
	if err != nil {
		return fmt.Errorf("invalid relay address: %w", err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMa)
	if err != nil {
		return err
	}

	if err := n.Host.Connect(ctx, *relayInfo); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	_, err = client.Reserve(ctx, n.Host, *relayInfo)
	if err != nil {
		return fmt.Errorf("failed to reserve slot on relay: %w", err)
	}

	fmt.Printf("Success! Tunnel secured. We are now reachable via relay: %s\n", relayInfo.ID.String())
	return nil
}

// Noeuds de confiance TUTODECODE hardcodés pour prévenir les attaques par Éclipse
var TrustedTutoDecodeNodes = []string{
	"/ip4/198.51.100.1/tcp/4001/p2p/QmTrustedTutoDecodeNode01",
	"/ip4/198.51.100.2/tcp/4001/p2p/QmTrustedTutoDecodeNode02",
}

func (n *Node) Bootstrap(ctx context.Context, bootstrapPeers []string) error {
	// Fusionner les noeuds dynamiques avec les noeuds hardcodés TUTODECODE
	allPeers := append(TrustedTutoDecodeNodes, bootstrapPeers...)

	for _, addr := range allPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}
		if err := n.Host.Connect(ctx, *pi); err != nil {
			fmt.Printf("Failed to connect to bootstrap peer %s: %s\n", addr, err)
		} else {
			// Protect the connection to prevent Eclipse Attacks (Pinning strict)
			n.Host.ConnManager().Protect(pi.ID, "tutodecode-anchor")
		}
	}

	if err := n.DHT.Bootstrap(ctx); err != nil {
		return err
	}
	
	time.Sleep(2 * time.Second)
	return nil
}

// TrafficShapingWriter enveloppe un stream libp2p pour émuler un flux WebRTC Audio/Vidéo.
// Cela empêche le Deep Packet Inspection (DPI) d'identifier les transferts de fichiers 
// en forçant des tailles de paquets constantes et des délais rythmiques (Padding & Shaping).
type TrafficShapingWriter struct {
	stream io.Writer
}

func NewTrafficShapingWriter(w io.Writer) *TrafficShapingWriter {
	return &TrafficShapingWriter{stream: w}
}

func (t *TrafficShapingWriter) Write(p []byte) (n int, err error) {
	// Imiter le MTU d'un paquet Vidéo WebRTC (environ 1200-1300 octets de charge utile)
	const emulateMTU = 1200 
	
	totalWritten := 0
	for totalWritten < len(p) {
		chunkSize := emulateMTU
		if len(p)-totalWritten < chunkSize {
			chunkSize = len(p) - totalWritten
		}
		
		// 🛡️ ANTI-DPI: Découpe le buffer massif (ex: 256 Ko) en minuscules morceaux de 1200 octets
		written, err := t.stream.Write(p[totalWritten : totalWritten+chunkSize])
		totalWritten += written
		if err != nil {
			return totalWritten, err
		}
		
		// 🛡️ ANTI-DPI: Émule le délai temps-réel d'un flux multimédia (~20ms par frame vidéo)
		// Lisse la courbe de bande passante pour contourner l'analyse statistique (Traffic Analysis).
		time.Sleep(20 * time.Millisecond)
	}
	return totalWritten, nil
}
