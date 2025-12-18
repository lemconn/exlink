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
- ✅ **Gate** - Spot & Perpetual Swaps

## API Support Matrix

| Exchange | Spot | Swap | Ticker | OHLCV | Balance | Orders | Trades | Positions | Leverage | Margin Mode |
|----------|------|------|--------|-------|---------|--------|--------|-----------|----------|-------------|
| Binance  | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| OKX      | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| Bybit    | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ✅          |
| Gate     | ✅   | ✅   | ✅     | ✅    | ✅      | ✅     | ✅     | ✅        | ✅       | ❌          |

**Legend:**
- ✅ Fully implemented
- ❌ Not supported by exchange API

**Notes:**
- **Orders**: Includes `CreateOrder`, `CancelOrder`, and `FetchOrder`.
- **Trades**: Includes `FetchTrades` (public trades) and `FetchMyTrades` (user trades).
- **Gate Margin Mode**: Gate does not support setting margin mode via API. It must be configured on the web interface.

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
    ex, err := exlink.NewExchange(exlink.ExchangeBinance)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get spot interface
    spot := ex.Spot()
    
    // Load markets
    if err := spot.LoadMarkets(ctx, false); err != nil {
        log.Fatal(err)
    }
    
    // Fetch ticker (using unified format BTC/USDT)
    ticker, err := spot.FetchTicker(ctx, "BTC/USDT")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("BTC/USDT Price: %s\n", ticker.Last)
}
```

### Using API Keys

```go
import (
    "github.com/lemconn/exlink"
    "github.com/lemconn/exlink/option"
)

// Create authenticated exchange instance
ex, err := exlink.NewExchange(
    exlink.ExchangeBinance,
    option.WithAPIKey("your-api-key"),
    option.WithSecretKey("your-secret-key"),
)
if err != nil {
    log.Fatal(err)
}

// Get spot interface
spot := ex.Spot()

// Load markets
if err := spot.LoadMarkets(ctx, false); err != nil {
    log.Fatal(err)
}

// Fetch balance
balances, err := spot.FetchBalance(ctx)
if err != nil {
    log.Fatal(err)
}

btcBalance := balances.GetBalance("BTC")
fmt.Printf("BTC Balance: %.8f\n", btcBalance.Free)
```

### Options

```go
import (
    "github.com/lemconn/exlink"
    "github.com/lemconn/exlink/option"
)

// Create exchange with options
ex, err := exlink.NewExchange(
    exlink.ExchangeBinance,
    option.WithAPIKey("your-api-key"),
    option.WithSecretKey("your-secret-key"),
    option.WithSandbox(true),                              // Enable sandbox mode
    option.WithProxy("http://proxy.example.com:8080"),    // Set proxy
)

// OKX requires password for authenticated requests
ex, err := exlink.NewExchange(
    exlink.ExchangeOKX,
    option.WithAPIKey("your-api-key"),
    option.WithSecretKey("your-secret-key"),
    option.WithPassword("your-password"),                  // Required for OKX
    option.WithSandbox(true),                              // Enable sandbox mode
    option.WithProxy("http://proxy.example.com:8080"),    // Set proxy
)
```

### Unified Symbol Format

All exchanges use the unified `BASE/QUOTE` format (e.g., `BTC/USDT`). The library automatically converts to each exchange's native format:

```go
// Get spot and perp interfaces
spot := ex.Spot()
perp := ex.Perp()

// Use unified format - library auto-converts
ticker, err := spot.FetchTicker(ctx, "BTC/USDT") 
// Binance: BTCUSDT, OKX: BTC-USDT, Gate: BTC_USDT, Bybit: BTCUSDT

// For perpetual contracts
ticker, err := perp.FetchTicker(ctx, "BTC/USDT:USDT")
// Binance: BTCUSDT, OKX: BTC-USDT-SWAP, Gate: BTC_USDT, Bybit: BTCUSDT
```

### Order Management

```go
import (
    "github.com/lemconn/exlink"
    "github.com/lemconn/exlink/model"
    "github.com/lemconn/exlink/option"
)

// Get spot interface
spot := ex.Spot()

// Create a limit order (with price option, it becomes a limit order)
order, err := spot.CreateOrder(ctx, "BTC/USDT", option.Buy,
    option.WithPrice("50000"),
    option.WithAmount("0.001"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Order created: %s\n", order.OrderId)

// Fetch order status
order, err = spot.FetchOrder(ctx, "BTC/USDT", order.ID, option.WithClientOrderID("clientOrderID"))
if err != nil {
    log.Fatal(err)
}

// Cancel order
err = spot.CancelOrder(ctx, "BTC/USDT", order.ID, option.WithClientOrderID("clientOrderID"))
if err != nil {
    log.Fatal(err)
}

```

### Contract Trading

```go
import (
    "github.com/lemconn/exlink"
    "github.com/lemconn/exlink/option"
)

// Get perp interface
perp := ex.Perp()

// Fetch positions
positions, err := perp.FetchPositions(ctx,
    option.WithSymbols("BTC/USDT:USDT"),
)
if err != nil {
    log.Fatal(err)
}

// Set leverage (contracts only)
err = perp.SetLeverage(ctx, "BTC/USDT:USDT", 10)
if err != nil {
    log.Fatal(err)
}

// Set margin mode (contracts only, not supported by Gate)
err = perp.SetMarginMode(ctx, "BTC/USDT:USDT", "isolated")
if err != nil {
    log.Fatal(err)
}
```

### More Examples

For more complex usage examples, see the [examples](./examples) directory.

## Adding a New Exchange

To add support for a new exchange:

1. Create a new package under the root directory (e.g., `myexchange/`)
2. Implement the `Exchange` interface from `exchange` package, which provides `Spot()` and `Perp()` methods
3. Implement `SpotExchange` and `PerpExchange` interfaces for spot and perpetual futures trading
4. Add the registration in `exlink.go`'s `init()` function

Example:

```go
package myexchange

import (
    "github.com/lemconn/exlink/exchange"
)

type MyExchange struct {
    spot *MyExchangeSpot
    perp *MyExchangePerp
}

func NewMyExchange(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
    // ... initialization logic
    return &MyExchange{
        spot: NewMyExchangeSpot(...),
        perp: NewMyExchangePerp(...),
    }, nil
}

func (e *MyExchange) Spot() exchange.SpotExchange {
    return e.spot
}

func (e *MyExchange) Perp() exchange.PerpExchange {
    return e.perp
}

func (e *MyExchange) Name() string {
    return "myexchange"
}
```

Then add the registration in `exlink.go`:

```go
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
- `ExchangeGate`: Gate exchange

### Market Types

- `model.MarketTypeSpot`: Spot market
- `model.MarketTypeSwap`: Perpetual swap market
- `model.MarketTypeFuture`: Perpetual swap market (synonym for MarketTypeSwap)

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
