package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/multiformats/go-multiaddr"
	"t2c/pkg/onion"
	"t2c/pkg/swarm"
)

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

func NewNode(ctx context.Context, privKey crypto.PrivKey, listenPort int, isRelay bool) (*Node, error) {
	var idht *dht.IpfsDHT

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)

	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableRelay(), // Always enable client relay capability
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			idht, err = dht.New(ctx, h, dht.Mode(dht.ModeServer))
			return idht, err
		}),
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

	n.SwarmManager = swarm.NewSwarmManager(h, idht, "data/swarm_zips")

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

func (n *Node) Bootstrap(ctx context.Context, bootstrapPeers []string) error {
	for _, addr := range bootstrapPeers {
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
		}
	}

	if err := n.DHT.Bootstrap(ctx); err != nil {
		return err
	}
	
	time.Sleep(2 * time.Second)
	return nil
}
