// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"context"
	"sync"
	"time"
)

// NonceManager manages transaction nonces across different blockchain networks.
// It tracks used and pending nonces to prevent nonce conflicts and ensure proper
// transaction ordering. The manager is thread-safe and handles concurrent access
// through mutex locking.
type NonceManager struct {
	nonces        map[NetworkType]uint64               // Tracks the latest nonce per network
	pendingNonces map[NetworkType]map[uint64]time.Time // Tracks pending nonces and when they were issued
	mu            sync.RWMutex                         // Mutex for thread-safe access
}

// newNonceManager creates a new nonce manager instance.
// It initializes the internal maps for tracking nonces across networks.
//
// Returns:
//   - *NonceManager: A new nonce manager instance
func newNonceManager() *NonceManager {
	return &NonceManager{
		nonces:        make(map[NetworkType]uint64),
		pendingNonces: make(map[NetworkType]map[uint64]time.Time),
	}
}

// GetNonce gets the next available nonce for the specified network.
// It queries the network for the current nonce and tracks pending nonces
// to prevent conflicts. If a nonce is already pending, it increments until
// finding an available one.
//
// Parameters:
//   - ctx: Context for the operation
//   - client: Client instance for network interaction
//   - network: Target blockchain network
//
// Returns:
//   - uint64: Next available nonce
//   - error: Error if nonce retrieval fails
func (nm *NonceManager) GetNonce(ctx context.Context, client *Client, network NetworkType) (uint64, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	ethClient, _, err := client.getClientAndConfig(network)
	if err != nil {
		return 0, NewWalletError(ErrCodeInvalidNetwork, "failed to get network client", err, network)
	}

	// Get the current nonce from the network
	nonce, err := ethClient.PendingNonceAt(ctx, client.keyManager.GetAddress())
	if err != nil {
		return 0, NewWalletError(ErrCodeRPCError, "failed to get nonce", err, network)
	}

	// Initialize pending nonces map if not exists
	if nm.pendingNonces[network] == nil {
		nm.pendingNonces[network] = make(map[uint64]time.Time)
	}

	// Find the next available nonce
	for {
		if _, isPending := nm.pendingNonces[network][nonce]; !isPending {
			nm.pendingNonces[network][nonce] = time.Now()
			return nonce, nil
		}
		nonce++
	}
}

// ReleaseNonce releases a previously used nonce, making it available for reuse.
// This should be called after a transaction is confirmed or fails permanently.
//
// Parameters:
//   - network: Network the nonce was used on
//   - nonce: The nonce value to release
func (nm *NonceManager) ReleaseNonce(network NetworkType, nonce uint64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.pendingNonces[network] != nil {
		delete(nm.pendingNonces[network], nonce)
	}
}
