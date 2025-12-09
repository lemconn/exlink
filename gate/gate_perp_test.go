package gate

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestGatePerp_FetchOHLCV(t *testing.T) {
	// 创建 Gate 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create Gate instance: %v", err)
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
