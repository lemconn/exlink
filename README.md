# ExLink - Go 加密货币交易所统一接口库

ExLink 是一个类似 Python ccxt 的 Go 语言加密货币交易所对接库，提供标准化的接口来访问多个交易所的 API。

## 特性

- 🎯 **统一接口**: 标准化的 API 接口，支持多个交易所
- 📊 **现货和合约**: 支持现货交易和永续合约
- 🔒 **类型安全**: 完整的类型定义，编译时检查
- 🚀 **易于扩展**: 简单的接口实现，轻松添加新交易所
- 📦 **模块化设计**: 清晰的代码结构，易于维护

## 支持的交易所

- ✅ Binance (现货)
- ✅ OKX (现货)

## 快速开始

### 安装

```bash
go get github.com/lemconn/exlink
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lemconn/exlink"
)

func main() {
    ctx := context.Background()
    
    // 创建交易所实例（不需要API密钥也可以获取公开数据）
    exchange, err := exlink.NewExchange("binance", "", "", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取行情（使用统一格式 BTC/USDT）
    ticker, err := exchange.FetchTicker(ctx, "BTC/USDT")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("BTC/USDT 价格: %.2f\n", ticker.Last)
    fmt.Printf("24h 涨跌幅: %.2f%%\n", ticker.ChangePercent)
}
```

### 使用 API 密钥

```go
// 创建带认证的交易所实例
exchange, err := exlink.NewExchange(
    "binance",
    "your-api-key",
    "your-secret-key",
    nil,
)
if err != nil {
    log.Fatal(err)
}

// 获取余额
balances, err := exchange.FetchBalance(ctx)
if err != nil {
    log.Fatal(err)
}

btcBalance := balances.GetBalance("BTC")
fmt.Printf("BTC 余额: %.8f\n", btcBalance.Free)
```

### 使用模拟盘（Sandbox）

```go
// 创建模拟盘交易所实例
exchange, err := exlink.NewExchange(
    "binance",
    "your-api-key",
    "your-secret-key",
    map[string]interface{}{
        "sandbox": true, // 启用模拟盘
    },
)
```

### 使用代理

```go
// 创建带代理的交易所实例
exchange, err := exlink.NewExchange(
    "binance",
    "your-api-key",
    "your-secret-key",
    map[string]interface{}{
        "proxy": "http://proxy.example.com:8080", // 设置代理
    },
)
```

### 统一交易对格式

所有交易所统一使用 `BASE/QUOTE` 格式（如 `BTC/USDT`），库会自动转换为各交易所的格式：

```go
// 使用统一格式，库会自动转换
ticker, err := exchange.FetchTicker(ctx, "BTC/USDT") // Binance会自动转换为BTCUSDT，OKX会自动转换为BTC-USDT

// 创建订单也使用统一格式
order, err := exchange.CreateOrder(
    ctx,
    "BTC/USDT", // 统一格式
    types.OrderSideBuy,
    types.OrderTypeLimit,
    0.001,
    50000,
    nil,
)
```

### 创建订单

```go
import "github.com/lemconn/exlink/types"

// 创建限价买单（使用统一格式 BTC/USDT）
order, err := exchange.CreateOrder(
    ctx,
    "BTC/USDT", // 统一格式，会自动转换
    types.OrderSideBuy,
    types.OrderTypeLimit,
    0.001,  // 数量
    50000,  // 价格
    nil,    // 额外参数
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("订单ID: %s\n", order.ID)
```

### 获取K线数据

```go
// 获取1小时K线数据（使用统一格式 BTC/USDT）
ohlcvs, err := exchange.FetchOHLCV(
    ctx,
    "BTC/USDT", // 统一格式
    "1h",
    time.Now().Add(-24 * time.Hour),
    100,
)
if err != nil {
    log.Fatal(err)
}

for _, ohlcv := range ohlcvs {
    fmt.Printf("时间: %s, 开盘: %.2f, 收盘: %.2f\n",
        ohlcv.Timestamp, ohlcv.Open, ohlcv.Close)
}
```

### 使用 OKX 交易所

```go
import (
    _ "github.com/lemconn/exlink/exchanges/okx" // 导入以注册OKX
)

// 创建OKX交易所实例（需要passphrase）
exchange, err := exlink.NewExchange(
    "okx",
    "your-api-key",
    "your-secret-key",
    map[string]interface{}{
        "passphrase": "your-passphrase", // OKX需要passphrase
        "sandbox":    true,              // 可选：使用模拟盘
        "proxy":      "http://proxy.example.com:8080", // 可选：使用代理
    },
)

// 使用统一格式调用
ticker, err := exchange.FetchTicker(ctx, "BTC/USDT")
```

## 核心概念

### 市场类型

- `MarketTypeSpot`: 现货市场
- `MarketTypeFuture`: 永续合约市场

### 订单类型

- `OrderTypeMarket`: 市价单
- `OrderTypeLimit`: 限价单

### 订单方向

- `OrderSideBuy`: 买入
- `OrderSideSell`: 卖出

### 订单状态

- `OrderStatusNew`: 新建
- `OrderStatusOpen`: 开放
- `OrderStatusFilled`: 完全成交
- `OrderStatusCanceled`: 已取消
- 等等...

## 项目结构

```
exlink/
├── types/              # 标准化数据类型
│   ├── market.go      # 市场信息
│   ├── order.go       # 订单信息
│   ├── balance.go     # 余额信息
│   ├── ticker.go     # 行情信息
│   ├── trade.go      # 交易记录
│   ├── ohlcv.go      # K线数据
│   └── position.go   # 持仓信息（合约）
├── exchanges/         # 交易所实现
│   └── binance/      # Binance 实现
├── common/           # 通用工具
│   ├── http.go      # HTTP 客户端
│   └── signature.go # 签名工具
├── exchange.go       # 交易所接口定义
├── registry.go       # 交易所注册机制
└── errors.go         # 错误定义
```

## 添加新交易所

要添加新的交易所支持，需要：

1. 在 `exchanges/` 目录下创建新的包
2. 实现 `Exchange` 接口
3. 在 `init()` 函数中注册交易所

示例：

```go
package myexchange

import "github.com/lemconn/exlink"

type MyExchange struct {
    *exlink.BaseExchange
    // ... 其他字段
}

func NewMyExchange(apiKey, secretKey string, options map[string]interface{}) (exlink.Exchange, error) {
    // ... 初始化逻辑
    return &MyExchange{
        BaseExchange: exlink.NewBaseExchange("myexchange"),
        // ...
    }, nil
}

func init() {
    exlink.Register("myexchange", NewMyExchange)
}
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
