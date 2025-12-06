package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

func main() {
	ctx := context.Background()

	// Get API keys from environment variables (please set them in actual use)
	apiKey := os.Getenv("BYBIT_API_KEY")
	secretKey := os.Getenv("BYBIT_SECRET_KEY")

	// Get proxy URL from environment variable
	proxyURL := os.Getenv("PROXY_URL")

	if apiKey == "" || secretKey == "" {
		fmt.Println("Warning: BYBIT_API_KEY and BYBIT_SECRET_KEY environment variables are not set")
		fmt.Println("Example code will demonstrate usage, but actual calls will fail")
		return
	}

	// Create Bybit exchange instance
	opts := []exlink.Option{
		exlink.WithAPIKey(apiKey),
		exlink.WithSecretKey(secretKey),
		exlink.WithProxy(proxyURL),
		exlink.WithSandbox(true), // Sandbox mode
	}

	ex, err := exlink.NewExchange(exlink.ExchangeBybit, opts...)
	if err != nil {
		fmt.Printf("Failed to create exchange instance: %v", err)
		return
	}

	// Get spot and perp interfaces
	spot := ex.Spot()
	perp := ex.Perp()

	// Load market information
	if err := spot.LoadMarkets(ctx, false); err != nil {
		fmt.Printf("Failed to load spot markets: %v", err)
		return
	}
	if err := perp.LoadMarkets(ctx, false); err != nil {
		fmt.Printf("Failed to load perp markets: %v", err)
		return
	}

	// Future trading pair (perpetual future)
	futureSymbol := "DOGE/USDT:USDT"
	// Get current future price
	ticker, err := perp.FetchTicker(ctx, futureSymbol)
	if err != nil {
		fmt.Printf("Failed to fetch future ticker: %v", err)
		return
	}
	fmt.Printf("Current future price: %s\n", ticker.Last)

	// Spot trading pair
	spotSymbol := "DOGE/USDT"
	// Get current spot price
	ticker, err = spot.FetchTicker(ctx, spotSymbol)
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
	openLongPosition(ctx, perp, futureSymbol, amount)

	// 2. Close long position (sell to close long)
	fmt.Println("\n2. Close long position")
	closeLongPosition(ctx, perp, futureSymbol, amount)

	// 3. Open short position (sell to open short)
	fmt.Println("\n3. Open short position")
	openShortPosition(ctx, perp, futureSymbol, amount)

	// 4. Close short position (buy to close short)
	fmt.Println("\n4. Close short position")
	closeShortPosition(ctx, perp, futureSymbol, amount)

	// ========== Spot Trading Examples ==========
	fmt.Println("\n=== Spot Trading Examples ===")

	// 5. Buy spot
	fmt.Println("\n5. Buy spot")
	buySpot(ctx, spot, spotSymbol, amount)

	// 6. Sell spot
	fmt.Println("\n6. Sell spot")
	sellSpot(ctx, spot, spotSymbol, amount)
}

// openLongPosition opens a long position (buy to open long)
func openLongPosition(ctx context.Context, perp exchange.PerpExchange, symbol string, amount string) {
	// Bybit uses reduceOnly=false to open position (if supported)
	order, err := perp.CreateOrder(ctx, symbol, types.OrderSideBuy, amount, types.WithPositionSide(types.PositionSideLong))
	if err != nil {
		fmt.Printf("Failed to open long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeLongPosition closes a long position (sell to close long)
func closeLongPosition(ctx context.Context, perp exchange.PerpExchange, symbol string, amount string) {
	// Close long position: sell with PositionSideLong
	order, err := perp.CreateOrder(ctx, symbol, types.OrderSideSell, amount, types.WithPositionSide(types.PositionSideLong))
	if err != nil {
		fmt.Printf("Failed to close long position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// openShortPosition opens a short position (sell to open short)
func openShortPosition(ctx context.Context, perp exchange.PerpExchange, symbol string, amount string) {
	// Open short position: sell with PositionSideShort
	order, err := perp.CreateOrder(ctx, symbol, types.OrderSideSell, amount, types.WithPositionSide(types.PositionSideShort))
	if err != nil {
		fmt.Printf("Failed to open short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// closeShortPosition closes a short position (buy to close short)
func closeShortPosition(ctx context.Context, perp exchange.PerpExchange, symbol string, amount string) {
	// Bybit uses reduceOnly=true to close position (if supported)
	order, err := perp.CreateOrder(ctx, symbol, types.OrderSideBuy, amount, types.WithPositionSide(types.PositionSideShort))
	if err != nil {
		fmt.Printf("Failed to close short position: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// buySpot buys spot assets
func buySpot(ctx context.Context, spot exchange.SpotExchange, symbol string, amount string) {
	order, err := spot.CreateOrder(ctx, symbol, types.OrderSideBuy, amount)
	if err != nil {
		fmt.Printf("Failed to buy spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}

// sellSpot sells spot assets
func sellSpot(ctx context.Context, spot exchange.SpotExchange, symbol string, amount string) {
	order, err := spot.CreateOrder(ctx, symbol, types.OrderSideSell, amount)
	if err != nil {
		fmt.Printf("Failed to sell spot: %v", err)
	} else {
		fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}
}
