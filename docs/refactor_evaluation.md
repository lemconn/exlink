# 使用 ArgsOption 替换可变参数的变更评估报告

## 一、需要修改的方法签名

### 1. 接口定义层（2个文件）

#### `exchange/spot.go`
- `FetchOHLCVs`: `(ctx, symbol, timeframe, since, limit)` → `(ctx, symbol, timeframe, opts ...option.ArgsOption)`
- `FetchTrades`: `(ctx, symbol, since, limit)` → `(ctx, symbol, opts ...option.ArgsOption)`
- `FetchMyTrades`: `(ctx, symbol, since, limit)` → `(ctx, symbol, opts ...option.ArgsOption)`
- `CreateOrder`: `(ctx, symbol, side, opts ...model.OrderOption)` → `(ctx, symbol, side, opts ...option.ArgsOption)`

#### `exchange/perp.go`
- `FetchOHLCVs`: `(ctx, symbol, timeframe, since, limit)` → `(ctx, symbol, timeframe, opts ...option.ArgsOption)`
- `FetchTrades`: `(ctx, symbol, since, limit)` → `(ctx, symbol, opts ...option.ArgsOption)`
- `FetchMyTrades`: `(ctx, symbol, since, limit)` → `(ctx, symbol, opts ...option.ArgsOption)`
- `FetchPositions`: `(ctx, symbols ...string)` → `(ctx, opts ...option.ArgsOption)`
- `CreateOrder`: `(ctx, symbol, side, amount, opts ...types.OrderOption)` → `(ctx, symbol, side, amount, opts ...option.ArgsOption)`

### 2. 实现层（8个文件）

每个交易所的 Spot 和 Perp 实现都需要修改：

#### Binance
- `binance/binance_spot.go`: 4个方法（FetchOHLCVs, FetchTrades, FetchMyTrades, CreateOrder）
- `binance/binance_perp.go`: 5个方法（FetchOHLCVs, FetchTrades, FetchMyTrades, FetchPositions, CreateOrder）

#### Bybit
- `bybit/bybit_spot.go`: 4个方法
- `bybit/bybit_perp.go`: 5个方法

#### OKX
- `okx/okx_spot.go`: 4个方法
- `okx/okx_perp.go`: 5个方法

#### Gate
- `gate/gate_spot.go`: 4个方法
- `gate/gate_perp.go`: 5个方法

### 3. 内部实现层（每个实现文件内部）

每个实现文件内部还有对应的内部实现方法需要修改：
- `*SpotMarket.FetchOHLCVs` - 内部实现
- `*SpotOrder.FetchTrades` - 内部实现
- `*SpotOrder.FetchMyTrades` - 内部实现
- `*SpotOrder.CreateOrder` - 内部实现
- `*PerpMarket.FetchOHLCVs` - 内部实现
- `*PerpOrder.FetchTrades` - 内部实现
- `*PerpOrder.FetchMyTrades` - 内部实现
- `*PerpOrder.FetchPositions` - 内部实现
- `*PerpOrder.CreateOrder` - 内部实现

## 二、文件变更统计

### 核心接口文件（2个）
1. `exchange/spot.go` - 4个方法签名修改
2. `exchange/perp.go` - 5个方法签名修改

### 实现文件（8个）
1. `binance/binance_spot.go` - 4个接口方法 + 4个内部实现方法 = 8处修改
2. `binance/binance_perp.go` - 5个接口方法 + 5个内部实现方法 = 10处修改
3. `bybit/bybit_spot.go` - 4个接口方法 + 4个内部实现方法 = 8处修改
4. `bybit/bybit_perp.go` - 5个接口方法 + 5个内部实现方法 = 10处修改
5. `okx/okx_spot.go` - 4个接口方法 + 4个内部实现方法 = 8处修改
6. `okx/okx_perp.go` - 5个接口方法 + 5个内部实现方法 = 10处修改
7. `gate/gate_spot.go` - 4个接口方法 + 4个内部实现方法 = 8处修改
8. `gate/gate_perp.go` - 5个接口方法 + 5个内部实现方法 = 10处修改

### 测试文件（8个）
1. `binance/binance_spot_test.go` - 需要更新测试用例
2. `binance/binance_perp_test.go` - 需要更新测试用例
3. `bybit/bybit_spot_test.go` - 需要更新测试用例
4. `bybit/bybit_perp_test.go` - 需要更新测试用例
5. `okx/okx_spot_test.go` - 需要更新测试用例
6. `okx/okx_perp_test.go` - 需要更新测试用例
7. `gate/gate_spot_test.go` - 需要更新测试用例
8. `gate/gate_perp_test.go` - 需要更新测试用例

### 示例文件（2个）
1. `examples/spot/main.go` - 需要更新调用方式
2. `examples/perp/main.go` - 需要更新调用方式

## 三、具体变更内容

### 变更类型 1: FetchOHLCVs
**原签名:**
```go
FetchOHLCVs(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (model.OHLCVs, error)
```

**新签名:**
```go
FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error)
```

**调用方式变更:**
```go
// 原方式
ohlcvs, err := spot.FetchOHLCVs(ctx, "BTC/USDT", "1h", time.Time{}, 100)

// 新方式
ohlcvs, err := spot.FetchOHLCVs(ctx, "BTC/USDT", "1h",
    option.WithLimit(100),
    option.WithSince(time.Time{}),
)
```

### 变更类型 2: FetchTrades / FetchMyTrades
**原签名:**
```go
FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
```

**新签名:**
```go
FetchTrades(ctx context.Context, symbol string, opts ...option.ArgsOption) ([]*types.Trade, error)
```

**调用方式变更:**
```go
// 原方式
trades, err := spot.FetchTrades(ctx, "BTC/USDT", time.Time{}, 100)

// 新方式
trades, err := spot.FetchTrades(ctx, "BTC/USDT",
    option.WithLimit(100),
    option.WithSince(time.Time{}),
)
```

### 变更类型 3: FetchPositions
**原签名:**
```go
FetchPositions(ctx context.Context, symbols ...string) (model.Positions, error)
```

**新签名:**
```go
FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error)
```

**调用方式变更:**
```go
// 原方式
positions, err := perp.FetchPositions(ctx, "BTC/USDT:USDT", "ETH/USDT:USDT")

// 新方式
positions, err := perp.FetchPositions(ctx,
    option.WithSymbols("BTC/USDT:USDT", "ETH/USDT:USDT"),
)
```

### 变更类型 4: CreateOrder (Spot)
**原签名:**
```go
CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...model.OrderOption) (*model.Order, error)
```

**新签名:**
```go
CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...option.ArgsOption) (*model.Order, error)
```

**调用方式变更:**
```go
// 原方式
orderOpts := []model.OrderOption{
    model.WithPrice("50000"),
    model.WithAmount("0.1"),
    model.WithClientOrderID("my-order-123"),
}
order, err := spot.CreateOrder(ctx, "BTC/USDT", model.OrderSideBuy, orderOpts...)

// 新方式
order, err := spot.CreateOrder(ctx, "BTC/USDT", model.OrderSideBuy,
    option.WithPrice("50000"),
    option.WithAmount("0.1"),
    option.WithClientOrderID("my-order-123"),
)
```

### 变更类型 5: CreateOrder (Perp)
**原签名:**
```go
CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error)
```

**新签名:**
```go
CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...option.ArgsOption) (*types.Order, error)
```

**调用方式变更:**
```go
// 原方式
orderOpts := []types.OrderOption{
    types.WithPrice("50000"),
    types.WithPositionSide(types.PositionSideLong),
    types.WithClientOrderID("my-order-123"),
}
order, err := perp.CreateOrder(ctx, "BTC/USDT:USDT", types.OrderSideBuy, "0.1", orderOpts...)

// 新方式
order, err := perp.CreateOrder(ctx, "BTC/USDT:USDT", types.OrderSideBuy, "0.1",
    option.WithPrice("50000"),
    option.WithPositionSide("long"),
    option.WithClientOrderID("my-order-123"),
)
```

## 四、实现细节变更

### 在方法内部需要添加参数解析逻辑

每个方法内部需要添加类似这样的代码：

```go
func (s *BinanceSpot) FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error) {
    // 解析参数
    args := &option.ExchangeArgsOptions{}
    for _, opt := range opts {
        opt(args)
    }
    
    // 获取参数值（带默认值）
    limit := 100 // 默认值
    if args.Limit != nil {
        limit = *args.Limit
    }
    
    since := time.Time{} // 默认值
    if args.Since != nil {
        since = *args.Since
    }
    
    // 调用内部实现
    return s.market.FetchOHLCVs(ctx, symbol, timeframe, since, limit)
}
```

### CreateOrder 需要处理订单参数转换

由于原来使用 `model.OrderOption` 和 `types.OrderOption`，现在需要从 `ArgsOption` 中提取参数并转换为内部使用的格式。

## 五、变更统计汇总

| 类别 | 文件数 | 方法数（估算） | 说明 |
|------|--------|---------------|------|
| 接口定义 | 2 | 9 | exchange/spot.go (4) + exchange/perp.go (5) |
| 实现层接口方法 | 8 | 36 | 每个交易所 Spot(4) + Perp(5) |
| 实现层内部方法 | 8 | 36 | 对应的内部实现方法 |
| 测试文件 | 8 | ~24 | 每个测试文件约3个测试用例 |
| 示例文件 | 2 | 3 | examples/spot/main.go (2) + examples/perp/main.go (1) |
| **总计** | **28** | **~108** | |

## 六、风险评估

### 高风险点
1. **CreateOrder 参数转换**: 需要将 `ArgsOption` 中的参数转换为原来 `OrderOption` 的格式，确保兼容性
2. **类型转换**: `PositionSide` 从 `types.PositionSide` 类型改为 `string`，需要确保转换正确
3. **默认值处理**: 需要确保所有方法的默认值与原来一致

### 中风险点
1. **测试覆盖**: 需要更新所有测试用例，确保测试通过
2. **示例代码**: 需要更新示例代码，确保示例可运行

### 低风险点
1. **接口签名变更**: 由于是新版本，不需要考虑向后兼容
2. **参数解析逻辑**: 逻辑相对简单，主要是参数提取和默认值处理

## 七、实施建议

### 阶段一：接口定义更新
1. 更新 `exchange/spot.go`
2. 更新 `exchange/perp.go`

### 阶段二：实现层更新（按交易所逐个进行）
1. Binance (先验证一个交易所)
2. Bybit
3. OKX
4. Gate

### 阶段三：测试和示例更新
1. 更新所有测试文件
2. 更新示例代码

### 阶段四：验证
1. 运行所有测试
2. 验证示例代码可运行
3. 代码审查

## 八、注意事项

1. **导入包**: 所有实现文件需要导入 `exlink` 包以使用 `ArgsOption`
2. **参数转换**: CreateOrder 需要将 `ArgsOption` 转换为原来的 `OrderOption` 格式，或者直接修改内部实现使用 `ArgsOption`
3. **默认值**: 确保所有默认值与原来一致：
   - `Limit`: 100
   - `Since`: `time.Time{}`
   - `Symbols`: `[]string{}`
4. **类型转换**: `PositionSide` 从枚举类型改为字符串，需要确保转换正确
