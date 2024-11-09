# Masa Agent-Go Wallet Package

A sophisticated Ethereum wallet implementation designed specifically for AI-powered Twitter agents. This package provides a secure, concurrent-safe interface for blockchain interactions, enabling agents to perform on-chain actions as part of their cognitive processing and response generation.

## Features

- 🌐 **Multi-Network Integration**
  - Seamless integration with Ethereum, Base, and BSC networks
  - Built for AI agent interactions with blockchain
  - Extensible for future network support

- 💰 **Agent Operations**
  - Secure transaction handling for agent responses
  - Token management for agent incentives
  - Balance monitoring for operational costs
  - Automated gas optimization for agent transactions

- 🔒 **Agent Security**
  - Protected key management for autonomous operations
  - Transaction signing with safety checks
  - Address validation for user interactions
  - Rate limiting and safety controls

- ⚡ **Agent Performance**
  - Concurrent transaction handling for multiple agent instances
  - Smart nonce management for rapid responses
  - Connection pooling for network efficiency
  - Configurable retry mechanisms for reliability

- 🛠 **AI Integration Features**
  - Transaction hooks for cognitive processing
  - Event monitoring for agent awareness
  - Structured logging for agent memory
  - Configurable gas strategies for cost optimization

## Installation

```bash
github.com/lisanmuaddib/agent-go/pkg/wallet
```

## Quick Start

### Environment Setup
First, create a `.env` file based on the provided `.env.example`:

The wallet package uses the following environment variables:

```env
# Network RPCs
ETH_RPC_URL=        # Ethereum RPC endpoint
BASE_RPC_URL=       # Base network RPC endpoint
BSC_RPC_URL=        # BSC RPC endpoint

# Wallet Configuration
WALLET_PRIVATE_KEY= # Private key for transaction signing

# Logging
LOG_LEVEL=          # Logging level (DEBUG, INFO, WARN, ERROR)
```

Make sure to set these variables in your `.env` file or environment before using the wallet package.

### Initialize Client

```go
import (
    "context"
    "os"
    "github.com/lisanmuaddib/agent-go/pkg/wallet"
    "github.com/sirupsen/logrus"
    "github.com/joho/godotenv"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Warn("Error loading .env file")
    }

    // Initialize logger with environment-based level
    log := logrus.New()
    if level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
        log.SetLevel(level)
    } else {
        log.SetLevel(logrus.InfoLevel)
    }

    // Create network configurations from environment
    configs := []wallet.NetworkConfig{
        {
            Type:               wallet.ETH,
            RPCURL:            os.Getenv("ETH_RPC_URL"),
            ChainID:           1,
            MaxRetries:        3,
            RetryDelay:        time.Second,
            GasLimitMultiplier: 1.2,
            MaxGasPrice:        big.NewInt(300000000000), // 300 gwei
        },
        {
            Type:               wallet.BASE,
            RPCURL:            os.Getenv("BASE_RPC_URL"),
            ChainID:           8453,
            MaxRetries:        3,
            RetryDelay:        time.Second,
            GasLimitMultiplier: 1.2,
            MaxGasPrice:        big.NewInt(100000000000), // 100 gwei
        },
        {
            Type:               wallet.BSC,
            RPCURL:            os.Getenv("BSC_RPC_URL"),
            ChainID:           56,
            MaxRetries:        3,
            RetryDelay:        time.Second,
            GasLimitMultiplier: 1.2,
            MaxGasPrice:        big.NewInt(5000000000), // 5 gwei
        },
    }

    // Initialize wallet client
    client, err := wallet.NewClient(
        context.Background(),
        log,
        configs,
        os.Getenv("WALLET_PRIVATE_KEY"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
}
```

### Network Configuration

You can either use the default configurations or create custom ones:

```go
// Using defaults (includes preset RPCs and gas strategies)
configs := wallet.DefaultNetworkConfigs()

// Or customize per network
customConfig := wallet.NetworkConfig{
    Type:               wallet.ETH,
    RPCURL:            os.Getenv("ETH_RPC_URL"),
    ChainID:           1,
    MaxRetries:        3,
    RetryDelay:        time.Second,
    GasLimitMultiplier: 1.2,
    MaxGasPrice:       big.NewInt(300000000000), // 300 gwei
}
```

### Send Native Currency

```go
func sendETH(client *wallet.Client, to string, amount *big.Int) {
    ctx := context.Background()
    
    // Validate address
    if err := client.ValidateAddress(wallet.ETH, to); err != nil {
        log.Fatal(err)
    }

    // Send transaction
    status, err := client.SendTransaction(
        ctx,
        wallet.ETH,
        common.HexToAddress(to),
        nil,
        amount,
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Transaction sent: %s", status.Hash.Hex())
}
```

### Transfer ERC20 Tokens

```go
func transferTokens(client *wallet.Client, tokenAddr, to string, amount *big.Int) {
    ctx := context.Background()
    
    hash, err := client.TransferERC20(
        ctx,
        wallet.ETH,
        common.HexToAddress(tokenAddr),
        common.HexToAddress(to),
        amount,
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Token transfer initiated: %s", hash.Hex())
}
```

## Configuration

### Gas Strategy

```go
strategy := &wallet.GasStrategy{
    MaxGasPrice:     big.NewInt(100000000000), // 100 gwei
    PriorityFee:     big.NewInt(1500000000),   // 1.5 gwei
    RetryOnHighGas:  true,
    WaitForLowerGas: true,
}
```

## Error Handling

The package provides detailed error types for better error handling:

```go
if err != nil {
    if wallet.IsWalletError(err, wallet.ErrCodeInsufficientFunds) {
        // Handle insufficient funds
    } else if wallet.IsWalletError(err, wallet.ErrCodeGasPrice) {
        // Handle high gas price
    }
}
```

## Advanced Usage

### Custom Transaction Options

```go
opts := &wallet.TransactionOptions{
    GasStrategy:  customGasStrategy,
    WaitReceipt: true,
    MaxRetries:  5,
}

status, err := client.SendTransactionWithOptions(
    ctx,
    wallet.ETH,
    to,
    data,
    value,
    opts,
)
```

### Transaction Status Tracking

```go
status, err := client.WaitForReceipt(ctx, wallet.ETH, txHash)
if err != nil {
    log.Fatal(err)
}

switch status.State {
case wallet.TxStateConfirmed:
    log.Printf("Transaction confirmed with %d confirmations", status.Confirmations)
case wallet.TxStateFailed:
    log.Printf("Transaction failed: %v", status.Error)
}
```

## Best Practices

1. Always use context for timeout management
2. Implement proper error handling
3. Close the client when done
4. Use appropriate gas strategies for different networks
5. Validate addresses before transactions
6. Monitor transaction status for confirmation

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

