package okx

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func getProxyURL() string {
	return os.Getenv("PROXY_URL")
}

func getOptions() map[string]interface{} {
	options := map[string]interface{}{
		"fetchMarkets": []string{"spot", "swap"},
	}
	if proxyURL := getProxyURL(); proxyURL != "" {
		options["proxy"] = proxyURL
	}
	return options
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
		t.Fatalf("Failed to load markets: %v", err)
	}

	timeframe := "1h"
	limit := 10

	// Test fetching spot OHLCV
	symbol := "BTC/USDT"
	fmt.Printf("Testing spot OHLCV: %s, timeframe: %s, limit: %d\n", symbol, timeframe, limit)

	ohlcvs, err := exchange.FetchOHLCV(ctx, symbol, timeframe, time.Time{}, limit)
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
		t.Fatalf("Failed to load markets: %v", err)
	}

	// Test fetching spot ticker
	symbol := "BTC/USDT"
	fmt.Printf("Testing spot ticker: %s\n", symbol)

	ticker, err := exchange.FetchTicker(ctx, symbol)
	if err != nil {
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
