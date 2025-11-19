package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/base"
	"github.com/lemconn/exlink/types"
)

func main() {
	ctx := context.Background()

	// Get API keys from environment variables (please set them in actual use)
	apiKey := os.Getenv("OKX_API_KEY")
	secretKey := os.Getenv("OKX_SECRET_KEY")
	passphrase := os.Getenv("OKX_PASSPHRASE")

	// Get proxy URL from environment variable
	proxyURL := os.Getenv("PROXY_URL")

	if apiKey == "" || secretKey == "" || passphrase == "" {
		fmt.Println("Warning: OKX_API_KEY, OKX_SECRET_KEY and OKX_PASSPHRASE environment variables are not set")
		fmt.Println("Example code will demonstrate usage, but actual calls will fail")
		return
	}

	// Create OKX exchange instance
	opts := []exlink.Option{
		exlink.WithAPIKey(apiKey),
		exlink.WithSecretKey(secretKey),
		exlink.WithPassphrase(passphrase),
		exlink.WithProxy(proxyURL),
		exlink.WithSandbox(true), // Sandbox mode
	}

	exchange, err := exlink.NewExchange(exlink.ExchangeOKX, opts...)
	if err != nil {
		fmt.Printf("Failed to create exchange instance: %v", err)
		return
	}

	// Load market information
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		fmt.Printf("Failed to load market information: %v", err)
		return
	}

	// Contract trading pair (perpetual contract)
	contractSymbol := "BTC/USDT:USDT"
	// Spot trading pair
	spotSymbol := "BTC/USDT"

	// Get current price
	ticker, err := exchange.FetchTicker(ctx, contractSymbol)
	if err != nil {
		fmt.Printf("Failed to fetch ticker: %v", err)
		return
	}
	fmt.Printf("Current price: %s\n", ticker.Last)

	amount := 0.001 // Order quantity (adjust according to actual situation)

	// ========== Contract Trading Examples ==========
	fmt.Println("=== Contract Trading Examples ===")

	// 1. Open long position (buy to open long)
	fmt.Println("\n1. Open long position")
	openLongPosition(ctx, exchange, contractSymbol, amount)

	// 2. Close long position (sell to close long)
	fmt.Println("\n2. Close long position")
	closeLongPosition(ctx, exchange, contractSymbol, amount)

	// 3. Open short position (sell to open short)
	fmt.Println("\n3. Open short position")
	openShortPosition(ctx, exchange, contractSymbol, amount)

	// 4. Close short position (buy to close short)
	fmt.Println("\n4. Close short position")
	closeShortPosition(ctx, exchange, contractSymbol, amount)

	// ========== Spot Trading Examples ==========
	fmt.Println("\n=== Spot Trading Examples ===")

	// 5. Buy spot
	fmt.Println("\n5. Buy spot")
	buySpot(ctx, exchange, spotSymbol, amount)

	// 6. Sell spot
	fmt.Println("\n6. Sell spot")
	sellSpot(ctx, exchange, spotSymbol, amount)
}

// openLongPosition opens a long position (buy to open long)
func openLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	params := map[string]interface{}{
		"posSide": "long", // OKX uses posSide to specify position direction
	}
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		fmt.Printf("Failed to open long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeLongPosition closes a long position (sell to close long)
func closeLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	params := map[string]interface{}{
		"posSide": "long", // To close long position, need to specify posSide="long"
	}
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		fmt.Printf("Failed to close long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// openShortPosition opens a short position (sell to open short)
func openShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	params := map[string]interface{}{
		"posSide": "short", // OKX uses posSide="short" to open short
	}
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		fmt.Printf("Failed to open short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeShortPosition closes a short position (buy to close short)
func closeShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	params := map[string]interface{}{
		"posSide": "short", // To close short position, need to specify posSide="short"
	}
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		fmt.Printf("Failed to close short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// buySpot buys spot assets
func buySpot(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		fmt.Printf("Failed to buy spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// sellSpot sells spot assets
func sellSpot(ctx context.Context, exchange base.Exchange, symbol string, amount float64) {
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		fmt.Printf("Failed to sell spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}
