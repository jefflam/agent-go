// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"fmt"
)

// Error codes for various wallet operations
const (
	// ErrCodeInvalidNetwork indicates the specified network is not supported
	ErrCodeInvalidNetwork = "INVALID_NETWORK"
	// ErrCodeInvalidAddress indicates an invalid blockchain address format
	ErrCodeInvalidAddress = "INVALID_ADDRESS"
	// ErrCodeInvalidPrivateKey indicates an invalid or malformed private key
	ErrCodeInvalidPrivateKey = "INVALID_PRIVATE_KEY"
	// ErrCodeTransactionFailed indicates a transaction failed to execute
	ErrCodeTransactionFailed = "TRANSACTION_FAILED"
	// ErrCodeGasEstimationFailed indicates gas estimation failed
	ErrCodeGasEstimationFailed = "GAS_ESTIMATION_FAILED"
	// ErrCodeNonceTooLow indicates transaction nonce is too low
	ErrCodeNonceTooLow = "NONCE_TOO_LOW"
	// ErrCodeInsufficientFunds indicates insufficient balance for transaction
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	// ErrCodeRPCError indicates an RPC connection or call failed
	ErrCodeRPCError = "RPC_ERROR"
	// ErrCodeTimeout indicates operation timed out
	ErrCodeTimeout = "TIMEOUT"
	// ErrCodeReceiptNotFound indicates transaction receipt not found
	ErrCodeReceiptNotFound = "RECEIPT_NOT_FOUND"
	// ErrCodeInvalidABI indicates invalid or malformed contract ABI
	ErrCodeInvalidABI = "INVALID_ABI"
	// ErrCodeContractError indicates contract interaction failed
	ErrCodeContractError = "CONTRACT_ERROR"
	// ErrCodePendingTransaction indicates transaction is still pending
	ErrCodePendingTransaction = "PENDING_TRANSACTION"
	// ErrCodeChainMismatch indicates chain ID mismatch
	ErrCodeChainMismatch = "CHAIN_MISMATCH"
	// ErrCodeGasPrice indicates gas price exceeds maximum allowed
	ErrCodeGasPrice = "GAS_PRICE_TOO_HIGH"
)

// WalletError represents a wallet-specific error with additional context
// about the error type, message, underlying error and network.
type WalletError struct {
	Code    string      // Error code identifying the type of error
	Message string      // Human readable error message
	Err     error       // Underlying error if any
	Network NetworkType // Network where the error occurred
}

// Error implements the error interface for WalletError.
// It formats the error message including the code, message, network (if present)
// and underlying error.
func (e *WalletError) Error() string {
	if e.Network != "" {
		return fmt.Sprintf("[%s] %s on network %s: %v", e.Code, e.Message, e.Network, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
}

// Unwrap returns the underlying error.
// This implements the errors.Unwrap interface for error wrapping.
func (e *WalletError) Unwrap() error {
	return e.Err
}

// NewWalletError creates a new WalletError with the given parameters.
//
// Parameters:
//   - code: Error code identifying the type of error
//   - message: Human readable error message
//   - err: Underlying error if any
//   - network: Network where the error occurred
//
// Returns:
//   - *WalletError: A new wallet error instance
func NewWalletError(code string, message string, err error, network NetworkType) *WalletError {
	return &WalletError{
		Code:    code,
		Message: message,
		Err:     err,
		Network: network,
	}
}

// IsWalletError checks if an error is a WalletError and matches the given code.
//
// Parameters:
//   - err: Error to check
//   - code: Error code to match against
//
// Returns:
//   - bool: true if err is a WalletError with matching code, false otherwise
func IsWalletError(err error, code string) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*WalletError); ok {
		return e.Code == code
	}
	return false
}
