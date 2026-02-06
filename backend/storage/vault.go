package storage

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

// VaultInterface defines the methods required for storage operations.
type VaultInterface interface {
	Store(key []byte, data []byte) error
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
}

// Vault handles the local, encrypted storage of shards.
type Vault struct {
	db *badger.DB
}

// InitVault opens an encrypted BadgerDB.
// path: Directory to store the DB.
// secretKey: Must be 32 bytes for AES-256 encryption.
func InitVault(path string, secretKey []byte) (*Vault, error) {
	if len(secretKey) != 32 {
		return nil, errors.New("secret key must be 32 bytes (AES-256) for maximum security")
	}

	// Configure BadgerDB with encryption
	// IndexCache is MANDATORY for encrypted workloads in Badger v3+
	opts := badger.DefaultOptions(path).
		WithEncryptionKey(secretKey).
		WithIndexCacheSize(100 << 20).   // 100 MB cache
		WithLoggingLevel(badger.WARNING) // Reduce noise

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open encrypted vault: %w", err)
	}

	return &Vault{db: db}, nil
}

// Close closes the database.
func (v *Vault) Close() error {
	return v.db.Close()
}

// Store saves a blob (shard) securely.
func (v *Vault) Store(key []byte, data []byte) error {
	return v.db.Update(func(txn *badger.Txn) error {
		// In a real system, you might want persistent storage, but for a "compute/cache" user node,
		// we might expire data or keep it indefinitely. Let's keep it indefinitely for now.
		return txn.Set(key, data)
	})
}

// Get retrieves a blob.
func (v *Vault) Get(key []byte) ([]byte, error) {
	var valCopy []byte
	err := v.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err // Key not found or other error
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return valCopy, err
}

// Has checks if a key exists.
func (v *Vault) Has(key []byte) (bool, error) {
	exists := false
	err := v.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			exists = true
			return nil
		}
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	})
	return exists, err
}
