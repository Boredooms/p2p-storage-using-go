package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"decentralized-net/storage"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	StoreProtocol    = protocol.ID("/decentralized-net/store/1.0.0")
	RetrieveProtocol = protocol.ID("/decentralized-net/retrieve/1.0.0")
	StreamTimeout    = 30 * time.Second
)

// HandleStoreStream accepts incoming store requests.
// Protocol Format:
// [KeyLength (4 bytes)] [Key Bytes]
// [DataLength (4 bytes)] [Data Bytes]
// Response:
// [Status (1 byte)] (0=Success, 1=Error)
func (n *Node) HandleStoreStream(v storage.VaultInterface) {
	n.Host.SetStreamHandler(StoreProtocol, func(s network.Stream) {
		defer s.Close()
		s.SetDeadline(time.Now().Add(StreamTimeout))

		reader := bufio.NewReader(s)

		// 1. Read Key Length
		var keyLen uint32
		if err := binary.Read(reader, binary.BigEndian, &keyLen); err != nil {
			log.Printf("[P2P] Protocol Error: Failed to read key length: %v", err)
			return
		}

		// 2. Read Key
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, key); err != nil {
			log.Printf("[P2P] Protocol Error: Failed to read key: %v", err)
			return
		}

		// 3. Read Data Length
		var dataLen uint32
		if err := binary.Read(reader, binary.BigEndian, &dataLen); err != nil {
			log.Printf("[P2P] Protocol Error: Failed to read data length: %v", err)
			return
		}

		// 4. Read Data
		data := make([]byte, dataLen)
		if _, err := io.ReadFull(reader, data); err != nil {
			log.Printf("[P2P] Protocol Error: Failed to read data: %v", err)
			return
		}

		log.Printf("[P2P] Received Shard: %s (%d bytes) from %s", string(key), len(data), s.Conn().RemotePeer())

		// 5. Store in Vault
		var status byte = 0 // Success
		if v != nil {
			if err := v.Store(key, data); err != nil {
				log.Printf("[P2P] Storage Failed: %v", err)
				status = 1 // Error
			} else {
				log.Printf("[Storage] Saved shard: %s", string(key))
				// -----------------------------------------------------
				// DHT ANNOUNCEMENT: "I have this shard!"
				// -----------------------------------------------------
				if n.DHT != nil {
					go func() {
						if err := n.DHT.Announce(string(key)); err != nil {
							log.Printf("[DHT] Failed to announce %s: %v", string(key), err)
						}
					}()
				}
			}
		}

		// 6. Send Acknowledgement
		if _, err := s.Write([]byte{status}); err != nil {
			log.Printf("[P2P] Failed to send ACK: %v", err)
		}
	})
}

// HandleRetrieveStream handles incoming requests for data.
// Protocol: [KeyLen] [Key] -> Response: [Status] [DataLen] [Data]
func (n *Node) HandleRetrieveStream(v storage.VaultInterface) {
	n.Host.SetStreamHandler(RetrieveProtocol, func(s network.Stream) {
		defer s.Close()
		s.SetDeadline(time.Now().Add(StreamTimeout))

		reader := bufio.NewReader(s)

		// 1. Read Key Len
		var keyLen uint32
		if err := binary.Read(reader, binary.BigEndian, &keyLen); err != nil {
			return
		}

		// 2. Read Key
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, key); err != nil {
			return
		}

		log.Printf("[P2P] Peer %s requesting shard: %s", s.Conn().RemotePeer(), string(key))

		// 3. Check Vault
		var data []byte
		var err error
		var status byte = 0

		if v != nil {
			data, err = v.Get(key)
			if err != nil {
				log.Printf("[P2P] Shard %s not found: %v", string(key), err)
				status = 1 // Not Found
			}
		} else {
			status = 1 // No Vault
		}

		// 4. Send Response Header (Status + DataLen)
		if err := binary.Write(s, binary.BigEndian, status); err != nil {
			return
		}

		// If success, send data
		if status == 0 {
			if err := binary.Write(s, binary.BigEndian, uint32(len(data))); err != nil {
				return
			}
			if _, err := s.Write(data); err != nil {
				return
			}
			log.Printf("[P2P] Sent shard %s (%d bytes) to %s", string(key), len(data), s.Conn().RemotePeer())
		}
	})
}

// SendStoreReq connects to a peer and sends data with the defined protocol.
func (n *Node) SendStoreReq(ctx context.Context, p peer.ID, key []byte, data []byte) error {
	s, err := n.Host.NewStream(ctx, p, StoreProtocol)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(StreamTimeout))

	writer := bufio.NewWriter(s)

	// 1. Write Key Length
	if err := binary.Write(writer, binary.BigEndian, uint32(len(key))); err != nil {
		return err
	}
	// 2. Write Key
	if _, err := writer.Write(key); err != nil {
		return err
	}

	// 3. Write Data Length
	if err := binary.Write(writer, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	// 4. Write Data
	if _, err := writer.Write(data); err != nil {
		return err
	}

	// Flush to ensure data is sent
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush stream: %w", err)
	}

	// 5. Read Acknowledgement
	ack := make([]byte, 1)
	if _, err := io.ReadFull(s, ack); err != nil {
		return fmt.Errorf("failed to read ACK: %w", err)
	}

	if ack[0] != 0 {
		return fmt.Errorf("peer returned error status")
	}

	return nil
}

// SendRetrieveReq requests data from a peer using the RetrieveProtocol.
func (n *Node) SendRetrieveReq(ctx context.Context, p peer.ID, key string) ([]byte, error) {
	s, err := n.Host.NewStream(ctx, p, RetrieveProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(StreamTimeout))

	// 1. Send Key
	keyBytes := []byte(key)
	if err := binary.Write(s, binary.BigEndian, uint32(len(keyBytes))); err != nil {
		return nil, err
	}
	if _, err := s.Write(keyBytes); err != nil {
		return nil, err
	}

	// 2. Read Status
	var status byte
	if err := binary.Read(s, binary.BigEndian, &status); err != nil {
		return nil, err
	}

	if status != 0 {
		return nil, fmt.Errorf("peer returned error status (file not found?)")
	}

	// 3. Read Data Len
	var dataLen uint32
	if err := binary.Read(s, binary.BigEndian, &dataLen); err != nil {
		return nil, err
	}

	// 4. Read Data
	data := make([]byte, dataLen)
	if _, err := io.ReadFull(s, data); err != nil {
		return nil, err
	}

	return data, nil
}
