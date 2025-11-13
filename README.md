# ExLink - Unified Cryptocurrency Exchange API Library for Go

[![ci](https://github.com/lemconn/exlink/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/lemconn/exlink/actions/workflows/test.yml)
[![GoDoc](https://godoc.org/github.com/lemconn/exlink?status.svg)](https://godoc.org/github.com/lemconn/exlink) 
[![Go Report Card](https://goreportcard.com/badge/github.com/lemconn/exlink)](https://goreportcard.com/report/github.com/lemconn/exlink)

ExLink is a Go library similar to Python's ccxt, providing a standardized interface to access multiple cryptocurrency exchange APIs.

## Features

- **Unified Interface**: Standardized API interface supporting multiple exchanges
- **Spot & Derivatives**: Support for spot trading and perpetual contracts
- **Type Safe**: Complete type definitions with compile-time checking
- **Easy to Extend**: Simple interface implementation for adding new exchanges
- **Modular Design**: Clear code structure, easy to maintain

## Supported Exchanges

- ✅ **Binance** - Spot & Perpetual Swaps
- ✅ **OKX** - Spot & Perpetual Swaps
- ✅ **Bybit** - Spot & Perpetual Swaps
- ✅ **Gate.io** - Spot & Perpetual Swaps

## API Support Matrix

| Exchange | Spot | Swap | Ticker | OHLCV | Balance | Orders | Trades | Positions | Leverage | Margin Mode |
|----------|------|------|--------|-------|---------|--------|--------|-----------|----------|-------------|
| Binance  | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| OKX      | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| Bybit    | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| Gate.io  | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ❌          |

**Legend:**
- ✅ Fully implemented
- ❌ Not supported by exchange API

**Notes:**
- **Orders**: Includes `CreateOrder`, `CancelOrder`, `FetchOrder`, and `FetchOpenOrders`. `FetchOrders` (all orders) is not directly supported by Gate.io and Bybit APIs.
- **Trades**: Includes `FetchTrades` (public trades) and `FetchMyTrades` (user trades).
- **Gate.io Margin Mode**: Gate.io does not support setting margin mode via API. It must be configured on the web interface.

## Quick Start

### Installation

```bash
go get github.com/lemconn/exlink
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lemconn/exlink"
)

func main() {
    ctx := context.Background()
    
    // Create exchange instance (no API keys needed for public data)
    exchange, err := exlink.NewExchange(exlink.ExchangeBinance)
    if err != nil {
        log.Fatal(err)
    }
    
    // Fetch ticker (using unified format BTC/USDT)
    ticker, err := exchange.FetchTicker(ctx, "BTC/USDT")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("BTC/USDT Price: %.2f\n", ticker.Last)
    fmt.Printf("24h Change: %.2f%%\n", ticker.ChangePercent)
}
```

### Using API Keys

```go
// Create authenticated exchange instance
exchange, err := exlink.NewExchange(
    exlink.ExchangeBinance,
    exlink.WithAPIKey("your-api-key"),
    exlink.WithSecretKey("your-secret-key"),
)
if err != nil {
    log.Fatal(err)
}

// Fetch balance
balances, err := exchange.FetchBalance(ctx)
if err != nil {
    log.Fatal(err)
}

btcBalance := balances.GetBalance("BTC")
fmt.Printf("BTC Balance: %.8f\n", btcBalance.Free)
```

### Options

```go
// Create exchange with options
exchange, err := exlink.NewExchange(
    exlink.ExchangeBinance,
    exlink.WithAPIKey("your-api-key"),
    exlink.WithSecretKey("your-secret-key"),
    exlink.WithSandbox(true),                              // Enable sandbox mode
    exlink.WithProxy("http://proxy.example.com:8080"),    // Set proxy
    exlink.WithFetchMarkets(exlink.MarketSpot, exlink.MarketSwap), // Load specific market types
)

// OKX requires passphrase for authenticated requests
exchange, err := exlink.NewExchange(
    exlink.ExchangeOKX,
    exlink.WithAPIKey("your-api-key"),
    exlink.WithSecretKey("your-secret-key"),
    exlink.WithPassphrase("your-passphrase"),             // Required for OKX
    exlink.WithSandbox(true),                              // Enable sandbox mode
    exlink.WithProxy("http://proxy.example.com:8080"),    // Set proxy
)
```

### Unified Symbol Format

All exchanges use the unified `BASE/QUOTE` format (e.g., `BTC/USDT`). The library automatically converts to each exchange's native format:

```go
// Use unified format - library auto-converts
ticker, err := exchange.FetchTicker(ctx, "BTC/USDT") 
// Binance: BTCUSDT, OKX: BTC-USDT, Gate: BTC_USDT, Bybit: BTCUSDT

// For perpetual contracts
ticker, err := exchange.FetchTicker(ctx, "BTC/USDT:USDT")
// Binance: BTCUSDT, OKX: BTC-USDT-SWAP, Gate: BTC_USDT, Bybit: BTCUSDT
```

### Order Management

```go
import (
    "github.com/lemconn/exlink"
    "github.com/lemconn/exlink/types"
)

// Create a limit order
order, err := exchange.CreateOrder(ctx, "BTC/USDT", types.OrderSideBuy, types.OrderTypeLimit, 0.001, 50000, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Order created: %s\n", order.ID)

// Fetch order status
order, err = exchange.FetchOrder(ctx, order.ID, "BTC/USDT")
if err != nil {
    log.Fatal(err)
}

// Cancel order
err = exchange.CancelOrder(ctx, order.ID, "BTC/USDT")
if err != nil {
    log.Fatal(err)
}

// Fetch open orders
openOrders, err := exchange.FetchOpenOrders(ctx, "BTC/USDT")
if err != nil {
    log.Fatal(err)
}
```

### Trading History

```go
import (
    "time"
    "github.com/lemconn/exlink"
)

// Fetch public trades
trades, err := exchange.FetchTrades(ctx, "BTC/USDT", time.Time{}, 100)
if err != nil {
    log.Fatal(err)
}

// Fetch my trades (requires authentication)
myTrades, err := exchange.FetchMyTrades(ctx, "BTC/USDT", time.Time{}, 100)
if err != nil {
    log.Fatal(err)
}
```

### Contract Trading

```go
import "github.com/lemconn/exlink"

// Fetch positions
positions, err := exchange.FetchPositions(ctx, "BTC/USDT:USDT")
if err != nil {
    log.Fatal(err)
}

// Set leverage (contracts only)
err = exchange.SetLeverage(ctx, "BTC/USDT:USDT", 10)
if err != nil {
    log.Fatal(err)
}

// Set margin mode (contracts only, not supported by Gate.io)
err = exchange.SetMarginMode(ctx, "BTC/USDT:USDT", "isolated")
if err != nil {
    log.Fatal(err)
}
```

### More Examples

For more complex usage examples, see the [examples](./examples) directory.

## Adding a New Exchange

To add support for a new exchange:

1. Create a new package under `exchanges/` directory
2. Implement the `Exchange` interface from `base` package
3. Add the registration in `registry.go`'s `init()` function

Example:

```go
package myexchange

import (
    "github.com/lemconn/exlink/base"
    "github.com/lemconn/exlink/common"
    "github.com/lemconn/exlink/types"
)

type MyExchange struct {
    *base.BaseExchange
    // ... other fields
}

func NewMyExchange(apiKey, secretKey string, options map[string]interface{}) (base.Exchange, error) {
    // ... initialization logic
    return &MyExchange{
        BaseExchange: base.NewBaseExchange("myexchange"),
        // ...
    }, nil
}
```

Then add the registration in `registry.go`:

```go
// 首先定义常量
const ExchangeMyExchange = "myexchange"

func init() {
    Register(ExchangeBinance, binance.NewBinance)
    Register(ExchangeBybit, bybit.NewBybit)
    Register(ExchangeOKX, okx.NewOKX)
    Register(ExchangeGate, gate.NewGate)
    Register(ExchangeMyExchange, myexchange.NewMyExchange) // Add your exchange here
}
```

## Core Concepts

### Exchange Names

- `ExchangeBinance`: Binance exchange
- `ExchangeBybit`: Bybit exchange
- `ExchangeOKX`: OKX exchange
- `ExchangeGate`: Gate.io exchange

### Market Types

- `MarketSpot`: Spot market
- `MarketSwap`: Perpetual swap market
- `MarketFuture`: Perpetual swap market (synonym for MarketSwap)

### Order Types

- `OrderTypeMarket`: Market order
- `OrderTypeLimit`: Limit order

### Order Sides

- `OrderSideBuy`: Buy
- `OrderSideSell`: Sell

### Order Status

- `OrderStatusNew`: New
- `OrderStatusOpen`: Open
- `OrderStatusFilled`: Filled
- `OrderStatusCanceled`: Canceled
- And more...
