// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// KeyManager handles wallet private keys and provides functionality for key management,
// address derivation, and signing operations. It securely stores the private key and
// provides methods to interact with it safely.
type KeyManager struct {
	privateKey *ecdsa.PrivateKey // The wallet's private key
	address    common.Address    // The derived Ethereum address
}

// NewKeyManager creates a new key manager from a private key string.
// It accepts a hex-encoded private key (with or without 0x prefix) and returns
// an initialized KeyManager instance.
//
// Parameters:
//   - privateKeyHex: Hex-encoded private key string (with optional 0x prefix)
//
// Returns:
//   - *KeyManager: Initialized key manager instance
//   - error: Error if private key is invalid or empty
//
// Example:
//
//	km, err := NewKeyManager("0x1234...")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	address := km.GetAddress()
func NewKeyManager(privateKeyHex string) (*KeyManager, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}

	// Remove "0x" prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &KeyManager{
		privateKey: privateKey,
		address:    address,
	}, nil
}

// GetAddress returns the Ethereum address associated with this key manager.
// The address is derived from the public key corresponding to the private key.
//
// Returns:
//   - common.Address: The Ethereum address for this wallet
func (km *KeyManager) GetAddress() common.Address {
	return km.address
}

// Sign signs the provided data using the wallet's private key.
// It first hashes the data using Keccak256 and then signs the hash.
//
// Parameters:
//   - data: Raw data bytes to sign
//
// Returns:
//   - []byte: The signature bytes
//   - error: Error if signing fails
func (km *KeyManager) Sign(data []byte) ([]byte, error) {
	return crypto.Sign(crypto.Keccak256Hash(data).Bytes(), km.privateKey)
}
