// Package wallet provides blockchain wallet functionality for managing transactions,
// accounts, and interactions with various networks like Ethereum and BSC.
package wallet

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// Standard ERC20 ABI defines the minimal ABI for interacting with ERC20 tokens.
// It includes the balanceOf and transfer functions which are required for basic token operations.
const erc20ABI = `[
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function"
	}
]`

// TransferERC20 transfers ERC20 tokens from the wallet's address to the specified recipient.
// It handles contract interaction, transaction signing and submission.
//
// Parameters:
//   - ctx: Context for the operation
//   - network: Target blockchain network
//   - tokenAddress: Address of the ERC20 token contract
//   - to: Recipient address
//   - amount: Amount of tokens to transfer
//
// Returns:
//   - *common.Hash: Transaction hash if successful
//   - error: Error if the transfer fails
//
// Example:
//
//	hash, err := client.TransferERC20(ctx, NetworkEthereum, tokenAddr, recipientAddr, big.NewInt(1000))
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) TransferERC20(ctx context.Context, network NetworkType, tokenAddress, to common.Address, amount *big.Int) (*common.Hash, error) {
	client, _, err := c.getClientAndConfig(network)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Create a new bound contract
	contract := bind.NewBoundContract(tokenAddress, parsedABI, client, client, client)

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.keyManager.privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	// Create the transaction
	tx, err := contract.Transact(auth, "transfer", to, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer tokens: %w", err)
	}

	hash := tx.Hash()
	return &hash, nil
}

// GetERC20Balance retrieves the token balance for a specific address.
// It queries the ERC20 contract's balanceOf function to get the current balance.
//
// Parameters:
//   - ctx: Context for the operation
//   - network: Target blockchain network
//   - tokenAddress: Address of the ERC20 token contract
//   - account: Address to check the balance for
//
// Returns:
//   - *big.Int: Token balance if successful
//   - error: Error if the balance check fails
//
// Example:
//
//	balance, err := client.GetERC20Balance(ctx, NetworkEthereum, tokenAddr, userAddr)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Token balance: %s\n", balance.String())
func (c *Client) GetERC20Balance(ctx context.Context, network NetworkType, tokenAddress, account common.Address) (*big.Int, error) {
	client, _, err := c.getClientAndConfig(network)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, NewWalletError(ErrCodeInvalidABI, "failed to parse ABI", err, network)
	}

	// Create a new bound contract
	contract := bind.NewBoundContract(tokenAddress, parsedABI, client, client, client)

	var out []interface{}
	err = contract.Call(&bind.CallOpts{Context: ctx}, &out, "balanceOf", account)
	if err != nil {
		return nil, NewWalletError(ErrCodeContractError, "failed to get token balance", err, network)
	}

	if len(out) == 0 {
		return nil, NewWalletError(ErrCodeContractError, "no balance returned", nil, network)
	}

	balance, ok := out[0].(*big.Int)
	if !ok {
		return nil, NewWalletError(ErrCodeContractError, "failed to convert balance to *big.Int", nil, network)
	}

	return balance, nil
}

// TokenMetadata holds ERC20 token information including the contract address,
// token symbol, number of decimals, and token name. This information is typically
// retrieved from the token's smart contract and used for display and calculation purposes.
type TokenMetadata struct {
	// Address is the contract address of the token
	Address common.Address

	// Symbol is the token's symbol (e.g., "DAI", "USDC")
	Symbol string

	// Decimals specifies the number of decimal places the token uses
	// For example, 18 decimals means 1 token = 1e18 base units
	Decimals uint8

	// Name is the full name of the token (e.g., "Dai Stablecoin")
	Name string
}
