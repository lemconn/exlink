package binance

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBinance_FetchOHLCV_Spot(t *testing.T) {
	ctx := context.Background()

	// 创建 Binance 实例，使用代理
	exchange, err := NewBinance("", "", map[string]interface{}{
		"proxy":        "http://127.0.0.1:7890",
		"fetchMarkets": []string{"spot"},
	})
	if err != nil {
		t.Fatalf("创建 Binance 实例失败: %v", err)
	}

	// 加载市场信息（只加载现货）
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		t.Fatalf("加载市场信息失败: %v", err)
	}

	// 测试获取现货 K 线
	symbol := "BTC/USDT"
	timeframe := "1h"
	limit := 10

	fmt.Printf("测试获取现货 K 线: %s, 时间框架: %s, 数量: %d\n", symbol, timeframe, limit)

	ohlcvs, err := exchange.FetchOHLCV(ctx, symbol, timeframe, time.Time{}, limit)
	if err != nil {
		t.Fatalf("获取 K 线失败: %v", err)
	}

	if len(ohlcvs) == 0 {
		t.Fatal("未获取到 K 线数据")
	}

	fmt.Printf("成功获取 %d 条 K 线数据\n", len(ohlcvs))

	// 打印前几条数据
	for i, ohlcv := range ohlcvs {
		if i >= 3 {
			break
		}
		fmt.Printf("K线 %d: 时间=%s, 开=%f, 高=%f, 低=%f, 收=%f, 量=%f\n",
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

func TestBinance_FetchOHLCV_Swap(t *testing.T) {
	ctx := context.Background()

	// 创建 Binance 实例，使用代理
	exchange, err := NewBinance("", "", map[string]interface{}{
		"proxy":        "http://127.0.0.1:7890",
		"fetchMarkets": []string{"swap"},
	})
	if err != nil {
		t.Fatalf("创建 Binance 实例失败: %v", err)
	}

	// 加载市场信息（只加载永续合约）
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		t.Fatalf("加载市场信息失败: %v", err)
	}

	// 测试获取永续合约 K 线
	symbol := "BTC/USDT:USDT"
	timeframe := "1h"
	limit := 10

	fmt.Printf("测试获取永续合约 K 线: %s, 时间框架: %s, 数量: %d\n", symbol, timeframe, limit)

	ohlcvs, err := exchange.FetchOHLCV(ctx, symbol, timeframe, time.Time{}, limit)
	if err != nil {
		t.Fatalf("获取 K 线失败: %v", err)
	}

	if len(ohlcvs) == 0 {
		t.Fatal("未获取到 K 线数据")
	}

	fmt.Printf("成功获取 %d 条 K 线数据\n", len(ohlcvs))

	// 打印前几条数据
	for i, ohlcv := range ohlcvs {
		if i >= 3 {
			break
		}
		fmt.Printf("K线 %d: 时间=%s, 开=%f, 高=%f, 低=%f, 收=%f, 量=%f\n",
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
