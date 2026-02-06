package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"time"
)

// Transaction represents a transfer of coins
type Transaction struct {
	From      string // Sender Address
	To        string // Recipient Address
	Amount    int    // Value
	Timestamp int64  // Time created
	Signature string // Cryptographic Signature of Sender
	ID        string // Hash of the Tx (calculated)
}

// CalculateHash generates the ID for the transaction
func (tx *Transaction) CalculateHash() string {
	record := fmt.Sprintf("%s%s%d%d", tx.From, tx.To, tx.Amount, tx.Timestamp)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// Block represents a secured batch of transactions
type Block struct {
	Index        int
	Timestamp    int64
	Transactions []*Transaction
	PrevHash     string
	Hash         string
	Nonce        int
}

// Serialize converts the block to bytes
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock decodes bytes into a Block
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}

// CalculateHash generates the hash of the block
func (b *Block) CalculateHash() string {
	// Simple verification hash (concatenating critical fields)
	// In production, use Merkle Root for Transactions
	txData := ""
	for _, tx := range b.Transactions {
		txData += tx.ID
	}
	record := fmt.Sprintf("%d%d%s%s%d", b.Index, b.Timestamp, txData, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// NewBlock creates a new block
func NewBlock(txs []*Transaction, prevHash string, height int) *Block {
	block := &Block{
		Index:        height,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PrevHash:     prevHash,
		Nonce:        0,
	}
	// Hash is calculated during Mining, not here.
	// But we set a placeholder.
	block.Hash = block.CalculateHash()
	return block
}

// NewGenesisBlock creates the first block
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, "0", 0)
}
