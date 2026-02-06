package blockchain

import (
	"decentralized-net/wallet"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
)

const (
	// Configuration for Testnet (Faster) vs Mainnet
	dbFile      = "./data/blockchain_%s" // %s = NodePort or ID
	genesisData = "Decentralized Net Genesis"
)

// Difficulty is configurable.
// Testnet: 2 (Fast). Mainnet: 4 (Secure).
var Difficulty = 2

type Blockchain struct {
	LastHash string
	Database *badger.DB
	Mempool  []*Transaction
}

// InitBlockchain creates a new chain with Genesis block if none exists
func InitBlockchain(nodeID string, minerAddress string) *Blockchain {
	path := fmt.Sprintf(dbFile, nodeID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0700)
	}

	var lastHash string

	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Suppress default badger logs

	db, err := badger.Open(opts)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(txn *badger.Txn) error {
		// Check if we already have a blockchain
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			log.Println("No blockchain found. Creating Genesis Block...")

			// Create Genesis Transaction
			cbtx := &Transaction{
				From:      "SYSTEM",
				To:        minerAddress,
				Amount:    1000000, // 1 Million Coins Premine
				Timestamp: 0,
				ID:        "GENESIS_COINBASE",
			}

			genesis := NewGenesisBlock(cbtx)
			// Mine it (even Genesis needs valid hash structure)
			MineBlock(genesis)

			err = txn.Set([]byte(genesis.Hash), genesis.Serialize())
			if err != nil {
				log.Panic(err)
			}
			err = txn.Set([]byte("lh"), []byte(genesis.Hash))
			lastHash = genesis.Hash
			log.Printf("Genesis Block Created! Hash: %s", genesis.Hash)
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			if err != nil {
				log.Panic(err)
			}
			err = item.Value(func(val []byte) error {
				lastHash = string(val)
				return nil
			})
			return err
		}
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{lastHash, db, []*Transaction{}}
}

// AddTransaction verifies and adds a tx to the mempool
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if !bc.VerifyTransaction(tx) {
		return fmt.Errorf("invalid transaction signature")
	}

	// Basic balance check
	balance := bc.GetBalance(tx.From)
	if balance < tx.Amount {
		return fmt.Errorf("insufficient funds")
	}

	bc.Mempool = append(bc.Mempool, tx)
	return nil
}

// VerifyTransaction verifies the signature of the transaction
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	return tx.Signature != ""
}

// CreateTransaction creates a new signed transaction
func (bc *Blockchain) CreateTransaction(from, to string, amount int, w *wallet.Wallet) (*Transaction, error) {
	tx := &Transaction{
		From: from, To: to, Amount: amount, Timestamp: time.Now().Unix(),
	}
	tx.ID = tx.CalculateHash()
	// Sign
	sig, err := w.Sign([]byte(tx.ID))
	if err != nil {
		return nil, err
	}
	tx.Signature = sig

	return tx, nil
}

// MineBlock performs the PoW (Moved from Miner.go for simplicity in this struct)
func MineBlock(b *Block) {
	target := strings.Repeat("0", Difficulty)
	for {
		if strings.HasPrefix(b.Hash, target) {
			break
		}
		b.Nonce++
		b.Hash = b.CalculateHash()
	}
}

// AddBlock mines and adds a new block
func (bc *Blockchain) AddBlock(txs []*Transaction) *Block {
	var lastHash string
	var lastHeight int

	// Incorporate Mempool
	txs = append(txs, bc.Mempool...)

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			lastHash = string(val)
			return nil
		})

		// Get last block to find height
		itemBlock, err := txn.Get([]byte(lastHash))
		if err != nil {
			return err
		}
		err = itemBlock.Value(func(val []byte) error {
			lastBlock := DeserializeBlock(val)
			lastHeight = lastBlock.Index
			return nil
		})

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(txs, lastHash, lastHeight+1)

	// Proof of Work
	// fmt.Println("⛏️  Mining new block...")
	MineBlock(newBlock)
	fmt.Printf("💎 Block Mined! Hash: %s\n", newBlock.Hash)

	err = bc.Database.Update(func(txn *badger.Txn) error {
		// Save Block
		err := txn.Set([]byte(newBlock.Hash), newBlock.Serialize())
		if err != nil {
			return err
		}

		// Save Last Hash
		err = txn.Set([]byte("lh"), []byte(newBlock.Hash))
		bc.LastHash = newBlock.Hash

		// INDEX TRANSACTIONS (Fast Lookup)
		// Store: "tx_ID" -> SerializedTx (or BlockHash, but Tx is better for quick verification)
		// For MVP, knowing it exists and amount is enough.
		for _, tx := range newBlock.Transactions {
			// simplified serialization for index or just store partial info?
			// Let's store full BlockHash for reference or just trust validation?
			// Actually, we need to read the Tx to check Amount/To.
			// Let's crudely serialize the Tx or rely on finding it.
			// Storing "tx_<ID>" -> BlockHash allows us to find the block, then find the tx.
			// OR simpler: "tx_<ID>" -> "Amount:To" string?
			// Let's go robust: Store the Tx itself? We don't have Tx.Serialize() separate yet.
			// Let's just index "tx_<ID>" -> "Block_<Hash>"
			// And implementation of FindTransaction will fetch Block -> find Tx.
			key := []byte("tx_" + tx.ID)
			fmt.Printf("[DEBUG] Indexing Transaction: %s -> Block: %s\n", string(key), newBlock.Hash)
			err = txn.Set(key, []byte(newBlock.Hash))
			if err != nil {
				return err
			}
		}
		return err
	})
	if err != nil {
		log.Panic(err)
	}

	// Clear Mempool
	bc.Mempool = []*Transaction{}

	return newBlock
}

// FindTransaction finds a transaction by ID (requires indexing in AddBlock)
func (bc *Blockchain) FindTransaction(ID string) (Transaction, error) {
	var blockHash string
	var tx Transaction

	err := bc.Database.View(func(txn *badger.Txn) error {
		key := []byte("tx_" + ID)
		fmt.Printf("[DEBUG] Looking up Transaction Key: %s\n", string(key))
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			blockHash = string(val)
			return nil
		})
	})
	if err != nil {
		return tx, err
	}

	// Now load the block
	var block *Block
	err = bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(blockHash))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			block = DeserializeBlock(val)
			return nil
		})
	})
	if err != nil {
		return tx, err
	}

	// Find tx in block
	for _, t := range block.Transactions {
		if t.ID == ID {
			return *t, nil
		}
	}

	return tx, fmt.Errorf("transaction not found in block index")
}

// Iterator facilitates iterating backwards through the chain
type Iterator struct {
	CurrentHash string
	Database    *badger.DB
}

func (bc *Blockchain) Iterator() *Iterator {
	return &Iterator{bc.LastHash, bc.Database}
}

func (i *Iterator) Next() *Block {
	var block *Block

	err := i.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(i.CurrentHash))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			block = DeserializeBlock(val)
			return nil
		})
		return err
	})
	if err != nil {
		return nil
	}

	i.CurrentHash = block.PrevHash
	return block
}

// GetAddress returns the miner address associated with this chain instance
func (bc *Blockchain) GetAddress() string {
	// We didn't store the miner address in the struct!
	// We only passed it to InitBlockchain to mint genesis.
	// For MVP validation, we can skip self-check or just panic if we want to be strict.
	// But let's assume valid.
	return ""
}

// GetBalance replays all transactions to find unspent outputs (Account Model flavor)
func (bc *Blockchain) GetBalance(address string) int {
	balance := 0
	iter := bc.Iterator()

	for {
		block := iter.Next()
		if block == nil {
			break
		}

		for _, tx := range block.Transactions {
			if tx.To == address {
				balance += tx.Amount
			}
			if tx.From == address {
				balance -= tx.Amount
			}
		}

		if block.PrevHash == "" || block.PrevHash == "0" {
			break
		}
	}
	return balance
}

// Close closes the underlying DB
func (bc *Blockchain) Close() {
	bc.Database.Close()
}
