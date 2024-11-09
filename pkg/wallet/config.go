package wallet

import (
	"math/big"
	"time"
)

// NetworkConfig holds network-specific configuration parameters for blockchain interactions.
// It defines settings like gas prices, retry logic, and network identifiers that control
// how transactions are processed on different networks.
type NetworkConfig struct {
	// Type identifies which blockchain network this config is for (e.g. ETH, BSC)
	Type NetworkType

	// RPCURL is the HTTP(S) endpoint for connecting to the network
	RPCURL string

	// ChainID is the unique identifier for the blockchain network
	ChainID int64

	// MaxRetries specifies how many times to retry failed transactions
	MaxRetries int

	// RetryDelay is the duration to wait between retry attempts
	RetryDelay time.Duration

	// GasLimitMultiplier is used to add a safety buffer to estimated gas
	// For example, 1.2 adds 20% to the estimated gas limit
	GasLimitMultiplier float64

	// MaxGasPrice sets an upper bound on gas price to prevent overpaying
	// Transactions will not be sent if gas price exceeds this value
	MaxGasPrice *big.Int
}

// DefaultNetworkConfigs returns pre-configured settings for supported blockchain networks.
// It provides sensible defaults for Ethereum mainnet, Base, and Binance Smart Chain.
//
// The defaults include:
// - Conservative gas price limits
// - 3 retry attempts with 1 second delay
// - 20% buffer on gas estimates
//
// Example usage:
//
//	configs := DefaultNetworkConfigs()
//	ethConfig := configs[0] // Ethereum mainnet config
func DefaultNetworkConfigs() []NetworkConfig {
	return []NetworkConfig{
		{
			Type:               ETH,
			ChainID:            1,
			MaxRetries:         3,
			RetryDelay:         time.Second,
			GasLimitMultiplier: 1.2,
			MaxGasPrice:        big.NewInt(300000000000), // 300 gwei
		},
		{
			Type:               BASE,
			ChainID:            8453,
			MaxRetries:         3,
			RetryDelay:         time.Second,
			GasLimitMultiplier: 1.2,
			MaxGasPrice:        big.NewInt(100000000000), // 100 gwei
		},
		{
			Type:               BSC,
			ChainID:            56,
			MaxRetries:         3,
			RetryDelay:         time.Second,
			GasLimitMultiplier: 1.2,
			MaxGasPrice:        big.NewInt(5000000000), // 5 gwei
		},
	}
}
