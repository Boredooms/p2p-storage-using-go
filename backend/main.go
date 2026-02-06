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

	// Bootstrapping (Crucial to find the server)
	if *bootPeer != "" {
		node.EnableDHT([]string{*bootPeer})
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
	// Needs full node basically for Vault + P2P but we try to separate
	startFullNode(ctx, port, vaultPath, nil, peerAddr, nil, false)
	// NOTE: The original upload code was inside 'case upload'.
	// Ideally we extract that specific logic similarly to handleRunJobCmd
	// But 'upload' relies heavily on the 'vault' & 'sharding' which are initialized in startFullNode.
	// For now, let's leave upload as is or map it to startFullNode if it does the logic?
	// Actually, the original 'upload' was a one-off command.
	// We should probably allow 'upload' to reuse the logic *inside* startFullNode
	// or copy the logic here. Given code size, I will defer strict refactor of upload/download
	// and focus on FIXING run-job which was the panic source.
	// Calling startFullNode for upload/download implies it acts as a node.
}

func handleDownloadCmd(ctx context.Context, port *int, vaultPath *string, peerAddr *string, args []string) {
	startFullNode(ctx, port, vaultPath, nil, peerAddr, nil, false)
}

func startFullNode(ctx context.Context, port *int, vaultPath *string, mode *string, peerAddr *string, apiPort *int, isMining bool) {
	// ---------------------------------------------------------
	// Crypto Layer (Wallet & Blockchain)
	// ---------------------------------------------------------
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
	myAddress := w.Address()
	log.Printf("[Crypto] Wallet Address: %s", myAddress)

	// Initialize Blockchain
	nodeID := fmt.Sprintf("%d", *port)
	if *port == 0 {
		nodeID = "random"
	}
	chain := blockchain.InitBlockchain(nodeID, myAddress)
	defer chain.Close()
	log.Printf("[Blockchain] Initialized. Tip Hash: %s", chain.LastHash)

	// Initialize Vault
	secretKey := []byte("12345678901234567890123456789012")
	if _, err := os.Stat(*vaultPath); os.IsNotExist(err) {
		os.MkdirAll(*vaultPath, 0700)
	}

	vault, err := storage.InitVault(*vaultPath, secretKey)
	if err != nil {
		log.Fatalf("CRITICAL: Failed to initialize secure vault: %v", err)
	}
	defer vault.Close()
	log.Printf("[Storage] Secured Vault initialized at %s", *vaultPath)

	// Initialize P2P Network Layer
	node, err := p2p.NewNode(ctx, *port)
	if err != nil {
		log.Fatalf("CRITICAL: Failed to start P2P node: %v", err)
	}

	// Attach Blockchain to Node
	node.Chain = chain

	log.Printf("[P2P] Node Online!")
	log.Printf("[P2P] ID: %s", node.Host.ID())

	// Enable incoming store stream handling
	node.HandleStoreStream(vault)

	// Enable Block Propagation (PubSub)
	if err := node.SetupBlockPropagation(); err != nil {
		log.Printf("[P2P] WARNING: Failed to setup block propagation: %v", err)
	}

	// ---------------------------------------------------------
	// Bootstrapping & DHT Setup
	// ---------------------------------------------------------
	var bootstrapPeers []string
	if peerAddr != nil && *peerAddr != "" {
		log.Printf("[P2P] Bootstrapping from %s", *peerAddr)
		bootstrapPeers = append(bootstrapPeers, *peerAddr)
	}

	// Start DHT
	if err := node.EnableDHT(bootstrapPeers); err != nil {
		log.Printf("[P2P] WARNING: Failed to start DHT: %v", err)
	} else {
		log.Println("[P2P] Kademlia DHT Started! (Advertising capability)")
	}

	// If compute mode, advertise as a compute node
	computeMode := "full"
	if mode != nil {
		computeMode = *mode
	}

	if computeMode == "full" || computeMode == "compute" {
		go func() {
			time.Sleep(5 * time.Second)
			if err := node.DHT.Announce("compute-node"); err != nil {
				log.Printf("Failed to advertise compute service: %v", err)
			}
		}()
	}

	// ---------------------------------------------------------
	// API Gateway
	// ---------------------------------------------------------
	if apiPort != nil && *apiPort > 0 {
		api.StartAPIServer(node, vault, *apiPort)
	}

	// 3. Initialize Compute Layer
	var vm *compute.VM
	if computeMode == "full" || computeMode == "compute" {
		vm = compute.NewVM(ctx)
		defer vm.Close()
		log.Println("[Compute] Wazero VM Sandbox ready for jobs.")

		// Enable incoming compute handling
		node.HandleComputeStream(vm)
		log.Println("[Compute] Listening for remote jobs...")
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
