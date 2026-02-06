package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// DHTWrapper wraps the Kademlia DHT
type DHTWrapper struct {
	DHT *dht.IpfsDHT
	Ctx context.Context
}

// SetupDHT initializes the Kademlia DHT.
func SetupDHT(ctx context.Context, h host.Host, bootstrapPeers []string) (*DHTWrapper, error) {
	// 1. NewDHT creates a Kademlia DHT.
	// ModeServer allows this node to answer queries.
	kademliaDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// 2. Bootstrap the DHT
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// 3. Connect to Bootstrap Nodes
	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		if peerAddr == "" {
			continue
		}
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Printf("[DHT] Invalid bootstrap addr %s: %v", peerAddr, err)
			continue
		}
		peerinfo, _ := peer.AddrInfoFromP2pAddr(addr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Try to connect, but don't block everything if one fails
			if err := h.Connect(ctx, *peerinfo); err != nil {
				log.Printf("[DHT] Failed to connect to bootstrap %s: %v", peerAddr, err)
			} else {
				log.Printf("[DHT] Connected to bootstrap node: %s", peerinfo.ID)
			}
		}()
	}
	wg.Wait()

	return &DHTWrapper{
		DHT: kademliaDHT,
		Ctx: ctx,
	}, nil
}

// getCID converts a string key into a CID (Content Identifier)
func getCID(key string) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // Default length
	}
	c, err := pref.Sum([]byte(key))
	if err != nil {
		return cid.Undef, err
	}
	return c, nil
}

// Announce tells the network "I have this data/service".
// Use this for Shards (key=shardName) AND for Service Discovery (key="compute-node").
func (d *DHTWrapper) Announce(key string) error {
	c, err := getCID(key)
	if err != nil {
		return fmt.Errorf("invalid cid: %w", err)
	}

	// Provide adds our PeerID to the DHT as a provider for this key.
	// This might take a moment to propagate.
	ctx, cancel := context.WithTimeout(d.Ctx, 10*time.Second)
	defer cancel()

	if err := d.DHT.Provide(ctx, c, true); err != nil {
		return fmt.Errorf("failed to provide: %w", err)
	}

	log.Printf("[DHT] Announced: %s (CID: %s)", key, c.String())
	return nil
}

// FindProviders asks the network "Who has this data/service?".
// Returns a list of Peer IDs.
func (d *DHTWrapper) FindProviders(key string) ([]peer.AddrInfo, error) {
	c, err := getCID(key)
	if err != nil {
		return nil, fmt.Errorf("invalid cid: %w", err)
	}

	// FindProvidersAsync returns a channel of providers.
	ctx, cancel := context.WithTimeout(d.Ctx, 10*time.Second)
	defer cancel()

	providers := d.DHT.FindProvidersAsync(ctx, c, 10) // Find up to 10 providers

	var nodes []peer.AddrInfo
	for p := range providers {
		if p.ID == "" {
			continue
		}
		nodes = append(nodes, p)
	}

	return nodes, nil
}
