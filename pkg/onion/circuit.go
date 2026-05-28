package onion

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/t2c/onion/1.0"

type OnionMessage struct {
	Next string `json:"next"`
}

// RegisterOnionRelay registers the handler so this node acts as an onion relay.
func RegisterOnionRelay(h host.Host, getExitPort func() int) {
	h.SetStreamHandler(ProtocolID, func(stream network.Stream) {
		reader := bufio.NewReader(stream)
		line, err := reader.ReadString('\n')
		if err != nil {
			stream.Reset()
			return
		}

		var msg OnionMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			stream.Reset()
			return
		}

		if msg.Next == "FINAL" {
			exitPort := getExitPort()
			if exitPort == 0 {
				stream.Reset() // This node is not hosting anything
				return
			}
			
			// Connect to the local hidden service (HTTPS)
			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", exitPort))
			if err != nil {
				stream.Reset()
				return
			}
			
			go func() {
				defer conn.Close()
				defer stream.Close()
				io.Copy(conn, reader)
			}()
			go func() {
				defer conn.Close()
				defer stream.Close()
				io.Copy(stream, conn)
			}()
			return
		}

		// Relay mode: Connect to the next hop
		nextPeerID, err := peer.Decode(msg.Next)
		if err != nil {
			stream.Reset()
			return
		}

		// We assume the network routing (DHT) will handle the connection implicitly if not already connected
		nextStream, err := h.NewStream(context.Background(), nextPeerID, ProtocolID)
		if err != nil {
			stream.Reset()
			return
		}

		go func() {
			defer nextStream.Close()
			defer stream.Close()
			io.Copy(nextStream, reader)
		}()
		go func() {
			defer nextStream.Close()
			defer stream.Close()
			io.Copy(stream, nextStream)
		}()
	})
}

// BuildOnionCircuit builds a multi-hop tunnel towards the target peer through intermediate relay peers.
func BuildOnionCircuit(ctx context.Context, h host.Host, target peer.ID, relays []peer.ID) (network.Stream, error) {
	if len(relays) == 0 {
		// No relays: Direct connection over onion protocol
		stream, err := h.NewStream(ctx, target, ProtocolID)
		if err != nil {
			return nil, err
		}
		
		msg := OnionMessage{Next: "FINAL"}
		msgBytes, _ := json.Marshal(msg)
		stream.Write(append(msgBytes, '\n'))
		return stream, nil
	}

	// 1. Connect to the first relay
	firstRelay := relays[0]
	stream, err := h.NewStream(ctx, firstRelay, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to first onion relay %s: %w", firstRelay, err)
	}

	// 2. Chain instructions for subsequent hops
	for i := 1; i < len(relays); i++ {
		nextRelay := relays[i]
		msg := OnionMessage{Next: nextRelay.String()}
		msgBytes, _ := json.Marshal(msg)
		_, err := stream.Write(append(msgBytes, '\n'))
		if err != nil {
			stream.Reset()
			return nil, fmt.Errorf("failed to chain relay %s: %w", nextRelay, err)
		}
	}

	// 3. Final instruction to connect to the target (host) node
	msg := OnionMessage{Next: target.String()}
	msgBytes, _ := json.Marshal(msg)
	_, err = stream.Write(append(msgBytes, '\n'))
	if err != nil {
		stream.Reset()
		return nil, fmt.Errorf("failed to link target through onion path: %w", err)
	}

	return stream, nil
}
