package main

import (
	"context"
	"fmt"
	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/base"
	"github.com/lemconn/exlink/types"
	"log"
	"os"
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
		log.Println("Warning: OKX_API_KEY, OKX_SECRET_KEY and OKX_PASSPHRASE environment variables are not set")
		log.Println("Example code will demonstrate usage, but actual calls will fail")
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
		log.Fatalf("Failed to create exchange instance: %v", err)
	}

	// Load market information
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		log.Fatalf("Failed to load market information: %v", err)
	}

	// Contract trading pair (perpetual contract)
	contractSymbol := "BTC/USDT:USDT"
	// Spot trading pair
	spotSymbol := "BTC/USDT"

	// Get current price
	ticker, err := exchange.FetchTicker(ctx, contractSymbol)
	if err != nil {
		log.Fatalf("Failed to fetch ticker: %v", err)
	}
	fmt.Printf("Current price: %.2f\n", ticker.Last)

	amount := 0.001 // Order quantity (adjust according to actual situation)

	// ========== Contract Trading Examples ==========
	fmt.Println("=== Contract Trading Examples ===")

	// 1. Open long position (buy to open long)
	fmt.Println("\n1. Open long position")
	order, err := openLongPosition(ctx, exchange, contractSymbol, amount)
	if err != nil {
		log.Printf("Failed to open long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 2. Close long position (sell to close long)
	fmt.Println("\n2. Close long position")
	order, err = closeLongPosition(ctx, exchange, contractSymbol, amount)
	if err != nil {
		log.Printf("Failed to close long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 3. Open short position (sell to open short)
	fmt.Println("\n3. Open short position")
	order, err = openShortPosition(ctx, exchange, contractSymbol, amount)
	if err != nil {
		log.Printf("Failed to open short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 4. Close short position (buy to close short)
	fmt.Println("\n4. Close short position")
	order, err = closeShortPosition(ctx, exchange, contractSymbol, amount)
	if err != nil {
		log.Printf("Failed to close short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// ========== Spot Trading Examples ==========
	fmt.Println("\n=== Spot Trading Examples ===")

	// 5. Buy spot
	fmt.Println("\n5. Buy spot")
	order, err = buySpot(ctx, exchange, spotSymbol, amount)
	if err != nil {
		log.Printf("Failed to buy spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 6. Sell spot
	fmt.Println("\n6. Sell spot")
	order, err = sellSpot(ctx, exchange, spotSymbol, amount)
	if err != nil {
		log.Printf("Failed to sell spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// Limit order example
	fmt.Println("\n=== Limit Order Example ===")
	limitPrice := ticker.Bid * 0.99 // Buy limit order, 1% below current price
	fmt.Printf("\nLimit buy: Price=%.2f, Quantity=%f\n", limitPrice, amount)
	order, err = exchange.CreateOrder(ctx, spotSymbol, types.OrderSideBuy, types.OrderTypeLimit, amount, limitPrice, nil)
	if err != nil {
		log.Printf("Failed to create limit buy order: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Price=%.2f, Amount=%f\n",
			order.ID, order.Price, order.Amount)
	}
}

// openLongPosition opens a long position (buy to open long)
func openLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	params := map[string]interface{}{
		"posSide": "long", // OKX uses posSide to specify position direction
	}
	return exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
}

// closeLongPosition closes a long position (sell to close long)
func closeLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	params := map[string]interface{}{
		"posSide": "long", // To close long position, need to specify posSide="long"
	}
	return exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
}

// openShortPosition opens a short position (sell to open short)
func openShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	params := map[string]interface{}{
		"posSide": "short", // OKX uses posSide="short" to open short
	}
	return exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
}

// closeShortPosition closes a short position (buy to close short)
func closeShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	params := map[string]interface{}{
		"posSide": "short", // To close short position, need to specify posSide="short"
	}
	return exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
}

// buySpot buys spot assets
func buySpot(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	return exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, nil)
}

// sellSpot sells spot assets
func sellSpot(ctx context.Context, exchange base.Exchange, symbol string, amount float64) (*types.Order, error) {
	return exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, nil)
}
