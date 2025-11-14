package gate

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lemconn/exlink/types"
)

func getProxyURL() string {
	return os.Getenv("PROXY_URL")
}

func getGateAPIKey() string {
	return os.Getenv("GATE_API_KEY")
}

func getGateSecretKey() string {
	return os.Getenv("GATE_SECRET_KEY")
}

func getOptions() map[string]interface{} {
	options := map[string]interface{}{
		"fetchMarkets": []string{"spot", "swap"},
		"sandbox":      true, // Use sandbox mode
	}
	if proxyURL := getProxyURL(); proxyURL != "" {
		options["proxy"] = proxyURL
	}
	return options
}

// isNetworkError checks if the error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()

	// Check for HTTP errors that indicate network/access issues
	if strings.Contains(errStr, "http error 403") ||
		strings.Contains(errStr, "http error 429") ||
		strings.Contains(errStr, "http error 451") {
		return true
	}

	// Check for network connection errors
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") {
		return true
	}

	return false
}

// skipIfNetworkError skips the test if it's a network error and no proxy is configured
func skipIfNetworkError(t *testing.T, err error) {
	if err != nil && isNetworkError(err) && getProxyURL() == "" {
		t.Skipf("Skipping test due to network error (no proxy configured): %v", err)
	}
}

func TestGate_FetchOHLCV(t *testing.T) {
	ctx := context.Background()

	// Create Gate instance
	exchange, err := NewGate("", "", getOptions())
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	timeframe := "1h"
	limit := 10

	// Test fetching spot OHLCV
	symbol := "BTC/USDT"
	fmt.Printf("Testing spot OHLCV: %s, timeframe: %s, limit: %d\n", symbol, timeframe, limit)

	ohlcvs, err := exchange.FetchOHLCV(ctx, symbol, timeframe, time.Time{}, limit)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch OHLCV: %v", err)
	}

	if len(ohlcvs) == 0 {
		t.Fatal("No OHLCV data received")
	}

	fmt.Printf("Successfully fetched %d OHLCV candles\n", len(ohlcvs))

	// Print first few candles
	for i, ohlcv := range ohlcvs {
		if i >= 3 {
			break
		}
		fmt.Printf("Candle %d: time=%s, open=%f, high=%f, low=%f, close=%f, volume=%f\n",
			i+1,
			ohlcv.Timestamp.Format("2006-01-02 15:04:05"),
			ohlcv.Open,
			ohlcv.High,
			ohlcv.Low,
			ohlcv.Close,
			ohlcv.Volume,
		)
	}

	// Test fetching swap OHLCV
	symbol = "BTC/USDT:USDT"
	fmt.Printf("Testing swap OHLCV: %s, timeframe: %s, limit: %d\n", symbol, timeframe, limit)

	ohlcvs, err = exchange.FetchOHLCV(ctx, symbol, timeframe, time.Time{}, limit)
	if err != nil {
		t.Fatalf("Failed to fetch OHLCV: %v", err)
	}

	if len(ohlcvs) == 0 {
		t.Fatal("No OHLCV data received")
	}

	fmt.Printf("Successfully fetched %d OHLCV candles\n", len(ohlcvs))

	// Print first few candles
	for i, ohlcv := range ohlcvs {
		if i >= 3 {
			break
		}
		fmt.Printf("Candle %d: time=%s, open=%f, high=%f, low=%f, close=%f, volume=%f\n",
			i+1,
			ohlcv.Timestamp.Format("2006-01-02 15:04:05"),
			ohlcv.Open,
			ohlcv.High,
			ohlcv.Low,
			ohlcv.Close,
			ohlcv.Volume,
		)
	}
}

func TestGate_FetchTicker(t *testing.T) {
	ctx := context.Background()

	// Create Gate instance
	exchange, err := NewGate("", "", getOptions())
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	// Test fetching spot ticker
	symbol := "BTC/USDT"
	fmt.Printf("Testing spot ticker: %s\n", symbol)

	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Spot ticker: bid=%f, ask=%f, last=%f, high24h=%f, low24h=%f, volume24h=%f\n",
		ticker.Bid,
		ticker.Ask,
		ticker.Last,
		ticker.High,
		ticker.Low,
		ticker.Volume,
	)

	// Test fetching swap ticker
	symbol = "BTC/USDT:USDT"
	fmt.Printf("Testing swap ticker: %s\n", symbol)

	ticker, err = exchange.FetchTicker(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Swap ticker: bid=%f, ask=%f, last=%f, high24h=%f, low24h=%f, volume24h=%f\n",
		ticker.Bid,
		ticker.Ask,
		ticker.Last,
		ticker.High,
		ticker.Low,
		ticker.Volume,
	)
}

// TestGate_CreateContractOrder_BuyOpenLong tests buying to open a long position
func TestGate_CreateContractOrder_BuyOpenLong(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT:USDT"
	amount := 0.1 // Order amount

	// Fetch current price (optional, using market order here)
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing buy open long: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Buy to open long: side="buy", size is positive for long
	params := map[string]interface{}{
		"size": strconv.FormatFloat(amount, 'f', -1, 64), // Positive size for long
	}

	// Use market order
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create buy open long order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}

// TestGate_CreateContractOrder_SellCloseLong tests selling to close a long position
func TestGate_CreateContractOrder_SellCloseLong(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT:USDT"
	amount := 0.1 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing sell close long: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Sell to close long: side="sell", size is negative to close long
	params := map[string]interface{}{
		"size": strconv.FormatFloat(-amount, 'f', -1, 64), // Negative size to close long
	}

	// Use market order
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create sell close long order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}

// TestGate_CreateContractOrder_SellOpenShort tests selling to open a short position
func TestGate_CreateContractOrder_SellOpenShort(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT:USDT"
	amount := 0.1 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing sell open short: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Sell to open short: side="sell", size is negative for short
	params := map[string]interface{}{
		"size": strconv.FormatFloat(-amount, 'f', -1, 64), // Negative size for short
	}

	// Use market order
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create sell open short order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}

// TestGate_CreateContractOrder_BuyCloseShort tests buying to close a short position
func TestGate_CreateContractOrder_BuyCloseShort(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT:USDT"
	amount := 0.1 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing buy close short: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Buy to close short: side="buy", size is positive to close short
	params := map[string]interface{}{
		"size": strconv.FormatFloat(amount, 'f', -1, 64), // Positive size to close short
	}

	// Use market order
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create buy close short order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}

// TestGate_FetchBalance tests fetching balance
func TestGate_FetchBalance(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	// Fetch balance
	balances, err := exchange.FetchBalance(ctx)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch balance: %v", err)
	}

	if balances == nil {
		t.Fatal("Balance data is nil")
	}

	fmt.Printf("Successfully fetched balance, total currencies: %d\n", len(balances))

	// Print balance information
	for currency, balance := range balances {
		if balance.Total > 0 {
			fmt.Printf("Currency: %s, Total: %f, Free: %f, Used: %f\n",
				currency, balance.Total, balance.Free, balance.Used)
		}
	}
}

// TestGate_CreateSpotOrder_Buy tests buying spot
func TestGate_CreateSpotOrder_Buy(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT"
	amount := 0.01 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing spot buy: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Use market order for spot buy
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create spot buy order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}

// TestGate_CreateSpotOrder_Sell tests selling spot
func TestGate_CreateSpotOrder_Sell(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getGateAPIKey()
	secretKey := getGateSecretKey()
	if apiKey == "" || secretKey == "" {
		t.Skip("Gate API credentials not set in environment variables")
	}

	// Create Gate instance
	options := getOptions()
	exchange, err := NewGate(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT"
	amount := 0.01 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing spot sell: %s, amount: %f\n", symbol, amount)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", ticker.Bid, ticker.Ask, ticker.Last)

	// Use market order for spot sell
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to create spot sell order: %v", err)
	}

	fmt.Printf("Order created successfully: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
		order.ID, order.Symbol, order.Side, order.Amount)

	// Wait a bit for order processing
	time.Sleep(2 * time.Second)

	// Query order status
	fetchedOrder, err := exchange.FetchOrder(ctx, order.ID, symbol)
	if err != nil {
		t.Logf("Warning: Failed to fetch order: %v", err)
	} else {
		fmt.Printf("Order status: ID=%s, Status=%s, Filled=%f, Remaining=%f\n",
			fetchedOrder.ID, fetchedOrder.Status, fetchedOrder.Filled, fetchedOrder.Remaining)
	}
}
