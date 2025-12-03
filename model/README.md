# Model 包

本目录包含 exlink 统一对外输出的模型定义。

这些模型是 exlink 库的公共 API 的一部分，用于：
- 统一不同交易所的数据格式
- 提供标准化的数据结构
- 作为库对外接口的类型定义

## 模型列表

- **Market** - 市场信息
- **Order** - 订单信息
- **Ticker** - 行情信息
- **Balance** - 余额信息
- **Position** - 持仓信息（合约）
- **Trade** - 交易记录
- **OHLCV** - K线数据

## 使用说明

所有模型都通过 `github.com/lemconn/exlink/model` 包导出，可以直接使用：

```go
import "github.com/lemconn/exlink/model"

// 使用模型
markets, err := exchange.Spot().FetchMarkets(ctx)
for _, market := range markets {
    // market 是 *model.Market 类型
}
```

