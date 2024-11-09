// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"math/big"
)

// GasStrategy defines parameters for gas price management and transaction retry behavior.
// It allows configuring maximum gas prices, priority fees, and retry strategies when
// gas prices are high.
type GasStrategy struct {
	// MaxGasPrice is the maximum gas price willing to pay for transactions in wei
	MaxGasPrice *big.Int

	// PriorityFee is the tip paid to validators in wei to prioritize the transaction
	PriorityFee *big.Int

	// RetryOnHighGas indicates whether to retry transactions when gas price exceeds MaxGasPrice
	RetryOnHighGas bool

	// WaitForLowerGas indicates whether to wait for gas prices to decrease before retrying
	WaitForLowerGas bool
}

// DefaultGasStrategy returns a default gas strategy with conservative settings.
// The default strategy includes:
//   - MaxGasPrice of 100 gwei to prevent overpaying for transactions
//   - PriorityFee of 1.5 gwei to incentivize validators
//   - Retry enabled when gas prices are high
//   - Waiting enabled for gas prices to decrease
//
// Example usage:
//
//	strategy := DefaultGasStrategy()
//	client.SetGasStrategy(strategy)
func DefaultGasStrategy() *GasStrategy {
	return &GasStrategy{
		MaxGasPrice:     big.NewInt(100000000000), // 100 gwei
		PriorityFee:     big.NewInt(1500000000),   // 1.5 gwei
		RetryOnHighGas:  true,
		WaitForLowerGas: true,
	}
}
