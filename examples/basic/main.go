package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lemconn/exlink"
)

func main() {
	ctx := context.Background()

	// 创建交易所实例（不需要API密钥也可以获取公开数据）
	exchange, err := exlink.NewExchange(exlink.ExchangeBinance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== 获取行情 ===")
	ticker, err := exchange.FetchTicker(ctx, "BTC/USDT") // 使用统一格式
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("交易对: %s\n", ticker.Symbol)
	fmt.Printf("最新价: %.2f\n", ticker.Last)
	fmt.Printf("买一价: %.2f\n", ticker.Bid)
	fmt.Printf("卖一价: %.2f\n", ticker.Ask)
	fmt.Printf("24h 最高: %.2f\n", ticker.High)
	fmt.Printf("24h 最低: %.2f\n", ticker.Low)
	fmt.Printf("24h 涨跌幅: %.2f%%\n", ticker.ChangePercent)
	fmt.Printf("24h 成交量: %.2f\n", ticker.Volume)

	fmt.Println("\n=== 获取K线数据 ===")
	ohlcvs, err := exchange.FetchOHLCV(
		ctx,
		"BTC/USDT", // 使用统一格式
		"1h",
		time.Now().Add(-24*time.Hour),
		10,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("获取到 %d 条K线数据:\n", len(ohlcvs))
	for i, ohlcv := range ohlcvs {
		if i >= 5 {
			break
		}
		fmt.Printf("时间: %s, 开盘: %.2f, 最高: %.2f, 最低: %.2f, 收盘: %.2f, 成交量: %.2f\n",
			ohlcv.Timestamp.Format("2006-01-02 15:04:05"),
			ohlcv.Open,
			ohlcv.High,
			ohlcv.Low,
			ohlcv.Close,
			ohlcv.Volume,
		)
	}

	fmt.Println("\n=== 获取市场列表 ===")
	markets, err := exchange.GetMarkets(ctx, "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("支持 %d 个交易对\n", len(markets))

	// 显示前5个交易对
	fmt.Println("前5个交易对:")
	for i, market := range markets {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s (%s/%s)\n", market.Symbol, market.Base, market.Quote)
	}
}
