package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	ComputeProtocol = protocol.ID("/decentralized-net/compute/1.0.0")
	ComputeTimeout  = 30 * time.Second // Allow 30s for job execution
)

// VMInterface defines what the P2P layer needs from the Compute Engine
type VMInterface interface {
	Run(wasmCode []byte, input []byte) ([]byte, error)
}

// HandleComputeStream accepts incoming compute jobs.
// Protocol:
// 1. Read WasmSize + WasmBytes
// 2. Read InputSize + InputBytes
// 3. Execute VM
// 4. Send OutputSize + OutputBytes
func (n *Node) HandleComputeStream(vm VMInterface) {
	n.Host.SetStreamHandler(ComputeProtocol, func(s network.Stream) {
		defer s.Close()
		s.SetDeadline(time.Now().Add(ComputeTimeout))
		reader := bufio.NewReader(s)
		writer := bufio.NewWriter(s)

		log.Printf("[Compute] Receiving job from %s", s.Conn().RemotePeer())

		// 0. Read TxID (Payment)
		var txIDLen uint32
		if err := binary.Read(reader, binary.BigEndian, &txIDLen); err != nil {
			log.Printf("[Compute] Error reading TxID length: %v", err)
			return
		}
		txIDBytes := make([]byte, txIDLen)
		if _, err := io.ReadFull(reader, txIDBytes); err != nil {
			log.Printf("[Compute] Error reading TxID: %v", err)
			return
		}
		txID := string(txIDBytes)

		// PAYMENT VERIFICATION
		if n.Chain != nil {
			// Bypass for testing
			if txID == "FREE_PASS" {
				log.Printf("[Compute] Payment Verification Bypassed (FREE_PASS used)")
			} else {
				tx, err := n.Chain.FindTransaction(txID)
				if err != nil {
					log.Printf("[Compute] REJECTED: Payment Tx %s not found. Error: %v", txID, err)
					// We should send error back, but for now just return/close
					return
				}

				// Check Amount
				if tx.Amount < 5 {
					log.Printf("[Compute] REJECTED: Insufficient payment. Got %d, need 5.", tx.Amount)
					return
				}
				log.Printf("[Compute] Payment Verified! Tx: %s (%d coins)", txID, tx.Amount)
			}
		}

		// 1. Read Wasm
		var wasmLen uint32
		if err := binary.Read(reader, binary.BigEndian, &wasmLen); err != nil {
			log.Printf("[Compute] Error reading Wasm length: %v", err)
			return
		}
		wasmCode := make([]byte, wasmLen)
		if _, err := io.ReadFull(reader, wasmCode); err != nil {
			log.Printf("[Compute] Error reading Wasm code: %v", err)
			return
		}

		// 2. Read Input
		var inputLen uint32
		if err := binary.Read(reader, binary.BigEndian, &inputLen); err != nil {
			log.Printf("[Compute] Error reading Input length: %v", err)
			return
		}
		inputData := make([]byte, inputLen)
		if _, err := io.ReadFull(reader, inputData); err != nil {
			log.Printf("[Compute] Error reading Input data: %v", err)
			return
		}

		// 3. Execute
		log.Printf("[Compute] Executing WASM (%d bytes)...", wasmLen)
		output, err := vm.Run(wasmCode, inputData)

		// 4. Send Response
		if err != nil {
			// In a real protocol, we'd send an Error flag.
			// For MVP, we send empty output and log the error.
			log.Printf("[Compute] Execution failed: %v", err)
			output = []byte(fmt.Sprintf("ERROR: %v", err))
		}

		// Write Output Length
		if err := binary.Write(writer, binary.BigEndian, uint32(len(output))); err != nil {
			log.Printf("[Compute] Failed to write result length: %v", err)
			return
		}
		// Write Output
		if _, err := writer.Write(output); err != nil {
			log.Printf("[Compute] Failed to write result: %v", err)
			return
		}
		writer.Flush()
		log.Printf("[Compute] Job complete. Sent %d bytes result.", len(output))
	})
}

// SendComputeReq sends a job to a peer and waits for the result.
func (n *Node) SendComputeReq(ctx context.Context, p peer.ID, wasm []byte, input []byte, txID string) ([]byte, error) {
	s, err := n.Host.NewStream(ctx, p, ComputeProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(ComputeTimeout))

	writer := bufio.NewWriter(s)
	reader := bufio.NewReader(s)

	// 0. Write TxID
	if err := binary.Write(writer, binary.BigEndian, uint32(len(txID))); err != nil {
		return nil, err
	}
	if _, err := writer.Write([]byte(txID)); err != nil {
		return nil, err
	}

	// 1. Write Wasm
	if err := binary.Write(writer, binary.BigEndian, uint32(len(wasm))); err != nil {
		return nil, err
	}
	if _, err := writer.Write(wasm); err != nil {
		return nil, err
	}

	// 2. Write Input
	if err := binary.Write(writer, binary.BigEndian, uint32(len(input))); err != nil {
		return nil, err
	}
	if _, err := writer.Write(input); err != nil {
		return nil, err
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush request: %w", err)
	}

	// 3. Read Result
	var outLen uint32
	if err := binary.Read(reader, binary.BigEndian, &outLen); err != nil {
		return nil, fmt.Errorf("failed to read result length: %w", err)
	}
	output := make([]byte, outLen)
	if _, err := io.ReadFull(reader, output); err != nil {
		return nil, fmt.Errorf("failed to read result data: %w", err)
	}

	return output, nil
}
