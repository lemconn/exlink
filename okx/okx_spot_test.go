package okx

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/option"
	"github.com/shopspring/decimal"
)

// setupTestExchange 从环境变量创建测试用的交易所实例
func setupTestExchange() (exchange.Exchange, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("OKX_API_KEY")
	secretKey := os.Getenv("OKX_SECRET_KEY")
	password := os.Getenv("OKX_PASSWORD")
	proxyURL := os.Getenv("PROXY_URL")

	// 构建选项
	options := make(map[string]interface{})
	if proxyURL != "" {
		options["proxy"] = proxyURL
	}
	if password != "" {
		options["password"] = password
	}

	return NewOKX(apiKey, secretKey, options)
}

// shouldSkipTestForHTTPError 检查错误是否包含需要跳过测试的 HTTP 错误码
// 如果错误信息包含 "http error 451"、"http error 429" 或 "http error 403"，返回 true
func shouldSkipTestForHTTPError(t *testing.T, err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	skipErrors := []string{
		"http error 451",
		"http error 429",
		"http error 403",
	}

	for _, skipErr := range skipErrors {
		if strings.Contains(errMsg, skipErr) {
			t.Skipf("Skipping test due to HTTP error: %v", err)
			return true
		}
	}

	return false
}

func TestOKXSpot_FetchOHLCVs(t *testing.T) {
	// 创建 OKX 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
	}

	spot := ex.Spot()

	// 加载市场信息
	ctx := context.Background()
	if err := spot.LoadMarkets(ctx, false); err != nil {
		if shouldSkipTestForHTTPError(t, err) {
			return
		}
		t.Fatalf("Failed to load markets: %v", err)
	}

	// 测试 FetchOHLCVs
	symbol := "BTC/USDT"
	timeframe := "1H"
	since := time.Time{} // 不指定开始时间，获取最新数据
	limit := 10

	ohlcvs, err := spot.FetchOHLCVs(ctx, symbol, timeframe,
		option.WithLimit(limit),
		option.WithSince(since),
	)
	if err != nil {
		t.Fatalf("Failed to fetch OHLCV: %v", err)
	}

	// 验证结果
	if len(ohlcvs) == 0 {
		t.Error("Expected at least one OHLCV data point, got 0")
	}

	// 验证数据结构
	for i, ohlcv := range ohlcvs {
		if ohlcv.Timestamp.IsZero() {
			t.Errorf("OHLCV[%d]: Timestamp is zero", i)
		}
		if ohlcv.Open.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Open price should be positive, got %s", i, ohlcv.Open.String())
		}
		if ohlcv.High.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: High price should be positive, got %s", i, ohlcv.High.String())
		}
		if ohlcv.Low.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Low price should be positive, got %s", i, ohlcv.Low.String())
		}
		if ohlcv.Close.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Close price should be positive, got %s", i, ohlcv.Close.String())
		}
		if ohlcv.Volume.IsNegative() {
			t.Errorf("OHLCV[%d]: Volume should be non-negative, got %s", i, ohlcv.Volume.String())
		}

		// 验证价格逻辑：High >= Low, High >= Open, High >= Close, Low <= Open, Low <= Close
		if ohlcv.High.LessThan(ohlcv.Low.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Low (%s)", i, ohlcv.High.String(), ohlcv.Low.String())
		}
		if ohlcv.High.LessThan(ohlcv.Open.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Open (%s)", i, ohlcv.High.String(), ohlcv.Open.String())
		}
		if ohlcv.High.LessThan(ohlcv.Close.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Close (%s)", i, ohlcv.High.String(), ohlcv.Close.String())
		}
		if ohlcv.Low.GreaterThan(ohlcv.Open.Decimal) {
			t.Errorf("OHLCV[%d]: Low (%s) should be <= Open (%s)", i, ohlcv.Low.String(), ohlcv.Open.String())
		}
		if ohlcv.Low.GreaterThan(ohlcv.Close.Decimal) {
			t.Errorf("OHLCV[%d]: Low (%s) should be <= Close (%s)", i, ohlcv.Low.String(), ohlcv.Close.String())
		}
	}

	t.Logf("Successfully fetched %d OHLCV data points for %s", len(ohlcvs), symbol)
	if len(ohlcvs) > 0 {
		t.Logf("First OHLCV: Time=%s, Open=%s, High=%s, Low=%s, Close=%s, Volume=%s",
			ohlcvs[0].Timestamp.Format(time.RFC3339),
			ohlcvs[0].Open.String(), ohlcvs[0].High.String(), ohlcvs[0].Low.String(), ohlcvs[0].Close.String(), ohlcvs[0].Volume.String())
	}
}

func TestOKXSpot_FetchMarkets(t *testing.T) {
	// 创建 OKX 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
	}

	spot := ex.Spot()

	// 测试 FetchMarkets
	ctx := context.Background()
	markets, err := spot.FetchMarkets(ctx)
	if err != nil {
		if shouldSkipTestForHTTPError(t, err) {
			return
		}
		t.Fatalf("Failed to fetch markets: %v", err)
	}

	// 验证结果
	if len(markets) == 0 {
		t.Error("Expected at least one market, got 0")
	}

	// 验证数据结构
	for i, market := range markets {
		if market.Symbol == "" {
			t.Errorf("Market[%d]: Symbol is empty", i)
		}
		if market.Base == "" {
			t.Errorf("Market[%d]: Base is empty", i)
		}
		if market.Quote == "" {
			t.Errorf("Market[%d]: Quote is empty", i)
		}
	}

	t.Logf("Successfully fetched %d markets", len(markets))
	if len(markets) > 0 {
		t.Logf("First market: %s", markets[0].Symbol)
	}
}

func TestOKXSpot_FetchTickers(t *testing.T) {
	// 创建 OKX 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create OKX instance: %v", err)
	}

	spot := ex.Spot()

	// 加载市场信息
	ctx := context.Background()
	if err := spot.LoadMarkets(ctx, false); err != nil {
		if shouldSkipTestForHTTPError(t, err) {
			return
		}
		t.Fatalf("Failed to load markets: %v", err)
	}

	// 测试 FetchTickers
	tickers, err := spot.FetchTickers(ctx)
	if err != nil {
		if shouldSkipTestForHTTPError(t, err) {
			return
		}
		t.Fatalf("Failed to fetch tickers: %v", err)
	}

	// 验证结果
	if len(tickers) == 0 {
		t.Error("Expected at least one ticker, got 0")
	}

	// 验证数据结构
	positivePriceCount := 0
	for symbol, ticker := range tickers {
		if symbol == "" {
			t.Error("Ticker symbol is empty")
		}
		if ticker.Symbol == "" {
			t.Errorf("Ticker[%s]: Symbol is empty", symbol)
		}
		// 允许 Last 价格为 0（已下架或暂停的交易对），但不允许为负数
		if ticker.Last.IsNegative() {
			t.Errorf("Ticker[%s]: Last price should not be negative, got %s", symbol, ticker.Last.String())
		}
		if ticker.Last.GreaterThan(decimal.Zero) {
			positivePriceCount++
		}
	}

	// 至少应该有一些 ticker 的价格是正数
	if positivePriceCount == 0 {
		t.Error("Expected at least one ticker with positive price, got 0")
	}

	t.Logf("Successfully fetched %d tickers", len(tickers))
	if len(tickers) > 0 {
		// 打印第一个 ticker
		for symbol, ticker := range tickers {
			t.Logf("First ticker: %s, Last=%s", symbol, ticker.Last.String())
			break
		}
	}
}
