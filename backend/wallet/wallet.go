package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

// Wallet represents a user's keypair
type Wallet struct {
	Private *ecdsa.PrivateKey
	Public  *ecdsa.PublicKey
}

// NewWallet generates a new ECDSA keypair
func NewWallet() *Wallet {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic("Failed to generate key: " + err.Error())
	}
	return &Wallet{Private: private, Public: &private.PublicKey}
}

// SaveFile saves the private key to a file (PEM encoded)
func (w *Wallet) SaveFile(filename string) error {
	x509Encoded, err := x509.MarshalECPrivateKey(w.Private)
	if err != nil {
		return err
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: x509Encoded})
	return os.WriteFile(filename, pemEncoded, 0600)
}

// LoadFile leads a wallet from a file
func LoadFile(filename string) (*Wallet, error) {
	pemEncoded, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemEncoded)
	x509Encoded := block.Bytes
	private, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	return &Wallet{Private: private, Public: &private.PublicKey}, nil
}

// Address returns the public address (Hash of Public Key)
func (w *Wallet) Address() string {
	return PublicKeyToAddress(w.Public)
}

// PublicKeyToAddress converts a public key to a hex address
func PublicKeyToAddress(pub *ecdsa.PublicKey) string {
	// Serialize public key: X || Y
	pubBytes := append(pub.X.Bytes(), pub.Y.Bytes()...)
	// Hash it: SHA256
	hash := sha256.Sum256(pubBytes)
	// Return as Hex string
	return hex.EncodeToString(hash[:])
}

// Sign signs a hash (e.g., Transaction Hash)
func (w *Wallet) Sign(dataHash []byte) (string, error) {
	r, s, err := ecdsa.Sign(rand.Reader, w.Private, dataHash)
	if err != nil {
		return "", err
	}
	// Return signature as "R|S" hex string
	signature := fmt.Sprintf("%x|%x", r, s)
	return signature, nil
}

// VerifySignature checks if a signature is valid for a given hash and public key
func VerifySignature(pub *ecdsa.PublicKey, hash []byte, signature string) bool {
	var r, s big.Int
	// Parse "R|S"
	var rHex, sHex string
	n, _ := fmt.Sscanf(signature, "%s|%s", &rHex, &sHex) // Need to handle pipe split better maybe?

	// Let's use simple string split manually to be safe
	// But actually, Scanf with %x might work if formatted correctly?
	// Let's implement a cleaner parsing manually in the caller or here.

	// Re-implementing parsing with fmt.Sscanf for hex directly
	_, err := fmt.Sscanf(signature, "%x|%x", &r, &s)
	if err != nil || n != 2 {
		// Fallback or fail
		return false
	}

	return ecdsa.Verify(pub, hash, &r, &s)
}
