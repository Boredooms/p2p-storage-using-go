package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"decentralized-net/blockchain"
)

// Node represents a P2P node in the network.
type Node struct {
	Host       host.Host
	DHT        *DHTWrapper
	Ctx        context.Context
	Chain      *blockchain.Blockchain
	PubSub     *pubsub.PubSub
	BlockTopic *pubsub.Topic
}

// NewNode creates a new libp2p Host with a generated identity.
// listenPort: 0 for random port, or specific port (e.g., 3000).
func NewNode(ctx context.Context, listenPort int) (*Node, error) {
	// 1. Generate an Ed25519 key pair for the node's identity.
	// In a real app, you'd save this private key to disk (Vault) to persist identity.
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// 2. Configure the Host options.
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", listenPort), // Enable QUIC for speed
		),
		libp2p.DefaultTransports,    // TCP, QUIC, WS
		libp2p.DefaultMuxers,        // Yamux, Mplex
		libp2p.DefaultSecurity,      // Noise, TLS
		libp2p.NATPortMap(),         // Try to punch through NAT
		libp2p.EnableRelayService(), // Enable acting as a relay
		libp2p.EnableHolePunching(), // Enable hole punching
	}

	// 3. Create the Host.
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	log.Printf("Helper: Node started. ID: %s", h.ID())
	log.Printf("Helper: Listening on:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID())
	}

	// 4. Create PubSub (GossipSub)
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossipsub: %w", err)
	}

	// 5. Create Node struct
	n := &Node{
		Host:   h,
		Ctx:    ctx,
		PubSub: ps,
	}

	// 5. Initialize DHT (Kademlia)
	// Note: We don't pass bootstrap peers here yet, we will do it via a method or main loop.
	// For now, initializing it enables the capability.
	// We'll call SetupDHT logic from main or here?
	// To keep NewNode simple, let's leave it nil and set it up explicitly in main or add a method.
	// Actually, let's add a method "EnableDHT" like in the plan.

	// 6. Set generic stream handler (example).
	h.SetStreamHandler("/decentralized-net/1.0.0", n.handleStream)

	return n, nil
}

// EnableDHT starts the Distributed Hash Table.
func (n *Node) EnableDHT(bootstrapPeers []string) error {
	dht, err := SetupDHT(n.Ctx, n.Host, bootstrapPeers)
	if err != nil {
		return err
	}
	n.DHT = dht
	return nil
}

// handleStream handles incoming streams.
func (n *Node) handleStream(s network.Stream) {
	log.Printf("Got a new stream from %s!", s.Conn().RemotePeer())

	// Create a buffer stream for non blocking read and write.
	// For now, just echo.
	_, err := io.Copy(s, s)
	if err != nil {
		log.Println("Error echoing stream:", err)
		s.Reset()
	} else {
		s.Close()
	}
}

// Connect establishes a connection to a peer.
func (n *Node) Connect(peerAddr string) error {
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	if err := n.Host.Connect(n.Ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	log.Printf("[P2P] Successfully connected to peer: %s", info.ID)
	return nil
}

// BuildMultiAddr helper to create a connectable address string.
func BuildMultiAddr(port int, id peer.ID) string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", port, id)
}
