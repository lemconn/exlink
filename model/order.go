package model

import "time"

// OrderSide 订单方向
type OrderSide string

const (
	// OrderSideBuy 买入
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell 卖出
	OrderSideSell OrderSide = "sell"
)

// OrderType 订单类型
type OrderType string

const (
	// OrderTypeMarket 市价单
	OrderTypeMarket OrderType = "market"
	// OrderTypeLimit 限价单
	OrderTypeLimit OrderType = "limit"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	// OrderStatusNew 新建订单
	OrderStatusNew OrderStatus = "new"
	// OrderStatusOpen 未成交订单
	OrderStatusOpen OrderStatus = "open"
	// OrderStatusClosed 已成交订单
	OrderStatusClosed OrderStatus = "closed"
	// OrderStatusCanceled 已取消订单
	OrderStatusCanceled OrderStatus = "canceled"
	// OrderStatusExpired 已过期订单
	OrderStatusExpired OrderStatus = "expired"
	// OrderStatusRejected 已拒绝订单
	OrderStatusRejected OrderStatus = "rejected"
)

// OrderTimeInForce 订单有效期
type OrderTimeInForce string

const (
	// OrderTimeInForceGTC 订单有效直到取消（Good Till Cancel）
	OrderTimeInForceGTC OrderTimeInForce = "GTC"
	// OrderTimeInForceIOC 立即成交或取消（Immediate Or Cancel）
	OrderTimeInForceIOC OrderTimeInForce = "IOC"
	// OrderTimeInForceFOK 全部成交或取消（Fill Or Kill）
	OrderTimeInForceFOK OrderTimeInForce = "FOK"
)

// PositionSide 持仓方向（用于合约）
type PositionSide string

const (
	// PositionSideLong 多头
	PositionSideLong PositionSide = "long"
	// PositionSideShort 空头
	PositionSideShort PositionSide = "short"
)

// Fee 手续费信息
type Fee struct {
	// Currency 手续费币种
	Currency string `json:"currency"`
	// Cost 手续费金额
	Cost float64 `json:"cost"`
	// Rate 手续费率
	Rate float64 `json:"rate,omitempty"`
}

// OrderOption 订单选项
type OrderOption struct {
	// Price 价格（限价单必填）
	Price *string `json:"price,omitempty"`
	// TimeInForce 订单有效期
	TimeInForce *OrderTimeInForce `json:"time_in_force,omitempty"`
	// ClientOrderID 客户端订单ID
	ClientOrderID *string `json:"client_order_id,omitempty"`
	// PositionSide 持仓方向（合约订单必填）
	PositionSide *PositionSide `json:"position_side,omitempty"`
	// ReduceOnly 是否只减仓（合约订单）
	ReduceOnly *bool `json:"reduce_only,omitempty"`
}

// Order 订单信息
type Order struct {
	// ID 订单ID
	ID string `json:"id"`
	// ClientOrderID 客户端订单ID
	ClientOrderID string `json:"client_order_id,omitempty"`
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Type 订单类型
	Type OrderType `json:"type"`
	// Side 订单方向
	Side OrderSide `json:"side"`
	// Amount 订单数量
	Amount float64 `json:"amount"`
	// Price 订单价格
	Price float64 `json:"price"`
	// Filled 已成交数量
	Filled float64 `json:"filled"`
	// Remaining 未成交数量
	Remaining float64 `json:"remaining"`
	// Cost 成交金额
	Cost float64 `json:"cost"`
	// Average 平均成交价格
	Average float64 `json:"average"`
	// Status 订单状态
	Status OrderStatus `json:"status"`
	// Fee 手续费
	Fee *Fee `json:"fee,omitempty"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}

