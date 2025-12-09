package okx

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lemconn/exlink/exchange"
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

func TestOKXSpot_FetchOHLCV(t *testing.T) {
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

	// 测试 FetchOHLCV
	symbol := "BTC/USDT"
	timeframe := "1H"
	since := time.Time{} // 不指定开始时间，获取最新数据
	limit := 10

	ohlcvs, err := spot.FetchOHLCV(ctx, symbol, timeframe, since, limit)
	if err != nil {
		t.Fatalf("Failed to fetch OHLCV: %v", err)
	}

	// 验证结果
	if len(ohlcvs) == 0 {
		t.Error("Expected at least one OHLCV data point, got 0")
	}

	// 验证数据结构
	for i, ohlcv := range ohlcvs {
		if ohlcv.Timestamp.Time.IsZero() {
			t.Errorf("OHLCV[%d]: Timestamp is zero", i)
		}
		if ohlcv.Open.Decimal.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Open price should be positive, got %s", i, ohlcv.Open.Decimal.String())
		}
		if ohlcv.High.Decimal.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: High price should be positive, got %s", i, ohlcv.High.Decimal.String())
		}
		if ohlcv.Low.Decimal.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Low price should be positive, got %s", i, ohlcv.Low.Decimal.String())
		}
		if ohlcv.Close.Decimal.LessThanOrEqual(decimal.Zero) {
			t.Errorf("OHLCV[%d]: Close price should be positive, got %s", i, ohlcv.Close.Decimal.String())
		}
		if ohlcv.Volume.Decimal.IsNegative() {
			t.Errorf("OHLCV[%d]: Volume should be non-negative, got %s", i, ohlcv.Volume.Decimal.String())
		}

		// 验证价格逻辑：High >= Low, High >= Open, High >= Close, Low <= Open, Low <= Close
		if ohlcv.High.Decimal.LessThan(ohlcv.Low.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Low (%s)", i, ohlcv.High.Decimal.String(), ohlcv.Low.Decimal.String())
		}
		if ohlcv.High.Decimal.LessThan(ohlcv.Open.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Open (%s)", i, ohlcv.High.Decimal.String(), ohlcv.Open.Decimal.String())
		}
		if ohlcv.High.Decimal.LessThan(ohlcv.Close.Decimal) {
			t.Errorf("OHLCV[%d]: High (%s) should be >= Close (%s)", i, ohlcv.High.Decimal.String(), ohlcv.Close.Decimal.String())
		}
		if ohlcv.Low.Decimal.GreaterThan(ohlcv.Open.Decimal) {
			t.Errorf("OHLCV[%d]: Low (%s) should be <= Open (%s)", i, ohlcv.Low.Decimal.String(), ohlcv.Open.Decimal.String())
		}
		if ohlcv.Low.Decimal.GreaterThan(ohlcv.Close.Decimal) {
			t.Errorf("OHLCV[%d]: Low (%s) should be <= Close (%s)", i, ohlcv.Low.Decimal.String(), ohlcv.Close.Decimal.String())
		}
	}

	t.Logf("Successfully fetched %d OHLCV data points for %s", len(ohlcvs), symbol)
	if len(ohlcvs) > 0 {
		t.Logf("First OHLCV: Time=%s, Open=%s, High=%s, Low=%s, Close=%s, Volume=%s",
			ohlcvs[0].Timestamp.Time.Format(time.RFC3339),
			ohlcvs[0].Open.Decimal.String(), ohlcvs[0].High.Decimal.String(), ohlcvs[0].Low.Decimal.String(), ohlcvs[0].Close.Decimal.String(), ohlcvs[0].Volume.Decimal.String())
	}
}
