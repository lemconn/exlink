package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lemconn/exlink"
	"github.com/lemconn/exlink/types"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取 API 密钥（实际使用时请设置）
	apiKey := os.Getenv("BYBIT_API_KEY")
	secretKey := os.Getenv("BYBIT_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		log.Println("警告: 未设置 BYBIT_API_KEY 和 BYBIT_SECRET_KEY 环境变量")
		log.Println("示例代码将展示如何使用，但实际调用会失败")
	}

	// 创建 Bybit 交易所实例
	exchange, err := exlink.NewExchange(
		exlink.ExchangeBybit,
		exlink.WithAPIKey(apiKey),
		exlink.WithSecretKey(secretKey),
	)
	if err != nil {
		log.Fatalf("创建交易所实例失败: %v", err)
	}

	// 加载市场信息
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		log.Fatalf("加载市场信息失败: %v", err)
	}

	// 合约交易对（永续合约）
	contractSymbol := "BTC/USDT:USDT"
	// 现货交易对
	spotSymbol := "BTC/USDT"

	// 获取当前价格
	ticker, err := exchange.FetchTicker(ctx, contractSymbol)
	if err != nil {
		log.Fatalf("获取行情失败: %v", err)
	}
	fmt.Printf("当前价格: %.2f\n\n", ticker.Last)

	amount := 0.001 // 订单数量（根据实际情况调整）

	// ========== 合约交易示例 ==========
	fmt.Println("=== 合约交易示例 ===")

	// 1. 合约买入开多
	fmt.Println("\n1. 合约买入开多")
	params := map[string]interface{}{
		"reduceOnly": false, // Bybit 使用 reduceOnly=false 表示开仓
	}
	order, err := exchange.CreateOrder(ctx, contractSymbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		log.Printf("合约买入开多失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 2. 合约卖出平多
	fmt.Println("\n2. 合约卖出平多")
	params = map[string]interface{}{
		"reduceOnly": true, // Bybit 使用 reduceOnly=true 表示平仓
	}
	order, err = exchange.CreateOrder(ctx, contractSymbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		log.Printf("合约卖出平多失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 3. 合约卖出开空
	fmt.Println("\n3. 合约卖出开空")
	params = map[string]interface{}{
		"reduceOnly": false, // Bybit 使用 reduceOnly=false 表示开仓
	}
	order, err = exchange.CreateOrder(ctx, contractSymbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		log.Printf("合约卖出开空失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 4. 合约买入平空
	fmt.Println("\n4. 合约买入平空")
	params = map[string]interface{}{
		"reduceOnly": true, // Bybit 使用 reduceOnly=true 表示平仓
	}
	order, err = exchange.CreateOrder(ctx, contractSymbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, params)
	if err != nil {
		log.Printf("合约买入平空失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// ========== 现货交易示例 ==========
	fmt.Println("\n=== 现货交易示例 ===")

	// 5. 现货买入
	fmt.Println("\n5. 现货买入")
	order, err = exchange.CreateOrder(ctx, spotSymbol, types.OrderSideBuy, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		log.Printf("现货买入失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 6. 现货卖出
	fmt.Println("\n6. 现货卖出")
	order, err = exchange.CreateOrder(ctx, spotSymbol, types.OrderSideSell, types.OrderTypeMarket, amount, 0, nil)
	if err != nil {
		log.Printf("现货卖出失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Symbol=%s, Side=%s, Amount=%f\n",
			order.ID, order.Symbol, order.Side, order.Amount)
	}

	// 限价单示例
	fmt.Println("\n=== 限价单示例 ===")
	limitPrice := ticker.Bid * 0.99 // 买入限价单，低于当前价格 1%
	fmt.Printf("\n限价买入: 价格=%.2f, 数量=%f\n", limitPrice, amount)
	order, err = exchange.CreateOrder(ctx, spotSymbol, types.OrderSideBuy, types.OrderTypeLimit, amount, limitPrice, nil)
	if err != nil {
		log.Printf("限价买入失败: %v", err)
	} else {
		fmt.Printf("订单创建成功: ID=%s, Price=%.2f, Amount=%f\n",
			order.ID, order.Price, order.Amount)
	}
}

