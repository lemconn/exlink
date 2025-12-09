package binance

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestBinancePerp_FetchOHLCV(t *testing.T) {
	// 创建 Binance 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create Binance instance: %v", err)
	}

	perp := ex.Perp()

	// 加载市场信息
	ctx := context.Background()
	if err := perp.LoadMarkets(ctx, false); err != nil {
		if shouldSkipTestForHTTPError(t, err) {
			return
		}
		t.Fatalf("Failed to load markets: %v", err)
	}

	// 测试 FetchOHLCV
	symbol := "BTC/USDT:USDT"
	timeframe := "1h"
	since := time.Time{} // 不指定开始时间，获取最新数据
	limit := 10

	ohlcvs, err := perp.FetchOHLCV(ctx, symbol, timeframe, since, limit)
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
