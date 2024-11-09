// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TransactionStatus represents the status of a transaction on the blockchain.
// It tracks important transaction details like hash, confirmation status,
// gas usage, and any errors that occurred during processing.
type TransactionStatus struct {
	// Hash is the unique transaction identifier
	Hash common.Hash

	// Status indicates transaction success (1) or failure (0)
	Status uint64

	// BlockNumber is the block height where transaction was mined
	BlockNumber *big.Int

	// GasUsed is the actual amount of gas consumed
	GasUsed uint64

	// EffectiveGasPrice is the actual gas price paid
	EffectiveGasPrice *big.Int

	// Confirmations is the number of block confirmations
	Confirmations uint64

	// State tracks the current transaction state
	State TransactionState

	// Timestamp when the status was last updated
	Timestamp time.Time

	// Error captures any transaction-specific errors
	Error error
}

// TransactionState represents the possible states of a transaction
type TransactionState int

const (
	// TxStatePending indicates transaction is waiting to be mined
	TxStatePending TransactionState = iota

	// TxStateConfirmed indicates transaction was successfully mined
	TxStateConfirmed

	// TxStateFailed indicates transaction failed during execution
	TxStateFailed

	// TxStateDropped indicates transaction was dropped from mempool
	TxStateDropped
)

const (
	// defaultReceiptTimeout is how long to wait for a receipt
	defaultReceiptTimeout = 5 * time.Minute

	// defaultPollInterval is how often to check for receipt
	defaultPollInterval = 5 * time.Second

	// minConfirmations is minimum blocks needed to consider tx confirmed
	minConfirmations = 1
)

// WaitForReceipt waits for a transaction receipt and returns the transaction status.
// It polls the network at regular intervals until the transaction is mined and
// has reached the minimum number of confirmations.
//
// Parameters:
//   - ctx: Context for cancellation
//   - network: Target blockchain network
//   - hash: Transaction hash to wait for
//
// Returns:
//   - *TransactionStatus: Final transaction status
//   - error: Error if receipt cannot be retrieved
//
// Example:
//
//	status, err := client.WaitForReceipt(ctx, NetworkEthereum, txHash)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Transaction confirmed in block %s\n", status.BlockNumber)
func (c *Client) WaitForReceipt(ctx context.Context, network NetworkType, hash common.Hash) (*TransactionStatus, error) {
	client, _, err := c.getClientAndConfig(network)
	if err != nil {
		return nil, NewWalletError(ErrCodeInvalidNetwork, "failed to get network client", err, network)
	}

	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	timeout := time.After(defaultReceiptTimeout)

	for {
		select {
		case <-ctx.Done():
			return nil, NewWalletError(ErrCodeTimeout, "context cancelled while waiting for receipt", ctx.Err(), network)
		case <-timeout:
			return nil, NewWalletError(ErrCodeTimeout, "timeout waiting for receipt", nil, network)
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(ctx, hash)
			if err != nil {
				continue // Receipt not found yet
			}

			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				return nil, NewWalletError(ErrCodeRPCError, "failed to get current block number", err, network)
			}

			confirmations := currentBlock - receipt.BlockNumber.Uint64()
			if confirmations < minConfirmations {
				continue // Wait for minimum confirmations
			}

			return &TransactionStatus{
				Hash:              hash,
				Status:            receipt.Status,
				BlockNumber:       receipt.BlockNumber,
				GasUsed:           receipt.GasUsed,
				EffectiveGasPrice: receipt.EffectiveGasPrice,
				Confirmations:     confirmations,
				State:             TxStateConfirmed,
				Timestamp:         time.Now(),
			}, nil
		}
	}
}

// NeedsResubmission checks if a transaction needs to be resubmitted based on its current state.
// Returns true if the transaction is dropped or has been pending too long.
//
// Returns:
//   - bool: True if transaction should be resubmitted
func (ts *TransactionStatus) NeedsResubmission() bool {
	return ts.State == TxStateDropped ||
		(ts.State == TxStatePending && time.Since(ts.Timestamp) > defaultReceiptTimeout)
}

// TransactionOptions represents configurable options for sending transactions.
// It allows customization of gas pricing, nonce management, receipt waiting,
// and retry behavior.
type TransactionOptions struct {
	// GasStrategy for managing gas prices
	GasStrategy *GasStrategy

	// Nonce allows manual nonce specification
	Nonce *uint64

	// WaitReceipt determines if we wait for mining
	WaitReceipt bool

	// MaxRetries is number of send attempts
	MaxRetries int
}

// DefaultTransactionOptions returns a TransactionOptions instance with default values.
// The defaults are configured for typical transaction sending scenarios.
//
// Returns:
//   - *TransactionOptions: Default options configuration
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		GasStrategy: DefaultGasStrategy(),
		WaitReceipt: true,
		MaxRetries:  3,
	}
}

// SendTransactionWithOptions sends a transaction with custom options for gas pricing,
// nonce management, and receipt waiting. It handles transaction creation, signing,
// and submission to the network.
//
// Parameters:
//   - ctx: Context for cancellation
//   - network: Target blockchain network
//   - to: Recipient address
//   - data: Transaction data payload
//   - value: Amount of native currency to send
//   - opts: Custom transaction options
//
// Returns:
//   - *TransactionStatus: Status of sent transaction
//   - error: Error if transaction fails
//
// Example:
//
//	opts := DefaultTransactionOptions()
//	opts.WaitReceipt = false
//	status, err := client.SendTransactionWithOptions(ctx, NetworkEthereum, toAddr, data, value, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SendTransactionWithOptions(
	ctx context.Context,
	network NetworkType,
	to common.Address,
	data []byte,
	value *big.Int,
	opts *TransactionOptions,
) (*TransactionStatus, error) {
	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	client, config, err := c.getClientAndConfig(network)
	if err != nil {
		return nil, err
	}

	// Use config for gas price checks
	if opts.GasStrategy.MaxGasPrice == nil {
		opts.GasStrategy.MaxGasPrice = config.MaxGasPrice
	}

	// Get or use provided nonce
	var nonce uint64
	if opts.Nonce != nil {
		nonce = *opts.Nonce
	} else {
		nonce, err = c.nonceManager.GetNonce(ctx, c, network)
		if err != nil {
			return nil, err
		}
		defer c.nonceManager.ReleaseNonce(network, nonce)
	}

	// Estimate gas with strategy
	gasLimit, err := c.EstimateGas(ctx, network, to, data, value)
	if err != nil {
		return nil, err
	}

	// Apply gas strategy
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	if opts.GasStrategy.MaxGasPrice != nil && gasPrice.Cmp(opts.GasStrategy.MaxGasPrice) > 0 {
		if opts.GasStrategy.WaitForLowerGas {
			return nil, NewWalletError(ErrCodeGasPrice, "gas price too high", nil, network)
		}
		gasPrice = opts.GasStrategy.MaxGasPrice
	}

	// Create and sign transaction
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.keyManager.privateKey)
	if err != nil {
		return nil, err
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, NewWalletError(ErrCodeTransactionFailed, "failed to send transaction", err, network)
	}

	// Return immediately if not waiting for receipt
	if !opts.WaitReceipt {
		return &TransactionStatus{
			Hash:      signedTx.Hash(),
			State:     TxStatePending,
			Timestamp: time.Now(),
		}, nil
	}

	// Wait for receipt
	return c.WaitForReceipt(ctx, network, signedTx.Hash())
}
