# 交易所 Model 组织设计方案

## 问题分析

当前每个交易所目录（如 `binance/`）中的 `model.go` 文件存在以下问题：
1. 现货和合约的模型定义混在一起，文件较长且难以维护
2. 如果分开定义（如 `model_spot.go` 和 `model_perp.go`），共享模型（如 `binanceFilter`, `binanceKline`）会产生循环引用或重复定义的问题

## 设计方案

### 方案一：按功能域分离 + 共享模型独立文件（推荐）⭐

**结构：**
```
binance/
  ├── model_common.go    # 共享模型（Filter, Kline 等）
  ├── model_spot.go      # 现货专用模型
  └── model_perp.go      # 合约专用模型
```

**优点：**
- 清晰分离：共享模型、现货模型、合约模型各司其职
- 避免循环引用：共享模型独立，现货和合约都可以引用
- 易于维护：每个文件职责单一，代码量适中
- 符合 Go 包设计原则：同一包内多个文件，编译器会自动合并

**实现示例：**
```go
// model_common.go
package binance

// binanceFilter Binance 过滤器（现货和合约共用）
type binanceFilter struct {
    // ...
}

// binanceKline Binance Kline 数据（现货和合约共用）
type binanceKline struct {
    // ...
}

// model_spot.go
package binance

// binanceSpotSymbol Binance 现货交易对信息
type binanceSpotSymbol struct {
    Filters []binanceFilter `json:"filters"`  // 引用共享模型
    // ...
}

// model_perp.go
package binance

// binancePerpSymbol Binance 永续合约交易对信息
type binancePerpSymbol struct {
    Filters []binanceFilter `json:"filters"`  // 引用共享模型
    // ...
}
```

---

### 方案二：按领域模型分离

**结构：**
```
binance/
  ├── model_market.go    # 市场相关模型（Markets, Symbols）
  ├── model_ticker.go    # Ticker 相关模型
  ├── model_kline.go     # Kline 相关模型
  ├── model_balance.go   # 余额相关模型
  └── model_order.go     # 订单相关模型（如果有）
```

**优点：**
- 按业务领域组织，更符合领域驱动设计
- 相关模型集中，便于查找和理解

**缺点：**
- 现货和合约的同类模型仍可能混在一起
- 需要额外的命名约定来区分现货和合约（如 `binanceSpotTicker`, `binancePerpTicker`）

---

### 方案三：按响应类型分离

**结构：**
```
binance/
  ├── model_response.go  # 所有响应模型（Spot 和 Perp）
  ├── model_request.go   # 所有请求模型（如果有）
  └── model_common.go    # 共享的基础模型
```

**优点：**
- 按数据流向组织（请求/响应）
- 适合 RESTful API 场景

**缺点：**
- 现货和合约模型仍然混在一起
- 文件可能仍然很大

---

### 方案四：使用内部子包（不推荐）

**结构：**
```
binance/
  ├── model/
  │   ├── common.go
  │   ├── spot.go
  │   └── perp.go
  └── binance_spot.go
```

**缺点：**
- 增加了包层级，导入路径变长
- 对于同一交易所的实现，不需要额外的包隔离
- Go 的包设计原则：同一功能域应该在同一包内

---

## 推荐方案：方案一

### 实施步骤

1. **创建 `model_common.go`**
   - 移动所有共享模型（如 `binanceFilter`, `binanceKline`）
   - 包含共享的 UnmarshalJSON 方法

2. **创建 `model_spot.go`**
   - 移动所有现货专用模型
   - 保留对共享模型的引用

3. **创建 `model_perp.go`**
   - 移动所有合约专用模型
   - 保留对共享模型的引用

4. **删除原 `model.go`**

### 命名约定

- 共享模型：使用通用前缀（如 `binanceFilter`, `binanceKline`）
- 现货模型：使用 `Spot` 后缀（如 `binanceSpotSymbol`, `binanceSpotTickerResponse`）
- 合约模型：使用 `Perp` 后缀（如 `binancePerpSymbol`, `binancePerpTickerResponse`）

### 文件大小建议

- `model_common.go`: 共享模型，通常较小（< 300 行）
- `model_spot.go`: 现货模型，中等大小（300-500 行）
- `model_perp.go`: 合约模型，中等大小（300-500 行）

如果某个文件仍然很大（> 800 行），可以考虑进一步拆分：
- 按功能拆分：`model_spot_market.go`, `model_spot_order.go` 等
- 按响应类型拆分：`model_spot_response.go`, `model_spot_request.go` 等

---

## 其他考虑

### 1. 类型别名处理

如果使用类型别名（如 `type binanceSpotKline = binanceKline`），建议：
- 将别名定义在使用它的文件中（`model_spot.go` 或 `model_perp.go`）
- 或者统一放在 `model_common.go` 中，便于查找

### 2. 方法定义

如果模型有方法（如 `UnmarshalJSON`），建议：
- 共享模型的方法放在 `model_common.go`
- 特定模型的方法放在对应的文件中

### 3. 向后兼容

重构时注意：
- 保持所有导出的类型和函数不变
- 只改变内部文件组织
- 确保测试通过

---

## 总结

**最佳实践：**
- ✅ 使用方案一：按功能域分离 + 共享模型独立文件
- ✅ 清晰的命名约定区分现货和合约
- ✅ 保持文件大小适中（< 800 行）
- ✅ 同一包内多个文件，利用 Go 的包特性

**避免：**
- ❌ 创建不必要的子包
- ❌ 文件过大（> 1000 行）
- ❌ 循环引用
- ❌ 重复定义共享模型
