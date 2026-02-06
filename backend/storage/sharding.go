package storage

import (
	"bytes"
	"fmt"


	"github.com/klauspost/reedsolomon"
)

// ShardManager handles splitting and reconstructing data.
type ShardManager struct {
	enc          reedsolomon.Encoder
	dataShards   int
	parityShards int
}

// NewShardManager creates a new manager with specific N/K parameters.
func NewShardManager(dataShards, parityShards int) (*ShardManager, error) {
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}
	return &ShardManager{
		enc:          enc,
		dataShards:   dataShards,
		parityShards: parityShards,
	}, nil
}

// Encode splits data into data+parity shards.
func (s *ShardManager) Encode(data []byte) ([][]byte, error) {
	// Split the data into shards
	shards, err := s.enc.Split(data)
	if err != nil {
		return nil, err
	}
	// Encode parity
	err = s.enc.Encode(shards)
	if err != nil {
		return nil, err
	}
	return shards, nil
}

// Reconstruct attempts to fix missing shards and return the full data.
// shards: slice of length (dataShards + parityShards). Missing shards must be nil.
// originalSize: the exact size of the original file (to strip padding).
func (s *ShardManager) Reconstruct(shards [][]byte, originalSize int) ([]byte, error) {
	// Check if reconstruction is possible/needed
	ok, err := s.enc.Verify(shards)
	if !ok || err != nil {
		// Attempt reconstruction
		err = s.enc.Reconstruct(shards)
		if err != nil {
			return nil, fmt.Errorf("reconstruction failed (not enough shards?): %w", err)
		}
		// Verify again to be sure
		if ok, err := s.enc.Verify(shards); !ok || err != nil {
			return nil, fmt.Errorf("shards are still invalid after reconstruction: %w", err)
		}
	}

	// Join the data shards
	var buf bytes.Buffer
	err = s.enc.Join(&buf, shards, originalSize)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Example usage helper
func (s *ShardManager) PrintStats(dataLen int) {
	fmt.Printf("Split %d bytes into %d data + %d parity shards.\n", dataLen, s.dataShards, s.parityShards)
}
