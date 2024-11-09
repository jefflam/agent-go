// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// addressRegex is a regular expression for validating the basic format of Ethereum-style addresses.
	// It checks for a "0x" prefix followed by exactly 40 hexadecimal characters.
	addressRegex = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
)

// ValidateAddress validates a blockchain address for a specific network.
// It performs format validation, checksum verification, and network-specific checks.
//
// Parameters:
//   - network: The blockchain network type (e.g., ETH, BSC, BASE)
//   - address: The address string to validate
//
// Returns:
//   - error: nil if the address is valid, otherwise returns a WalletError with details
//
// Example:
//
//	err := client.ValidateAddress(NetworkEthereum, "0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) ValidateAddress(network NetworkType, address string) error {
	// Basic format validation
	if !addressRegex.MatchString(address) {
		return NewWalletError(
			ErrCodeInvalidAddress,
			"invalid address format",
			nil,
			network,
		)
	}

	// Convert to checksum address
	checksumAddr := common.HexToAddress(address).Hex()

	// If the address was provided with checksum, verify it matches
	if address != strings.ToLower(address) && address != checksumAddr {
		return NewWalletError(
			ErrCodeInvalidAddress,
			"invalid address checksum",
			nil,
			network,
		)
	}

	// Network-specific validation
	switch network {
	case ETH, BASE:
		// ETH and Base use the same address format
		return nil
	case BSC:
		// BSC uses the same format as ETH
		return nil
	default:
		return NewWalletError(
			ErrCodeInvalidNetwork,
			fmt.Sprintf("unsupported network: %s", network),
			nil,
			network,
		)
	}
}
