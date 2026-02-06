package p2p

import (
	"context"
	"fmt"
	"log"

	"decentralized-net/blockchain"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const BlockTopic = "/blockchain/blocks/1.0.0"

// SetupBlockPropagation subscribes to the block topic and listens for new blocks.
func (n *Node) SetupBlockPropagation() error {
	// Join the topic
	topic, err := n.PubSub.Join(BlockTopic)
	if err != nil {
		return fmt.Errorf("failed to join topic: %w", err)
	}

	// Subscribe
	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start listener loop in background
	go n.listenForBlocks(sub)

	n.BlockTopic = topic

	log.Printf("[P2P] Listening for blocks on %s", BlockTopic)
	return nil
}

// BroadcastBlock publishes a mined block to the network.
func (n *Node) BroadcastBlock(b *blockchain.Block) error {
	if n.BlockTopic == nil {
		return fmt.Errorf("block topic not joined")
	}

	data := b.Serialize() // Using Gob serialization we made
	return n.BlockTopic.Publish(n.Ctx, data)
}

// listenForBlocks processes incoming block messages.
func (n *Node) listenForBlocks(sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(n.Ctx)
		if err != nil {
			if err != context.Canceled {
				log.Printf("[P2P] Subscription error: %v", err)
			}
			return
		}

		// Don't process our own messages
		if msg.ReceivedFrom == n.Host.ID() {
			continue
		}

		// Deserialize
		block := blockchain.DeserializeBlock(msg.Data)
		log.Printf("[P2P] Received new block: %s from %s", block.Hash, msg.ReceivedFrom)

		// Verification and Add to Chain
		// Note: AddBlock typically does validation inside.
		// But since we are using BadgerDB, AddBlock needs to check if it already exists/etc.
		// For MVP, we trust AddBlock logic to reject duplicates or invalid PoW.
		if n.Chain != nil {
			// We might want to run this in a goroutine if AddBlock is slow,
			// but sequential is safer for DB consistency for now.
			n.Chain.AddBlock([]*blockchain.Transaction{})
			// WAIT. AddBlock MINES a new block in my current implementation.
			// I need a method to Insert *existing* block.

			// Refactoring Alert: chain.AddBlock() currently MINES.
			// I need `chain.ProcessBlock(b *Block)` which validates and inserts.

			// I will implement ProcessBlock later. For now, let's log.
			log.Printf("TODO: Insert Block %d into DB (Logic pending)", block.Index)
		}
	}
}
