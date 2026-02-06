package api

import (
	"decentralized-net/compute"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

// handleJobSubmit handles POST /api/jobs/submit (JSON endpoint for frontend)
func (s *APIServer) handleJobSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wasm      []byte `json:"wasm"`
		Input     string `json:"input"`
		PaymentTx string `json:"paymentTx"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Wasm) == 0 {
		http.Error(w, "WASM code is required", http.StatusBadRequest)
		return
	}

	log.Printf("[API] Job Submit: WASM=%d bytes, Input=%s, TxID=%s", len(req.Wasm), req.Input, req.PaymentTx)

	// Detect language and compile if needed
	lang := compute.DetectLanguage(req.Wasm)
	var wasmCode []byte
	var err error

	switch lang {
case "c":
		log.Printf("[API] Detected C code, compiling to WASM...")
		wasmCode, err = compute.CompileCToWasm(string(req.Wasm))
		if err != nil {
			resp := map[string]interface{}{
				"id":     fmt.Sprintf("job_%d", time.Now().UnixNano()),
				"status": "failed",
				"error":  fmt.Sprintf("Compilation failed: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		log.Printf("[API] Compilation successful, WASM size: %d bytes", len(wasmCode))
	case "wasm":
		wasmCode = req.Wasm
	default:
		http.Error(w, "Unsupported code format. Please provide C source or WASM binary.", http.StatusBadRequest)
		return
	}

	// Find compute nodes
	ctx := r.Context()
	var result []byte

	// Try to find remote compute nodes first
	if s.Node.DHT != nil {
		providers, dhtErr := s.Node.DHT.FindProviders(ctx, "compute-node")
		if dhtErr == nil && len(providers) > 0 {
			// Filter out self
			for _, provider := range providers {
				if provider.ID != s.Node.Host.ID() {
					targetPeer := provider.ID
					if s.Node.Host.Network().Connectedness(targetPeer) != network.Connected {
						s.Node.Host.Connect(ctx, provider)
					}
					result, err = s.Node.SendComputeReq(ctx, targetPeer, wasmCode, []byte(req.Input), req.PaymentTx)
					break
				}
			}
		}
	}

	// Fallback to local execution if no remote peers or remote execution failed
	if result == nil && s.VM != nil {
		log.Printf("[API] No remote peers available, executing locally")
		result, err = s.VM.Run(wasmCode, []byte(req.Input))
	}

	// Build response
	resp := map[string]interface{}{
		"id":     fmt.Sprintf("job_%d", time.Now().UnixNano()),
		"status": "complete",
	}

	if err != nil {
		resp["status"] = "failed"
		resp["error"] = err.Error()
		log.Printf("[API] Job failed: %v", err)
	} else {
		resp["result"] = string(result)
		log.Printf("[API] Job completed successfully")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth handles GET /api/health
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	peers := s.Node.Host.Network().Peers()

	response := map[string]interface{}{
		"status":    "online",
		"nodeId":    s.Node.Host.ID().String(),
		"peers":     len(peers),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
