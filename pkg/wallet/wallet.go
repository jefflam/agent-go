// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

// NetworkType represents supported EVM networks
type NetworkType string

const (
	// ETH represents the Ethereum mainnet network
	ETH NetworkType = "ETH"
	// BASE represents the Base network
	BASE NetworkType = "BASE"
	// BSC represents the Binance Smart Chain network
	BSC NetworkType = "BSC"
)

// Client represents an EVM wallet client that manages connections and interactions
// with multiple blockchain networks. It handles transaction signing, nonce management,
// and network-specific configurations.
type Client struct {
	clients      map[NetworkType]*ethclient.Client
	configs      map[NetworkType]NetworkConfig
	keyManager   *KeyManager
	nonceManager *NonceManager
	mu           sync.RWMutex
	log          *logrus.Logger
}

// NewClient creates a new wallet client with the provided configurations and private key.
// It establishes connections to all configured networks and initializes the key manager.
//
// Parameters:
//   - ctx: Context for initialization operations
//   - log: Logger instance for client operations
//   - configs: Network configurations for supported chains
//   - privateKey: Private key for transaction signing
//
// Returns:
//   - *Client: Initialized wallet client
//   - error: Error if initialization fails
//
// Example:
//
//	configs := []NetworkConfig{
//	    {Type: ETH, RPCURL: "https://eth-mainnet.example.com"},
//	    {Type: BSC, RPCURL: "https://bsc-mainnet.example.com"},
//	}
//	client, err := NewClient(ctx, logger, configs, privateKey)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewClient(ctx context.Context, log *logrus.Logger, configs []NetworkConfig, privateKey string) (*Client, error) {
	keyManager, err := NewKeyManager(privateKey)
	if err != nil {
		return nil, NewWalletError(ErrCodeInvalidPrivateKey, "failed to initialize key manager", err, "")
	}

	client := &Client{
		clients:      make(map[NetworkType]*ethclient.Client),
		configs:      make(map[NetworkType]NetworkConfig),
		keyManager:   keyManager,
		nonceManager: newNonceManager(),
		log:          log,
	}

	for _, config := range configs {
		ethClient, err := client.dialWithRetry(ctx, config)
		if err != nil {
			return nil, NewWalletError(ErrCodeRPCError, "failed to connect to network", err, config.Type)
		}
		client.clients[config.Type] = ethClient
		client.configs[config.Type] = config
	}

	return client, nil
}

// GetBalance retrieves the native token balance for an address on the specified network.
//
// Parameters:
//   - ctx: Context for the operation
//   - network: Target blockchain network
//   - address: Address to check balance for
//
// Returns:
//   - *big.Int: Balance in wei if successful
//   - error: Error if balance check fails
//
// Example:
//
//	balance, err := client.GetBalance(ctx, ETH, "0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Balance: %s ETH\n", balance.String())
func (c *Client) GetBalance(ctx context.Context, network NetworkType, address string) (*big.Int, error) {
	c.mu.RLock()
	client, ok := c.clients[network]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("network %s not configured", network)
	}

	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"network": network,
		"address": address,
		"balance": balance.String(),
	}).Debug("Retrieved balance")

	return balance, nil
}

// EstimateGas estimates the gas required for a transaction on the specified network.
// It applies network-specific gas price limits and multipliers to the estimate.
//
// Parameters:
//   - ctx: Context for the operation
//   - network: Target blockchain network
//   - to: Recipient address
//   - data: Transaction data payload
//   - value: Amount of native currency to send
//
// Returns:
//   - uint64: Estimated gas limit
//   - error: Error if estimation fails
//
// Example:
//
//	gasLimit, err := client.EstimateGas(ctx, ETH, toAddr, data, value)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) EstimateGas(ctx context.Context, network NetworkType, to common.Address, data []byte, value *big.Int) (uint64, error) {
	client, config, err := c.getClientAndConfig(network)
	if err != nil {
		return 0, err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get gas price: %w", err)
	}

	if config.MaxGasPrice != nil && gasPrice.Cmp(config.MaxGasPrice) > 0 {
		gasPrice = config.MaxGasPrice
	}

	msg := ethereum.CallMsg{
		From:     c.keyManager.GetAddress(),
		To:       &to,
		Data:     data,
		Value:    value,
		GasPrice: gasPrice,
	}

	estimatedGas, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}

	// Apply gas limit multiplier
	return uint64(float64(estimatedGas) * config.GasLimitMultiplier), nil
}

// SendTransaction sends a transaction on the specified network and waits for confirmation.
// It handles nonce management, gas estimation, and transaction signing.
//
// Parameters:
//   - ctx: Context for the operation
//   - network: Target blockchain network
//   - to: Recipient address
//   - data: Transaction data payload
//   - value: Amount of native currency to send
//
// Returns:
//   - *TransactionStatus: Status of sent transaction
//   - error: Error if transaction fails
//
// Example:
//
//	status, err := client.SendTransaction(ctx, ETH, toAddr, data, value)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Transaction confirmed in block %s\n", status.BlockNumber)
func (c *Client) SendTransaction(ctx context.Context, network NetworkType, to common.Address, data []byte, value *big.Int) (*TransactionStatus, error) {
	client, _, err := c.getClientAndConfig(network)
	if err != nil {
		return nil, err
	}

	nonce, err := c.nonceManager.GetNonce(ctx, c, network)
	if err != nil {
		return nil, err
	}
	defer c.nonceManager.ReleaseNonce(network, nonce)

	gasLimit, err := c.EstimateGas(ctx, network, to, data, value)
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.keyManager.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, NewWalletError(ErrCodeTransactionFailed, "failed to send transaction", err, network)
	}

	// Wait for receipt and return status
	return c.WaitForReceipt(ctx, network, signedTx.Hash())
}

// dialWithRetry attempts to connect to the network with retry mechanism.
// It will retry failed connection attempts based on the network configuration.
func (c *Client) dialWithRetry(ctx context.Context, config NetworkConfig) (*ethclient.Client, error) {
	var client *ethclient.Client
	var err error

	for i := 0; i <= config.MaxRetries; i++ {
		client, err = ethclient.DialContext(ctx, config.RPCURL)
		if err == nil {
			return client, nil
		}

		if i < config.MaxRetries {
			c.log.WithFields(logrus.Fields{
				"network": config.Type,
				"attempt": i + 1,
				"error":   err,
			}).Debug("Retrying network connection")

			time.Sleep(config.RetryDelay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", config.MaxRetries, err)
}

// getClientAndConfig returns the client and config for a network.
// It provides thread-safe access to the client and configuration maps.
func (c *Client) getClientAndConfig(network NetworkType) (*ethclient.Client, NetworkConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	client, ok := c.clients[network]
	if !ok {
		return nil, NetworkConfig{}, fmt.Errorf("network %s not configured", network)
	}

	config, ok := c.configs[network]
	if !ok {
		return nil, NetworkConfig{}, fmt.Errorf("network %s configuration not found", network)
	}

	return client, config, nil
}

// Close closes all network connections and cleans up resources.
// It should be called when the client is no longer needed.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for network, client := range c.clients {
		client.Close()
		c.log.WithField("network", network).Debug("Closed network connection")
	}
}
