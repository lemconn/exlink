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
	apiKey := os.Getenv("GATE_API_KEY")
	secretKey := os.Getenv("GATE_SECRET_KEY")

	// Get proxy URL from environment variable
	proxyURL := os.Getenv("PROXY_URL")

	if apiKey == "" || secretKey == "" {
		fmt.Println("Warning: GATE_API_KEY and GATE_SECRET_KEY environment variables are not set")
		fmt.Println("Example code will demonstrate usage, but actual calls will fail")
		return
	}

	// Create Gate exchange instance
	opts := []exlink.Option{
		exlink.WithAPIKey(apiKey),
		exlink.WithSecretKey(secretKey),
		exlink.WithProxy(proxyURL),
		exlink.WithSandbox(true), // Sandbox mode
		exlink.WithFetchMarkets(exlink.MarketFuture, exlink.MarketSpot),
	}

	exchange, err := exlink.NewExchange(exlink.ExchangeGate, opts...)
	if err != nil {
		fmt.Printf("Failed to create exchange instance: %v", err)
		return
	}

	// Load market information
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		fmt.Printf("Failed to load market information: %v", err)
		return
	}

	// Future trading pair (perpetual future)
	futureSymbol := "DOGE/USDT:USDT"
	// Get current future price
	ticker, err := exchange.FetchTicker(ctx, futureSymbol)
	if err != nil {
		fmt.Printf("Failed to fetch future ticker: %v", err)
		return
	}
	fmt.Printf("Current future price: %s\n", ticker.Last)

	// Spot trading pair
	spotSymbol := "DOGE/USDT"
	// Get current spot price
	ticker, err = exchange.FetchTicker(ctx, spotSymbol)
	if err != nil {
		fmt.Printf("Failed to fetch spot ticker: %v", err)
		return
	}
	fmt.Printf("Current spot price: %s\n", ticker.Last)

	amount := "0.001" // Order quantity (adjust according to actual situation)

	// ========== Contract Trading Examples ==========
	fmt.Println("=== Contract Trading Examples ===")

	// 1. Open long position (buy to open long)
	fmt.Println("\n1. Open long position")
	openLongPosition(ctx, exchange, futureSymbol, amount)

	// 2. Close long position (sell to close long)
	fmt.Println("\n2. Close long position")
	closeLongPosition(ctx, exchange, futureSymbol, amount)

	// 3. Open short position (sell to open short)
	fmt.Println("\n3. Open short position")
	openShortPosition(ctx, exchange, futureSymbol, amount)

	// 4. Close short position (buy to close short)
	fmt.Println("\n4. Close short position")
	closeShortPosition(ctx, exchange, futureSymbol, amount)

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
func openLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	// Open long position: buy with PositionSideLong
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, amount, types.WithPositionSide(types.PositionSideLong))
	if err != nil {
		fmt.Printf("Failed to open long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeLongPosition closes a long position (sell to close long)
func closeLongPosition(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	// Close long position: sell with PositionSideLong
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, amount, types.WithPositionSide(types.PositionSideLong))
	if err != nil {
		fmt.Printf("Failed to close long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// openShortPosition opens a short position (sell to open short)
func openShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	// Open short position: sell with PositionSideShort
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, amount, types.WithPositionSide(types.PositionSideShort))
	if err != nil {
		fmt.Printf("Failed to open short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeShortPosition closes a short position (buy to close short)
func closeShortPosition(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	// Close short position: buy with PositionSideShort
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, amount, types.WithPositionSide(types.PositionSideShort))
	if err != nil {
		fmt.Printf("Failed to close short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// buySpot buys spot assets
func buySpot(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, amount)
	if err != nil {
		fmt.Printf("Failed to buy spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// sellSpot sells spot assets
func sellSpot(ctx context.Context, exchange base.Exchange, symbol string, amount string) {
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, amount)
	if err != nil {
		fmt.Printf("Failed to sell spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}
