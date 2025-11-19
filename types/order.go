package types

import "time"

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"  // 买入
	OrderSideSell OrderSide = "sell" // 卖出
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeMarket OrderType = "market" // 市价单
	OrderTypeLimit  OrderType = "limit"  // 限价单
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"              // 新建
	OrderStatusOpen            OrderStatus = "open"             // 开放（部分成交）
	OrderStatusClosed          OrderStatus = "closed"           // 已关闭（完全成交）
	OrderStatusCanceled        OrderStatus = "canceled"         // 已取消
	OrderStatusExpired         OrderStatus = "expired"          // 已过期
	OrderStatusRejected        OrderStatus = "rejected"         // 已拒绝
	OrderStatusPartiallyFilled OrderStatus = "partially_filled" // 部分成交
	OrderStatusFilled          OrderStatus = "filled"           // 完全成交
)

// Order 订单信息
type Order struct {
	ID            string                 `json:"id"`              // 订单ID
	ClientOrderID string                 `json:"client_order_id"` // 客户端订单ID
	Symbol        string                 `json:"symbol"`          // 交易对
	Type          OrderType              `json:"type"`            // 订单类型
	Side          OrderSide              `json:"side"`            // 订单方向
	Amount        float64                `json:"amount"`          // 订单数量
	Price         float64                `json:"price"`           // 订单价格（限价单）
	Filled        float64                `json:"filled"`          // 已成交数量
	Remaining     float64                `json:"remaining"`       // 剩余数量
	Cost          float64                `json:"cost"`            // 成交金额
	Average       float64                `json:"average"`         // 平均成交价格
	Status        OrderStatus            `json:"status"`          // 订单状态
	Fee           *Fee                   `json:"fee,omitempty"`   // 手续费
	Timestamp     time.Time              `json:"timestamp"`       // 创建时间
	LastTradeTime time.Time              `json:"last_trade_time"` // 最后交易时间
	Info          map[string]interface{} `json:"info"`            // 交易所原始信息
}

// Fee 手续费
type Fee struct {
	Currency string  `json:"currency"` // 手续费币种
	Cost     float64 `json:"cost"`     // 手续费金额
	Rate     float64 `json:"rate"`     // 手续费率
}
