package binance

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lemconn/exlink/exchange"
)

// setupTestExchange 从环境变量创建测试用的交易所实例
func setupTestExchange() (exchange.Exchange, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	proxyURL := os.Getenv("PROXY_URL")

	// 构建选项
	options := make(map[string]interface{})
	if proxyURL != "" {
		options["proxy"] = proxyURL
	}

	return NewBinance(apiKey, secretKey, options)
}

func TestBinanceSpot_FetchOHLCV(t *testing.T) {
	// 创建 Binance 实例（从环境变量获取配置）
	ex, err := setupTestExchange()
	if err != nil {
		t.Fatalf("Failed to create Binance instance: %v", err)
	}

	spot := ex.Spot()

	// 加载市场信息
	ctx := context.Background()
	if err := spot.LoadMarkets(ctx, false); err != nil {
		t.Fatalf("Failed to load markets: %v", err)
	}

	// 测试 FetchOHLCV
	symbol := "BTC/USDT"
	timeframe := "1h"
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
		if ohlcv.Timestamp.IsZero() {
			t.Errorf("OHLCV[%d]: Timestamp is zero", i)
		}
		if ohlcv.Open <= 0 {
			t.Errorf("OHLCV[%d]: Open price should be positive, got %f", i, ohlcv.Open)
		}
		if ohlcv.High <= 0 {
			t.Errorf("OHLCV[%d]: High price should be positive, got %f", i, ohlcv.High)
		}
		if ohlcv.Low <= 0 {
			t.Errorf("OHLCV[%d]: Low price should be positive, got %f", i, ohlcv.Low)
		}
		if ohlcv.Close <= 0 {
			t.Errorf("OHLCV[%d]: Close price should be positive, got %f", i, ohlcv.Close)
		}
		if ohlcv.Volume < 0 {
			t.Errorf("OHLCV[%d]: Volume should be non-negative, got %f", i, ohlcv.Volume)
		}

		// 验证价格逻辑：High >= Low, High >= Open, High >= Close, Low <= Open, Low <= Close
		if ohlcv.High < ohlcv.Low {
			t.Errorf("OHLCV[%d]: High (%f) should be >= Low (%f)", i, ohlcv.High, ohlcv.Low)
		}
		if ohlcv.High < ohlcv.Open {
			t.Errorf("OHLCV[%d]: High (%f) should be >= Open (%f)", i, ohlcv.High, ohlcv.Open)
		}
		if ohlcv.High < ohlcv.Close {
			t.Errorf("OHLCV[%d]: High (%f) should be >= Close (%f)", i, ohlcv.High, ohlcv.Close)
		}
		if ohlcv.Low > ohlcv.Open {
			t.Errorf("OHLCV[%d]: Low (%f) should be <= Open (%f)", i, ohlcv.Low, ohlcv.Open)
		}
		if ohlcv.Low > ohlcv.Close {
			t.Errorf("OHLCV[%d]: Low (%f) should be <= Close (%f)", i, ohlcv.Low, ohlcv.Close)
		}
	}

	t.Logf("Successfully fetched %d OHLCV data points for %s", len(ohlcvs), symbol)
	if len(ohlcvs) > 0 {
		t.Logf("First OHLCV: Time=%s, Open=%f, High=%f, Low=%f, Close=%f, Volume=%f",
			ohlcvs[0].Timestamp.Format(time.RFC3339),
			ohlcvs[0].Open, ohlcvs[0].High, ohlcvs[0].Low, ohlcvs[0].Close, ohlcvs[0].Volume)
	}
}
