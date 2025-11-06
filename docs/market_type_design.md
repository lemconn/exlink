# 市场类型设计说明

## CCXT 的市场类型区分机制

根据对 ccxt Python 实现的分析，ccxt 通过以下方式区分现货和永续合约：

### 1. 市场加载阶段

在 `fetch_markets()` 时，通过 `options['fetchMarkets']` 参数指定要加载的市场类型：

```python
defaultTypes = ['spot', 'linear', 'inverse']
fetchMarketsOptions = self.safe_dict(self.options, 'fetchMarkets')
if fetchMarketsOptions is not None:
    rawFetchMarkets = self.safe_list(fetchMarketsOptions, 'types', defaultTypes)
```

- `spot`: 现货市场
- `linear`: 线性永续合约（USDT保证金）
- `inverse`: 反向永续合约（币本位保证金）

### 2. 市场解析阶段

在 `parse_market()` 中，通过解析市场数据来确定市场类型：

```python
contractType = self.safe_string(market, 'contractType')
if (contractType == 'PERPETUAL') or (expiry == 4133404800000):
    swap = True  # 永续合约
elif expiry is not None:
    future = True  # 期货
else:
    spot = True  # 现货
```

### 3. Symbol 格式

ccxt 使用不同的 symbol 格式来区分市场类型：

- **现货**: `BTC/USDT`
- **永续合约**: `BTC/USDT:USDT` (swap, 线性)
- **期货**: `BTC/USDT:USDT-240329` (future with expiry)
- **期权**: `BTC/USDT:USDT-240329-50000-C` (option)

### 4. 查询时的判断

在查询时（如 `fetch_ticker`），通过 `market(symbol)` 获取市场信息，然后根据市场属性判断使用哪个 API：

```python
market = self.market(symbol)
if market['option']:
    response = self.eapiPublicGetTicker(...)  # 期权API
elif market['linear']:
    response = self.fapiPublicGetTicker24hr(...)  # 永续合约API (USDT保证金)
elif market['inverse']:
    response = self.dapiPublicGetTicker24hr(...)  # 反向合约API (币本位)
else:
    response = self.publicGetTicker24hr(...)  # 现货API
```

## 关键点

1. **不是通过参数区分**：ccxt 不是通过查询参数来区分市场类型，而是通过：
   - Symbol 的格式（是否包含 `:` 和结算货币）
   - 已加载的市场信息中的 `type`、`linear`、`inverse`、`swap` 等属性

2. **市场信息是关键**：每个市场在加载时就被标记了类型，查询时通过 `market(symbol)` 获取这些信息

3. **Symbol 格式统一**：ccxt 使用统一的 symbol 格式，通过格式本身就能区分市场类型

## 对我们的实现的建议

1. **扩展 Market 类型**：在 `types.Market` 中添加更多属性来标识市场类型
2. **Symbol 格式扩展**：支持合约市场的 symbol 格式（如 `BTC/USDT:USDT`）
3. **市场类型判断**：在查询时根据市场的 `Type` 字段判断使用哪个 API
4. **加载市场时指定类型**：在 `LoadMarkets` 时可以通过参数指定要加载的市场类型
