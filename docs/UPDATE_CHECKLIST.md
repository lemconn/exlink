# 需要更新的文件清单

## 一、README.md 需要更新的位置

### 1. CreateOrder 示例（第179行）
**当前代码：**
```go
order, err := spot.CreateOrder(ctx, "BTC/USDT", types.OrderSideBuy, "0.001", types.WithPrice("50000"))
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

order, err := spot.CreateOrder(ctx, "BTC/USDT", model.OrderSideBuy,
    option.WithPrice("50000"),
    option.WithAmount("0.001"),
)
```

**注意：** 
- 需要导入 `option` 包
- Spot 的 CreateOrder 使用 `model.OrderSide`，不是 `types.OrderSide`
- 需要将 amount 作为参数传入，或使用 `option.WithAmount()`

### 2. FetchTrades 示例（第211行）
**当前代码：**
```go
trades, err := spot.FetchTrades(ctx, "BTC/USDT", time.Time{}, 100)
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

trades, err := spot.FetchTrades(ctx, "BTC/USDT",
    option.WithLimit(100),
    option.WithSince(time.Time{}),
)
```

### 3. FetchMyTrades 示例（第217行）
**当前代码：**
```go
myTrades, err := spot.FetchMyTrades(ctx, "BTC/USDT", time.Time{}, 100)
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

myTrades, err := spot.FetchMyTrades(ctx, "BTC/USDT",
    option.WithLimit(100),
    option.WithSince(time.Time{}),
)
```

### 4. FetchPositions 示例（第232行）
**当前代码：**
```go
positions, err := perp.FetchPositions(ctx, "BTC/USDT:USDT")
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

positions, err := perp.FetchPositions(ctx,
    option.WithSymbols("BTC/USDT:USDT"),
)
```

## 二、docs/interface_design_proposal.md 需要更新的位置

### 1. SpotExchange 接口定义（第71行）
**当前代码：**
```go
FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error)
```

**需要更新为：**
```go
FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error)
```

**注意：** 方法名应该是 `FetchOHLCVs`（复数），返回类型是 `model.OHLCVs`

### 2. SpotExchange 接口定义（第81行）
**当前代码：**
```go
CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error)
```

**需要更新为：**
```go
CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...option.ArgsOption) (*model.Order, error)
```

**注意：** Spot 使用 `model.OrderSide` 和 `model.Order`，不需要 `amount` 参数（通过 option 传入）

### 3. SpotExchange 接口定义（第92-95行）
**当前代码：**
```go
FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
```

**需要更新为：**
```go
FetchTrades(ctx context.Context, symbol string, opts ...option.ArgsOption) ([]*types.Trade, error)
FetchMyTrades(ctx context.Context, symbol string, opts ...option.ArgsOption) ([]*types.Trade, error)
```

### 4. PerpExchange 接口定义（第121行）
**当前代码：**
```go
FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error)
```

**需要更新为：**
```go
FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error)
```

### 5. PerpExchange 接口定义（第126行）
**当前代码：**
```go
FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error)
```

**需要更新为：**
```go
FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error)
```

### 6. PerpExchange 接口定义（第131行）
**当前代码：**
```go
CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error)
```

**需要更新为：**
```go
CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...option.ArgsOption) (*types.Order, error)
```

### 7. PerpExchange 接口定义（第142-145行）
**当前代码：**
```go
FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)
```

**需要更新为：**
```go
FetchTrades(ctx context.Context, symbol string, opts ...option.ArgsOption) ([]*types.Trade, error)
FetchMyTrades(ctx context.Context, symbol string, opts ...option.ArgsOption) ([]*types.Trade, error)
```

### 8. 示例代码（第300-302行）
**当前代码：**
```go
order, err := spot.CreateOrder(ctx, "BTC/USDT", types.OrderSideBuy, "0.001",
    types.WithPrice("50000"),
)
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

order, err := spot.CreateOrder(ctx, "BTC/USDT", model.OrderSideBuy,
    option.WithPrice("50000"),
    option.WithAmount("0.001"),
)
```

### 9. 示例代码（第317-320行）
**当前代码：**
```go
order, err := perp.CreateOrder(ctx, "BTC/USDT:USDT", types.OrderSideBuy, "0.001",
    types.WithPrice("50000"),
    types.WithPositionSide(types.PositionSideLong),
)
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

order, err := perp.CreateOrder(ctx, "BTC/USDT:USDT", types.OrderSideBuy, "0.001",
    option.WithPrice("50000"),
    option.WithPositionSide("long"),
)
```

### 10. 示例代码（第323行）
**当前代码：**
```go
positions, err := perp.FetchPositions(ctx, "BTC/USDT:USDT")
```

**需要更新为：**
```go
import "github.com/lemconn/exlink/option"

positions, err := perp.FetchPositions(ctx,
    option.WithSymbols("BTC/USDT:USDT"),
)
```

### 11. 设计说明（第182-183行）
**当前代码：**
```go
- `FetchOrderBook` 的 `limit` 参数使用可变参数 `...int`，方便调用
- `CreateOrder` 使用 `...types.OrderOption` 处理可选参数（价格、时间等）
```

**需要更新为：**
```go
- `FetchOrderBook` 的 `limit` 参数使用 `option.ArgsOption`，方便调用
- `CreateOrder` 使用 `...option.ArgsOption` 处理可选参数（价格、时间等）
```

## 三、REFACTOR_EVALUATION.md 需要更新的位置

### 1. 所有 `exlink.ArgsOption` 引用
**需要更新为：** `option.ArgsOption`

### 2. 所有 `exlink.With*` 引用
**需要更新为：** `option.With*`

### 3. 所有 `exlink.ExchangeArgsOptions` 引用
**需要更新为：** `option.ExchangeArgsOptions`

**具体位置：**
- 第8-18行：方法签名变更说明
- 第88-209行：所有示例代码
- 第218行：实现细节说明

## 四、测试文件检查

所有测试文件已经更新完成 ✅

检查结果：
- `binance/binance_spot_test.go` ✅
- `binance/binance_perp_test.go` ✅
- `bybit/bybit_spot_test.go` ✅
- `bybit/bybit_perp_test.go` ✅
- `okx/okx_spot_test.go` ✅
- `okx/okx_perp_test.go` ✅
- `gate/gate_spot_test.go` ✅
- `gate/gate_perp_test.go` ✅

## 五、示例文件检查

所有示例文件已经更新完成 ✅

检查结果：
- `examples/spot/main.go` ✅
- `examples/perp/main.go` ✅

## 六、总结

### 需要更新的文件（3个）

1. **README.md** - 4处需要更新
   - CreateOrder 示例（第179行）
   - FetchTrades 示例（第211行）
   - FetchMyTrades 示例（第217行）
   - FetchPositions 示例（第232行）

2. **docs/interface_design_proposal.md** - 11处需要更新
   - 接口定义（7处）
   - 示例代码（3处）
   - 设计说明（1处）

3. **REFACTOR_EVALUATION.md** - 多处需要更新
   - 所有 `exlink.ArgsOption` → `option.ArgsOption`
   - 所有 `exlink.With*` → `option.With*`
   - 所有 `exlink.ExchangeArgsOptions` → `option.ExchangeArgsOptions`

### 已完成的文件

- ✅ 所有测试文件（8个）
- ✅ 所有示例文件（2个）
