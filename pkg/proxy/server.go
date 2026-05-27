package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"t2c/pkg/crypto"
	"t2c/pkg/p2p"
	"t2c/pkg/service"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// streamConn implements net.Conn over a libp2p stream.
type streamConn struct {
	network.Stream
}

func (c *streamConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

func (c *streamConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

func resolveCarnetAlias(carnetPath, host string) string {
	if carnetPath == "" {
		return host
	}
	
	data, err := os.ReadFile(carnetPath)
	if err != nil {
		return host // File doesn't exist or is unreadable, ignore
	}

	var aliases map[string]string
	if err := json.Unmarshal(data, &aliases); err != nil {
		fmt.Printf("Error parsing carnet: %v\n", err)
		return host
	}

	for k, v := range aliases {
		if strings.EqualFold(k, host) {
			return v
		}
	}
	return host
}

func StartProxyServer(node *p2p.Node, listenAddr, carnetPath string) error {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, networkLayer, addr string) (net.Conn, error) {
			host := strings.Split(addr, ":")[0]
			
			// Naming Service: Check local carnet.json for aliases
			resolvedHost := resolveCarnetAlias(carnetPath, host)

			if !strings.HasSuffix(resolvedHost, ".t2c") {
				return nil, fmt.Errorf("only .t2c domains are supported")
			}

			peerID, err := crypto.ParseT2CAddress(resolvedHost)
			if err != nil {
				return nil, err
			}

			// Force local connection for POC if mDNS is slow
			if localAddrBytes, err := os.ReadFile("data/local_addr.txt"); err == nil {
				if ma, err := multiaddr.NewMultiaddr(string(localAddrBytes)); err == nil {
					if pi, err := peer.AddrInfoFromP2pAddr(ma); err == nil {
						node.Host.Connect(ctx, *pi)
					}
				}
			}

			// Try to find peer in DHT to discover addresses, but ignore errors
			// because the peer might already be connected locally via mDNS.
			pi, err := node.DHT.FindPeer(ctx, peerID)
			if err == nil {
				node.Host.Connect(ctx, pi)
			} else {
				fmt.Printf("DHT FindPeer failed (might use mDNS or local_addr): %v\n", err)
			}

			// Create P2P stream
			stream, err := node.Host.NewStream(ctx, peerID, service.ProtocolID)
			if err != nil {
				return nil, err
			}

			return &streamConn{Stream: stream}, nil
		},
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
		},
		Transport: transport,
	}

	fmt.Printf("Starting HTTP Proxy on %s\n", listenAddr)
	return http.ListenAndServe(listenAddr, proxy)
}
