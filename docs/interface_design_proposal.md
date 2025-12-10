# 接口设计提案

## 一、顶层接口设计评估

### ✅ 基本设计可行

你提出的顶层接口设计是**完全可行**的：

```go
type Exchange interface {
    Spot() SpotExchange
    Perp() PerpExchange
    Name() string
}
```

这个设计简洁明了，符合 Go 的接口设计原则。

## 二、完整接口设计建议

### 当前设计的问题

你提供的 `SpotExchange` 和 `PerpExchange` 接口**缺少很多重要方法**。根据当前 `base.Exchange` 接口，需要补充以下方法：

### 完整的接口设计

```go
package exchange

import (
    "context"
    "time"
    "github.com/lemconn/exlink/types"
)

// Exchange 顶层交易所接口
type Exchange interface {
    // 获取现货接口
    Spot() SpotExchange
    
    // 获取永续合约接口
    Perp() PerpExchange
    
    // 交易所名称
    Name() string
}

// SpotExchange 现货交易接口
type SpotExchange interface {
    // ========== 市场数据 ==========
    
    // 加载市场信息
    LoadMarkets(ctx context.Context, reload bool) error
    
    // 获取市场列表
    FetchMarkets(ctx context.Context) ([]*types.Market, error)
    
    // 获取单个市场信息
    GetMarket(symbol string) (*types.Market, error)
    
    // 获取行情（单个）
    FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error)
    
    // 批量获取行情
    FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error)
    
    // 获取订单簿
    FetchOrderBook(ctx context.Context, symbol string, limit ...int) (*types.OrderBook, error)
    
    // 获取K线数据
    FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error)
    
    // ========== 账户信息 ==========
    
    // 获取余额
    FetchBalance(ctx context.Context) (types.Balances, error)
    
    // ========== 订单操作 ==========
    
    // 创建订单
    CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error)
    
    // 取消订单
    CancelOrder(ctx context.Context, orderID, symbol string) error
    
    // 查询订单
    FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error)
    
    // ========== 交易记录 ==========
    
    // 获取交易记录（公共）
    FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
    
    // 获取我的交易记录
    FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
}

// PerpExchange 永续合约交易接口
type PerpExchange interface {
    // ========== 市场数据 ==========
    
    // 加载市场信息
    LoadMarkets(ctx context.Context, reload bool) error
    
    // 获取市场列表
    FetchMarkets(ctx context.Context) ([]*types.Market, error)
    
    // 获取单个市场信息
    GetMarket(symbol string) (*types.Market, error)
    
    // 获取行情（单个）
    FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error)
    
    // 批量获取行情
    FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error)
    
    // 获取订单簿
    FetchOrderBook(ctx context.Context, symbol string, limit ...int) (*types.OrderBook, error)
    
    // 获取K线数据
    FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error)
    
    // ========== 账户信息 ==========
    
    // 获取持仓
    FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error)
    
    // ========== 订单操作 ==========
    
    // 创建订单
    CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error)
    
    // 取消订单
    CancelOrder(ctx context.Context, orderID, symbol string) error
    
    // 查询订单
    FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error)
    
    // ========== 交易记录 ==========
    
    // 获取交易记录（公共）
    FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
    
    // 获取我的交易记录
    FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
    
    // ========== 合约特有功能 ==========
    
    // 设置杠杆
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    
    // 设置保证金模式（isolated/cross）
    SetMarginMode(ctx context.Context, symbol string, mode string) error
    
    // 设置双向持仓模式
    SetHedgeMode(hedgeMode bool)
    
    // 是否为双向持仓模式
    IsHedgeMode() bool
}
```

## 三、设计说明

### 1. 方法分组

接口方法按功能分组：
- **市场数据**：行情、订单簿、K线等
- **账户信息**：余额（现货）、持仓（合约）
- **订单操作**：创建、取消、查询订单
- **交易记录**：公共交易记录和我的交易记录
- **合约特有**：杠杆、保证金模式、双向持仓等

### 2. 方法签名保持一致

- 所有方法都使用 `context.Context` 作为第一个参数
- 错误处理统一使用 Go 的 `error` 接口
- 返回类型使用 `types` 包中定义的标准类型

### 3. 可选参数处理

- `FetchOrderBook` 的 `limit` 参数使用可变参数 `...int`，方便调用
- `CreateOrder` 使用 `...types.OrderOption` 处理可选参数（价格、时间等）

### 4. 向后兼容考虑

如果需要保持向后兼容，可以创建一个适配器：

```go
// LegacyExchange 适配器，保持向后兼容
type LegacyExchange struct {
    exchange Exchange
}

func (l *LegacyExchange) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
    // 根据 symbol 判断是现货还是合约
    market, err := l.exchange.Spot().GetMarket(symbol)
    if err == nil && market != nil {
        return l.exchange.Spot().CreateOrder(ctx, symbol, side, amount, opts...)
    }
    
    market, err = l.exchange.Perp().GetMarket(symbol)
    if err == nil && market != nil {
        return l.exchange.Perp().CreateOrder(ctx, symbol, side, amount, opts...)
    }
    
    return nil, fmt.Errorf("market not found: %s", symbol)
}
```

## 四、实现示例

### Binance 实现结构

```go
package binance

import (
    "github.com/lemconn/exlink/exchange"
    "github.com/lemconn/exlink/common"
)

// Binance 交易所实现
type Binance struct {
    *base.BaseExchange
    client     *common.HTTPClient  // 现货客户端
    fapiClient *common.HTTPClient  // 合约客户端
    apiKey     string
    secretKey  string
    
    spot *BinanceSpot  // 现货实现
    perp *BinancePerp  // 永续合约实现
}

// 实现 Exchange 接口
func (b *Binance) Spot() exchange.SpotExchange {
    return b.spot
}

func (b *Binance) Perp() exchange.PerpExchange {
    return b.perp
}

func (b *Binance) Name() string {
    return "binance"
}

// BinanceSpot 现货实现
type BinanceSpot struct {
    *Binance  // 嵌入 Binance 以访问共享资源
}

// 实现 SpotExchange 接口的所有方法
func (s *BinanceSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
    // 实现逻辑
}

func (s *BinanceSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
    // 使用 s.client 发送现货订单请求
}

// BinancePerp 永续合约实现
type BinancePerp struct {
    *Binance  // 嵌入 Binance 以访问共享资源
}

// 实现 PerpExchange 接口的所有方法
func (p *BinancePerp) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
    // 实现逻辑
}

func (p *BinancePerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
    // 使用 p.fapiClient 发送合约订单请求
}

func (p *BinancePerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
    // 实现逻辑
}
```

## 五、使用示例

```go
// 创建交易所
exchange, err := exlink.NewExchange(exlink.ExchangeBinance,
    exlink.WithAPIKey(apiKey),
    exlink.WithSecretKey(secretKey),
)
if err != nil {
    log.Fatal(err)
}

// 现货交易
spot := exchange.Spot()

// 获取现货市场
markets, err := spot.FetchMarkets(ctx)

// 创建现货订单
order, err := spot.CreateOrder(ctx, "BTC/USDT", types.OrderSideBuy, "0.001",
    types.WithPrice("50000"),
)

// 查询现货余额
balance, err := spot.FetchBalance(ctx)

// 永续合约交易
perp := exchange.Perp()

// 获取合约市场
markets, err := perp.FetchMarkets(ctx)

// 设置杠杆
err = perp.SetLeverage(ctx, "BTC/USDT:USDT", 10)

// 创建合约订单
order, err := perp.CreateOrder(ctx, "BTC/USDT:USDT", types.OrderSideBuy, "0.001",
    types.WithPrice("50000"),
    types.WithPositionSide(types.PositionSideLong),
)

// 查询持仓
positions, err := perp.FetchPositions(ctx, "BTC/USDT:USDT")
```

## 六、总结

### ✅ 你的设计方向正确

1. **顶层接口简洁**：`Exchange` 接口只包含 `Spot()`、`Perp()` 和 `Name()`，符合接口隔离原则
2. **职责分离清晰**：现货和永续合约完全分离
3. **易于扩展**：未来可以轻松添加其他市场类型（如交割合约）

### ⚠️ 需要补充的内容

1. **完整的方法列表**：需要包含所有市场数据、订单、交易记录等方法
2. **方法签名**：需要包含 `context.Context` 和正确的参数类型
3. **错误处理**：统一使用 `error` 接口

### 📝 建议

1. **分阶段实施**：先实现核心方法，再逐步补充
2. **保持兼容**：通过适配器模式保持向后兼容
3. **充分测试**：确保新接口的稳定性和正确性

