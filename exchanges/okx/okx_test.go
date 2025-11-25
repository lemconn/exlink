package okx

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

func getOKXAPIKey() string {
	return os.Getenv("OKX_API_KEY")
}

func getOKXSecretKey() string {
	return os.Getenv("OKX_SECRET_KEY")
}

func getOKXPassphrase() string {
	return os.Getenv("OKX_PASSPHRASE")
}

func getOptions() map[string]interface{} {
	options := map[string]interface{}{
		"fetchMarkets": []string{"spot", "swap"},
		"sandbox":      true, // Use sandbox mode
	}
	if proxyURL := getProxyURL(); proxyURL != "" {
		options["proxy"] = proxyURL
	}
	if passphrase := getOKXPassphrase(); passphrase != "" {
		options["passphrase"] = passphrase
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

func TestOKX_FetchOHLCV(t *testing.T) {
	ctx := context.Background()

	// Create OKX instance
	exchange, err := NewOKX("", "", getOptions())
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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

func TestOKX_FetchTicker(t *testing.T) {
	ctx := context.Background()

	// Create OKX instance
	exchange, err := NewOKX("", "", getOptions())
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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

	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	high, _ := strconv.ParseFloat(ticker.High, 64)
	low, _ := strconv.ParseFloat(ticker.Low, 64)
	volume, _ := strconv.ParseFloat(ticker.Volume, 64)
	fmt.Printf("Spot ticker: bid=%f, ask=%f, last=%f, high24h=%f, low24h=%f, volume24h=%f\n",
		bid, ask, last, high, low, volume,
	)

	// Test fetching swap ticker
	symbol = "BTC/USDT:USDT"
	fmt.Printf("Testing swap ticker: %s\n", symbol)

	ticker, err = exchange.FetchTicker(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	bid, _ = strconv.ParseFloat(ticker.Bid, 64)
	ask, _ = strconv.ParseFloat(ticker.Ask, 64)
	last, _ = strconv.ParseFloat(ticker.Last, 64)
	high, _ = strconv.ParseFloat(ticker.High, 64)
	low, _ = strconv.ParseFloat(ticker.Low, 64)
	volume, _ = strconv.ParseFloat(ticker.Volume, 64)
	fmt.Printf("Swap ticker: bid=%f, ask=%f, last=%f, high24h=%f, low24h=%f, volume24h=%f\n",
		bid, ask, last, high, low, volume,
	)
}

// TestOKX_CreateContractOrder_BuyOpenLong tests buying to open a long position
func TestOKX_CreateContractOrder_BuyOpenLong(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Buy to open long: side="buy", posSide="long"
	// Use market order with posSide="long" to open long position
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, strconv.FormatFloat(amount, 'f', -1, 64), types.WithExtraParam("posSide", "long"))
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

// TestOKX_CreateContractOrder_SellCloseLong tests selling to close a long position
func TestOKX_CreateContractOrder_SellCloseLong(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Sell to close long: side="sell", posSide="long"
	// Use market order with posSide="long" to close long position
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, strconv.FormatFloat(amount, 'f', -1, 64), types.WithExtraParam("posSide", "long"))
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

// TestOKX_CreateContractOrder_SellOpenShort tests selling to open a short position
func TestOKX_CreateContractOrder_SellOpenShort(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Sell to open short: side="sell", posSide="short"
	// Use market order with posSide="short" to open short position
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, strconv.FormatFloat(amount, 'f', -1, 64), types.WithExtraParam("posSide", "short"))
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

// TestOKX_CreateContractOrder_BuyCloseShort tests buying to close a short position
func TestOKX_CreateContractOrder_BuyCloseShort(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Buy to close short: side="buy", posSide="short"
	// Use market order with posSide="short" to close short position
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, strconv.FormatFloat(amount, 'f', -1, 64), types.WithExtraParam("posSide", "short"))
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

// TestOKX_FetchBalance tests fetching balance
func TestOKX_FetchBalance(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
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

// TestOKX_CreateSpotOrder_Buy tests buying spot
func TestOKX_CreateSpotOrder_Buy(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT"
	amount := 0.1 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing spot buy: %s, amount: %f\n", symbol, amount)
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Use market order for spot buy
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideBuy, strconv.FormatFloat(amount, 'f', -1, 64))
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

// TestOKX_CreateSpotOrder_Sell tests selling spot
func TestOKX_CreateSpotOrder_Sell(t *testing.T) {
	ctx := context.Background()

	// Read API credentials from environment variables
	apiKey := getOKXAPIKey()
	secretKey := getOKXSecretKey()
	passphrase := getOKXPassphrase()
	if apiKey == "" || secretKey == "" || passphrase == "" {
		t.Skip("OKX API credentials not set in environment variables")
	}

	// Create OKX instance
	options := getOptions()
	exchange, err := NewOKX(apiKey, secretKey, options)
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
	}

	// Load markets
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to load markets: %v", err)
	}

	symbol := "SOL/USDT"
	amount := 0.1 // Order amount

	// Fetch current price
	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
		skipIfNetworkError(t, err)
		t.Fatalf("Failed to fetch ticker: %v", err)
	}

	fmt.Printf("Testing spot sell: %s, amount: %f\n", symbol, amount)
	bid, _ := strconv.ParseFloat(ticker.Bid, 64)
	ask, _ := strconv.ParseFloat(ticker.Ask, 64)
	last, _ := strconv.ParseFloat(ticker.Last, 64)
	fmt.Printf("Current price: bid=%f, ask=%f, last=%f\n", bid, ask, last)

	// Use market order for spot sell
	order, err := exchange.CreateOrder(ctx, symbol, types.OrderSideSell, strconv.FormatFloat(amount, 'f', -1, 64))
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
