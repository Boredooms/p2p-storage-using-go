package storage

import (
	"crypto/rand"
	"fmt"
	"os"
)

// LoadOrGenerateKey checks if a key file exists.
// If it does, it loads and verifies it.
// If not, it generates a new 32-byte secure key and saves it.
func LoadOrGenerateKey(keyPath string) ([]byte, error) {
	// 1. Check if key exists
	if _, err := os.Stat(keyPath); err == nil {
		// Key exists, load it
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}

		if len(data) != 32 {
			return nil, fmt.Errorf("invalid key length in %s: expected 32 bytes, got %d", keyPath, len(data))
		}

		return data, nil
	}

	// 2. Generate new key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// 3. Save key
	// 0600 = Read/Write for owner only (Secure)
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save key file: %w", err)
	}

	return key, nil
}
