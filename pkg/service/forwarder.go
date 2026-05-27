package service

import (
	"context"
	"fmt"
	"io"
	"net"

	"t2c/pkg/p2p"

	"github.com/libp2p/go-libp2p/core/network"
)

const ProtocolID = "/t2c/http/1.0.0"

// ExposeService binds a local TCP port to the P2P network.
func ExposeService(ctx context.Context, node *p2p.Node, localPort int) error {
	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

	// Register a stream handler for the T2C protocol
	node.Host.SetStreamHandler(ProtocolID, func(stream network.Stream) {
		defer stream.Close()
		
		// Connect to the local service
		conn, err := net.Dial("tcp", localAddr)
		if err != nil {
			fmt.Printf("Failed to connect to local service on %s: %v\n", localAddr, err)
			return
		}
		defer conn.Close()

		// Bidirectional copy
		go io.Copy(conn, stream)
		io.Copy(stream, conn)
	})

	fmt.Printf("Service on local port %d exposed via P2P on Peer ID: %s\n", localPort, node.Host.ID())
	return nil
}
