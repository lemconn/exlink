package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/option"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取 API 密钥
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

	// 创建 Binance 交易所实例
	opts := []option.Option{
		option.WithAPIKey(apiKey),
		option.WithSecretKey(secretKey),
		option.WithSandbox(true),
	}

	ex, err := exlink.NewExchange(exlink.ExchangeBinance, opts...)
	if err != nil {
		fmt.Printf("创建交易所实例失败: %v\n", err)
		return
	}

	// 获取现货接口
	spot := ex.Spot()

	// 加载市场信息
	if err := spot.LoadMarkets(ctx, false); err != nil {
		fmt.Printf("加载市场信息失败: %v\n", err)
		return
	}

	// 获取 OHLCV 数据
	symbol := "BTC/USDT"
	timeframe := "1h"
	ohlcvs, err := spot.FetchOHLCVs(ctx, symbol, timeframe,
		option.WithLimit(10),
		option.WithSince(time.Time{}),
	)
	if err != nil {
		fmt.Printf("获取 OHLCV 数据失败: %v\n", err)
		return
	}

	// 打印结果
	fmt.Printf("获取到 %d 条K线数据:\n", len(ohlcvs))
	for _, ohlcv := range ohlcvs {
		fmt.Printf("时间: %s, 开盘: %s, 最高: %s, 最低: %s, 收盘: %s, 成交量: %s\n",
			ohlcv.Timestamp.Format("2006-01-02 15:04:05"),
			ohlcv.Open.String(), ohlcv.High.String(), ohlcv.Low.String(), ohlcv.Close.String(), ohlcv.Volume.String())
	}

	// 创建订单
	symbol = "DOGE/USDT"
	side := model.OrderSideBuy
	order, err := spot.CreateOrder(ctx, symbol, side, "50",
		option.WithPrice("0.11"),
	)
	if err != nil {
		fmt.Printf("下单失败: %v\n", err)
		return
	}

	// 打印结果
	fmt.Printf("下单成功，打印结果:\n")
	fmt.Printf("ID:%s, 订单ID：%s, 交易对：%s, 订单类型：%s, 方向：%s, 数量：%s, 价格：%s, 状态：%s\n",
		order.ID, order.ClientOrderID, order.Symbol, order.Type, order.Side, order.Amount, order.Price, order.Status)

	// 取消订单
	err = spot.CancelOrder(ctx, order.ID, symbol)
	if err != nil {
		fmt.Printf("订单（%s）取消失败: %v\n", order.ID, err)
		return
	}

	// 打印结果
	fmt.Printf("订单取消成功: %s\n", order.ID)

}
