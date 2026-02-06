package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"decentralized-net/blockchain"
	"decentralized-net/p2p"
	"decentralized-net/storage"

	"github.com/libp2p/go-libp2p/core/network"
)

// APIServer holds dependencies for the API
type APIServer struct {
	Node  *p2p.Node
	Vault storage.VaultInterface
}

// JobRequest represents a compute job submission
type JobRequest struct {
	WasmBase64 string `json:"wasm_base64"` // For simplicity in MVP, or multipart? Let's use multipart for files usually, but JSON for small stuff.
	// Actually, standard is multipart for files. Let's support Multipart form data for robust file handling.
	InputData string `json:"input_data"`
}

// StartAPIServer starts the HTTP gateway
func StartAPIServer(node *p2p.Node, vault storage.VaultInterface, port int) {
	server := &APIServer{
		Node:  node,
		Vault: vault,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/job", server.handleJob)
	mux.HandleFunc("/api/v1/upload", server.handleUpload)
	mux.HandleFunc("/api/v1/transaction", server.handleTransaction)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[API] HTTP Gateway listening on http://localhost%s", addr)

	// Run server in background
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[API] Server failed: %v", err)
		}
	}()
}

// handleJob handles POST /api/v1/job
// Expects Multipart Form: "wasm" (file), "input" (text)
func (s *APIServer) handleJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse Multipart Form
	// Limit upload size to 10MB for safety
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("wasm")
	if err != nil {
		http.Error(w, "Missing 'wasm' file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	wasmBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read wasm file", http.StatusInternalServerError)
		return
	}

	inputData := r.FormValue("input")
	txID := r.FormValue("tx_id")

	log.Printf("[API] Received Job Request. WASM: %s (%d bytes), Input: %s, TxID: %s", header.Filename, len(wasmBytes), inputData, txID)

	// 2. Find a Worker (DHT Logic reused!)
	if s.Node.DHT == nil {
		http.Error(w, "DHT not validated/enabled on this node", http.StatusServiceUnavailable)
		return
	}

	// Simple discovery logic (reused from main.go)
	// In a real app, we might cache providers.
	ctx := r.Context()
	providers, err := s.Node.DHT.FindProviders(ctx, "compute-node")
	if err != nil || len(providers) == 0 {
		http.Error(w, "No compute nodes found in the network", http.StatusServiceUnavailable)
		return
	}

	// Pick first available
	targetPeer := providers[0].ID
	if s.Node.Host.Network().Connectedness(targetPeer) != network.Connected {
		s.Node.Host.Connect(ctx, providers[0])
	}

	// 3. Execute Job
	result, err := s.Node.SendComputeReq(ctx, targetPeer, wasmBytes, []byte(inputData), txID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Compute failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 4. Return Result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"worker": targetPeer.String(),
		"result": string(result),
	})
}

// handleUpload handles POST /api/v1/upload
func (s *APIServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit to 50MB
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Sharding Logic (Reused)
	// TODO: Refactor sharding logic into a shared service to avoid duplication with main.go
	// For MVP, simplistic duplication or we check if we can import from storage easily if logic is there.
	// The ShardManager is in storage, but the "Distribution" logic is in main.go currently.
	// We should probably move distribution logic to a helper in p2p or storage.
	// For now, I'll implement basic local storage + announce here to verify API.

	sm, err := storage.NewShardManager(10, 4)
	if err != nil {
		http.Error(w, "Sharding init failed", http.StatusInternalServerError)
		return
	}

	shards, err := sm.Encode(data)
	if err != nil {
		http.Error(w, "Sharding failed", http.StatusInternalServerError)
		return
	}

	// Distribute (Round Robin implementation inline for now)
	peers := s.Node.Host.Network().Peers()
	allNodes := append(peers, s.Node.Host.ID())

	storedCount := 0
	for i, shard := range shards {
		targetPeer := allNodes[i%len(allNodes)]
		key := []byte(fmt.Sprintf("%s-shard-%d", header.Filename, i))

		if targetPeer == s.Node.Host.ID() {
			if err := s.Vault.Store(key, shard); err == nil {
				if s.Node.DHT != nil {
					go s.Node.DHT.Announce(string(key))
				}
				storedCount++
			}
		} else {
			// Remote
			if err := s.Node.SendStoreReq(r.Context(), targetPeer, key, shard); err == nil {
				storedCount++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"filename":       header.Filename,
		"shards_created": len(shards),
		"shards_stored":  storedCount,
		"nodes_involved": len(allNodes),
	})
}

// handleTransaction handles POST /api/v1/transaction
func (s *APIServer) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tx blockchain.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Add to Blockchain
	if s.Node.Chain == nil {
		http.Error(w, "Blockchain not initialized", http.StatusServiceUnavailable)
		return
	}

	log.Printf("[API] Received Transaction: %s (Amount: %d)", tx.ID, tx.Amount)

	if err := s.Node.Chain.AddTransaction(&tx); err != nil {
		http.Error(w, fmt.Sprintf("Transaction Rejected: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"tx_id":  tx.ID,
	})
}
