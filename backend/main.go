package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"decentralized-net/api"
	"decentralized-net/blockchain"
	"decentralized-net/compute"
	"decentralized-net/p2p"
	"decentralized-net/storage"
	"decentralized-net/wallet"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {
	// 1. Define Global Flags
	port := flag.Int("port", 0, "Port to listen on (0 for random)")
	vaultPath := flag.String("vault", "./data/vault", "Path to secure storage vault")
	mode := flag.String("mode", "full", "Node mode: full, storage, or compute")
	peerAddr := flag.String("peer", "", "Bootstrap peer address to connect to")
	apiPort := flag.Int("api-port", 0, "Port for HTTP API Gateway (e.g., 8080)")

	// 2. Parse Global Flags
	flag.Parse()

	// ---------------------------------------------------------
	// CLI Handling (Decision Logic)
	// ---------------------------------------------------------
	args := flag.Args()
	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\nShutdown signal received. Exiting...")
		cancel()
		os.Exit(0)
	}()

	switch command {
	case "wallet":
		handleWalletCmd(port)
	case "run-job":
		handleRunJobCmd(ctx, args[1:], peerAddr)
	case "pay":
		// Pay requires blockchain access (for nonces/balance).
		// If the node is running, the DB is locked.
		// For this MVP, we will try to open it. If locked, we warn the user.
		handlePayCmd(port, args[1:])
	case "upload":
		handleUploadCmd(ctx, port, vaultPath, peerAddr, args[1:])
	case "download":
		handleDownloadCmd(ctx, port, vaultPath, peerAddr, args[1:])
	case "mine":
		// Mine is a server-side activity usually, but exposed as CLI.
		// It creates a full node.
		startFullNode(ctx, port, vaultPath, mode, peerAddr, apiPort, true)
	default:
		// No command -> Start Full Node
		startFullNode(ctx, port, vaultPath, mode, peerAddr, apiPort, false)
	}
}

// ---------------------------------------------------------
// Command Handlers (Refactored)
// ---------------------------------------------------------

func handleWalletCmd(port *int) {
	walletPath := fmt.Sprintf("./data/wallet_%d.dat", *port)
	if *port == 0 {
		walletPath = "./data/wallet_default.dat"
	}

	w, err := wallet.LoadFile(walletPath)
	if err != nil {
		log.Println("No existing wallet found. Creating new...")
		w = wallet.NewWallet()
		if err := w.SaveFile(walletPath); err != nil {
			log.Fatalf("Failed to save wallet: %v", err)
		}
	}
	fmt.Printf("Wallet Address: %s\n", w.Address())
}

func handleRunJobCmd(ctx context.Context, args []string, bootPeer *string) {
	// Lightweight P2P Node (No Chain, No Vault)
	jobCmd := flag.NewFlagSet("run-job", flag.ExitOnError)
	wasmFile := jobCmd.String("wasm", "", "WASM file to execute")
	inputText := jobCmd.String("input", "", "Input string data")
	targetID := jobCmd.String("target", "", "Specific Peer ID to send job to (optional)")
	txID := jobCmd.String("tx", "", "Transaction ID for payment")
	// Allow --peer to be specified AFTER the subcommand
	subPeer := jobCmd.String("peer", "", "Bootstrap peer address")

	if err := jobCmd.Parse(args); err != nil {
		log.Fatalf("Failed to parse run-job flags: %v", err)
	}

	if *wasmFile == "" {
		log.Fatal("Please specify --wasm")
	}

	// Initialize Lightweight P2P Node (Random Port)
	log.Println("[CLI] Starting lightweight P2P client...")
	node, err := p2p.NewNode(ctx, 0) // 0 = Random Port
	if err != nil {
		log.Fatalf("Failed to start P2P client: %v", err)
	}

	// Bootstrapping
	// Prioritize subcommand flag, then global flag
	effectivePeer := ""
	if *subPeer != "" {
		effectivePeer = *subPeer
	} else if *bootPeer != "" {
		effectivePeer = *bootPeer
	}

	if effectivePeer != "" {
		node.EnableDHT([]string{effectivePeer})
	} else {
		// Try to look for local defaults if not specified?
		// For now, assume user provides --peer OR we rely on mDNS (if enabled in libp2p)
		node.EnableDHT(nil)
	}
	// Give DHT a moment
	time.Sleep(1 * time.Second)

	log.Printf("Reading WASM: %s", *wasmFile)
	wasmCode, err := os.ReadFile(*wasmFile)
	if err != nil {
		log.Fatalf("Failed to read wasm file: %v", err)
	}

	// Discovery Logic
	var targetPeer peer.ID
	if *targetID != "" {
		id, err := peer.Decode(*targetID)
		if err != nil {
			log.Fatalf("Invalid target peer ID: %v", err)
		}
		targetPeer = id
	} else {
		log.Println("No --target specified. Searching network for 'compute-node'...")
		providers, err := node.DHT.FindProviders("compute-node")
		if err != nil || len(providers) == 0 {
			log.Fatal("No compute nodes found. Ensure the server is running.")
		}
		targetPeer = providers[0].ID
		log.Printf("Found Compute Node: %s", targetPeer)
		node.Host.Connect(ctx, providers[0])
	}

	log.Printf("Sending job to %s...", targetPeer)
	result, err := node.SendComputeReq(ctx, targetPeer, wasmCode, []byte(*inputText), *txID)
	if err != nil {
		log.Fatalf("Job Execution Failed: %v", err)
	}

	log.Println("------------------------------------------------")
	log.Printf("REMOTE RESULT:\n%s", string(result))
	log.Println("------------------------------------------------")
}

func handlePayCmd(port *int, args []string) {
	// Re-uses full node logic partially but fails if locked.
	// For MVP: Must open chain to create valid TX.
	walletPath := fmt.Sprintf("./data/wallet_%d.dat", *port)
	if *port == 0 {
		walletPath = "./data/wallet_default.dat"
	}
	w, err := wallet.LoadFile(walletPath)
	if err != nil {
		log.Fatalf("Wallet not found: %v", err)
	}

	payCmd := flag.NewFlagSet("pay", flag.ExitOnError)
	toAddr := payCmd.String("to", "", "Recipient Address")
	amount := payCmd.Int("amount", 0, "Amount to send")
	if err := payCmd.Parse(args); err != nil {
		log.Fatalf("Failed flags: %v", err)
	}

	// Try Open Chain
	nodeID := "random"
	if *port != 0 {
		nodeID = fmt.Sprintf("%d", *port)
	}

	// WARNING: This will fail if server is running.
	// The user must stop server to use 'pay' via CLI in this architecture,
	// OR we need a "Remote Wallet" feature (skipped for this phase).
	chain := blockchain.InitBlockchain(nodeID, w.Address())
	defer chain.Close()

	tx, err := chain.CreateTransaction(w.Address(), *toAddr, *amount, w)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	chain.AddTransaction(tx)
	log.Printf("Transaction Created: %s", tx.ID)

	// Auto-mine to confirm
	newBlock := chain.AddBlock([]*blockchain.Transaction{})
	log.Printf("Confirmed in Block #%d", newBlock.Index)
}

func handleUploadCmd(ctx context.Context, port *int, vaultPath *string, peerAddr *string, args []string) {
	// Upload needs a full node (Vault + P2P) to shard and distribute
	node, vault, _, _, err := setupNode(ctx, port, vaultPath, peerAddr, nil, nil)
	if err != nil {
		log.Fatalf("Failed to setup node for upload: %v", err)
	}

	uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
	fileToUpload := uploadCmd.String("file", "", "File to upload")

	if err := uploadCmd.Parse(args); err != nil {
		log.Fatalf("Failed to parse upload flags: %v", err)
	}

	if *fileToUpload == "" {
		log.Fatal("Please specify --file")
	}

	log.Printf("Uploading file: %s", *fileToUpload)

	// Read file
	data, err := os.ReadFile(*fileToUpload)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	fileSize := len(data)

	// Shard it (10 data, 4 parity)
	sm, err := storage.NewShardManager(10, 4)
	if err != nil {
		log.Fatalf("Failed to init sharding: %v", err)
	}

	shards, err := sm.Encode(data)
	if err != nil {
		log.Fatalf("Sharding failed: %v", err)
	}

	log.Printf("File split into %d shards.", len(shards))

	// Distributed Storage Logic
	// Give DHT/Peers a moment to connect if bootstrapping
	time.Sleep(2 * time.Second)

	peers := node.Host.Network().Peers()
	allNodes := append(peers, node.Host.ID()) // Include self
	log.Printf("[P2P] Distributing shards across %d nodes...", len(allNodes))

	for i, shard := range shards {
		targetPeer := allNodes[i%len(allNodes)]
		key := []byte(fmt.Sprintf("%s-shard-%d", *fileToUpload, i))

		if targetPeer == node.Host.ID() {
			err := vault.Store(key, shard)
			if err != nil {
				log.Printf("Failed to store shard %d locally: %v", i, err)
			} else {
				log.Printf("Shard %d -> SELF (Stored)", i)
				if node.DHT != nil {
					go node.DHT.Announce(string(key))
				}
			}
		} else {
			log.Printf("Shard %d -> Sending to %s...", i, targetPeer)
			err := node.SendStoreReq(ctx, targetPeer, key, shard)
			if err != nil {
				log.Printf("Failed to send shard %d to %s: %v", i, targetPeer, err)
			} else {
				log.Printf("Shard %d -> %s (ACK Received)", i, targetPeer)
			}
		}
	}

	log.Printf("Upload Complete! original_size=%d", fileSize)
	// We don't block here because it's a CLI command.
	// However, if we uploaded to ourselves, we might want to keep running to serve it?
	// For a CLI tool, typically you upload and exit. The "Daemon" node should be running separately if you want to serve.
	// But in this P2P design, *we* are a node.
	// If we exit, our local shards become unavailable immediately.
	// So we should probably block IF we stored anything locally.
	log.Println("Node remaining active to serve stored shards. Press Ctrl+C to exit.")
	select {}
}

func handleDownloadCmd(ctx context.Context, port *int, vaultPath *string, peerAddr *string, args []string) {
	node, vault, _, _, err := setupNode(ctx, port, vaultPath, peerAddr, nil, nil)
	if err != nil {
		log.Fatalf("Failed to setup node for download: %v", err)
	}

	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	fileToDownload := downloadCmd.String("file", "", "File to download/reconstruct")
	size := downloadCmd.Int("size", 0, "Original file size")

	if err := downloadCmd.Parse(args); err != nil {
		log.Fatalf("Failed to parse download flags: %v", err)
	}

	if *fileToDownload == "" || *size == 0 {
		log.Fatal("Please specify --file and --size")
	}

	log.Printf("Attempting to reconstruct: %s", *fileToDownload)
	time.Sleep(2 * time.Second) // Wait for connections

	shards := make([][]byte, 14)
	foundCount := 0

	// Try to find shards
	for i := 0; i < 14; i++ {
		keyStr := fmt.Sprintf("%s-shard-%d", *fileToDownload, i)
		key := []byte(keyStr)

		// 1. Try Local
		data, err := vault.Get(key)
		if err == nil {
			log.Printf("Shard %d found locally.", i)
			shards[i] = data
			foundCount++
			continue
		}

		// 2. Try DHT
		if node.DHT != nil {
			log.Printf("Shard %d missing. Querying DHT...", i)
			providers, err := node.DHT.FindProviders(keyStr)
			if err == nil && len(providers) > 0 {
				log.Printf("Found provider for shard %d: %s", i, providers[0].ID)
				// Simplified: Just proving discovery works for MVP.
				// Implementing full retrieval requires a new P2P protocol handler.
			}
		}
	}

	if foundCount < 10 {
		log.Fatalf("Cannot reconstruct. Only found %d/10 required shards (mostly local for now).", foundCount)
	}

	// Reconstruct
	sm, err := storage.NewShardManager(10, 4)
	if err != nil {
		log.Fatalf("Failed to init sharding: %v", err)
	}

	reconstructedData, err := sm.Reconstruct(shards, *size)
	if err != nil {
		log.Fatalf("Reconstruction failed: %v", err)
	}

	outputFile := "restored_" + *fileToDownload
	err = os.WriteFile(outputFile, reconstructedData, 0644)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	log.Printf("Success! File restored to: %s", outputFile)
}

func startFullNode(ctx context.Context, port *int, vaultPath *string, mode *string, peerAddr *string, apiPort *int, isMining bool) {
	node, _, chain, myAddress, err := setupNode(ctx, port, vaultPath, peerAddr, mode, apiPort)
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Mining Loop (if isMining is true)
	if isMining {
		log.Printf("Starting Miner... Address: %s", myAddress)
		for {
			cbTx := &blockchain.Transaction{
				From: "SYSTEM", To: myAddress,
				Amount: 50, Timestamp: time.Now().Unix(),
				ID: fmt.Sprintf("COINBASE_%d", time.Now().UnixNano()),
			}
			block := chain.AddBlock([]*blockchain.Transaction{cbTx})
			if err := node.BroadcastBlock(block); err != nil {
				log.Printf("[Miner] Failed broadcast: %v", err)
			} else {
				log.Printf("[Miner] Mined Block #%d", block.Index)
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		// Server Mode - Block Forever
		select {}
	}
}

// setupNode handles the heavy lifting of initializing Crypto, Vault, and P2P
func setupNode(ctx context.Context, port *int, vaultPath *string, peerAddr *string, mode *string, apiPort *int) (*p2p.Node, *storage.Vault, *blockchain.Blockchain, string, error) {
	// 1. Wallet
	walletPath := fmt.Sprintf("./data/wallet_%d.dat", *port)
	if *port == 0 {
		walletPath = "./data/wallet_default.dat"
	}
	var w *wallet.Wallet
	if wLoaded, err := wallet.LoadFile(walletPath); err == nil {
		w = wLoaded
	} else {
		w = wallet.NewWallet()
		w.SaveFile(walletPath)
	}
	log.Printf("[Crypto] Wallet Address: %s", w.Address())

	// 2. Blockchain
	nodeID := fmt.Sprintf("%d", *port)
	if *port == 0 {
		nodeID = "random"
	}
	chain := blockchain.InitBlockchain(nodeID, w.Address())
	log.Printf("[Blockchain] Initialized. Tip Hash: %s", chain.LastHash)

	// 3. Vault
	secretKey := []byte("12345678901234567890123456789012")
	if _, err := os.Stat(*vaultPath); os.IsNotExist(err) {
		os.MkdirAll(*vaultPath, 0700)
	}
	vault, err := storage.InitVault(*vaultPath, secretKey)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("vault init failed: %v", err)
	}
	log.Printf("[Storage] Secured Vault initialized at %s", *vaultPath)

	// 4. P2P Node
	node, err := p2p.NewNode(ctx, *port)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("p2p node init failed: %v", err)
	}
	node.Chain = chain
	log.Printf("[P2P] Node Online! ID: %s", node.Host.ID())

	// 5. Handlers
	node.HandleStoreStream(vault)
	node.SetupBlockPropagation()

	// 6. Bootstrapping
	var bootstrapPeers []string
	if peerAddr != nil && *peerAddr != "" {
		log.Printf("[P2P] Bootstrapping from %s", *peerAddr)
		bootstrapPeers = append(bootstrapPeers, *peerAddr)
	}
	node.EnableDHT(bootstrapPeers)
	log.Println("[P2P] Kademlia DHT Started!")

	// 7. Compute Mode
	computeMode := "full"
	if mode != nil {
		computeMode = *mode
	}
	if computeMode == "full" || computeMode == "compute" {
		go func() {
			time.Sleep(5 * time.Second)
			node.DHT.Announce("compute-node")
		}()
		vm := compute.NewVM(ctx)
		// Note: We don't defer close here easily, caller must handle context cancellation
		log.Println("[Compute] VM Ready")
		node.HandleComputeStream(vm)
	}

	// 8. API
	if apiPort != nil && *apiPort > 0 {
		api.StartAPIServer(node, vault, *apiPort)
	}

	return node, vault, chain, w.Address(), nil
}
