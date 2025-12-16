package model

import (
	"strings"

	"github.com/lemconn/exlink/types"
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"  // 买入
	OrderSideSell OrderSide = "sell" // 卖出
)

func (s OrderSide) Upper() string {
	return strings.ToUpper(string(s))
}

func (s OrderSide) Lower() string {
	return strings.ToLower(string(s))
}

func (s OrderSide) IsBuy() bool {
	return s == OrderSideBuy
}

func (s OrderSide) IsSell() bool {
	return s == OrderSideSell
}

// OrderType 订单类型
type OrderType string

const (
	OrderTypeMarket OrderType = "market" // 市价单
	OrderTypeLimit  OrderType = "limit"  // 限价单
)

func (t OrderType) Upper() string {
	return strings.ToUpper(string(t))
}

func (t OrderType) Lower() string {
	return strings.ToLower(string(t))
}

func (t OrderType) Capitalize() string {
	s := strings.ToLower(string(t))
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (t OrderType) IsMarket() bool {
	return t == OrderTypeMarket
}

func (t OrderType) IsLimit() bool {
	return t == OrderTypeLimit
}

// OrderTimeInForce 时间有效性类型
type OrderTimeInForce string

const (
	OrderTimeInForceGTC OrderTimeInForce = "GTC" // OrderTimeInForceGTC 订单有效直到取消（Good Till Cancel）
	OrderTimeInForceIOC OrderTimeInForce = "IOC" // OrderTimeInForceIOC 立即成交或取消（Immediate Or Cancel）
	OrderTimeInForceFOK OrderTimeInForce = "FOK" // OrderTimeInForceFOK 全部成交或取消（Fill Or Kill）
)

func (t OrderTimeInForce) Upper() string {
	return strings.ToUpper(string(t))
}

func (t OrderTimeInForce) Lower() string {
	return strings.ToLower(string(t))
}

func (t OrderTimeInForce) IsGTC() bool {
	return t == OrderTimeInForceGTC
}

func (t OrderTimeInForce) IsIOC() bool {
	return t == OrderTimeInForceIOC
}

func (t OrderTimeInForce) IsFOK() bool {
	return t == OrderTimeInForceFOK
}

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

// PositionSide 持仓方向（用于合约）
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"  // PositionSideLong 多头
	PositionSideShort PositionSide = "short" // PositionSideShort 空头
)

// Fee 手续费信息
type Fee struct {
	Currency string          `json:"currency"`       // Currency 手续费币种
	Cost     types.ExDecimal `json:"cost"`           // Cost 手续费金额
	Rate     types.ExDecimal `json:"rate,omitempty"` // Rate 手续费率
}

// Order 订单信息
type Order struct {
	ID            string                 `json:"id"`                        // ID 订单ID
	ClientOrderID string                 `json:"client_order_id,omitempty"` // ClientOrderID 客户端订单ID
	Symbol        string                 `json:"symbol"`                    // Symbol 交易对
	Type          OrderType              `json:"type"`                      // Type 订单类型
	Side          OrderSide              `json:"side"`                      // Side 订单方向
	Amount        types.ExDecimal        `json:"amount"`                    // Amount 订单数量
	Size          types.ExDecimal        `json:"size"`                      // Size 张数数量（gate/okx交易所使用张数交易）
	Price         types.ExDecimal        `json:"price"`                     // Price 订单价格
	Filled        types.ExDecimal        `json:"filled"`                    // Filled 已成交数量
	Remaining     types.ExDecimal        `json:"remaining"`                 // Remaining 未成交数量
	Cost          types.ExDecimal        `json:"cost"`                      // Cost 成交金额
	Average       types.ExDecimal        `json:"average"`                   // Average 平均成交价格
	Status        OrderStatus            `json:"status"`                    // Status 订单状态
	Fee           *Fee                   `json:"fee,omitempty"`             // Fee 手续费
	Timestamp     types.ExTimestamp      `json:"timestamp"`                 // Timestamp 时间戳
	Info          map[string]interface{} `json:"info,omitempty"`            // Info 交易所原始信息
}

// orderOptions 订单选项
type orderOptions struct {
	Price         *string           `json:"price,omitempty"`           // Price 价格（限价单必填）
	Amount        *string           `json:"amount,omitempty"`          // Amount 购买数量
	Size          *string           `json:"size,omitempty"`            // Size 购买张数
	TimeInForce   *OrderTimeInForce `json:"time_in_force,omitempty"`   // TimeInForce 订单有效期
	ClientOrderID *string           `json:"client_order_id,omitempty"` // ClientOrderID 客户端订单ID
	PositionSide  *PositionSide     `json:"position_side,omitempty"`   // PositionSide 持仓方向（合约订单必填）
	ReduceOnly    *bool             `json:"reduce_only,omitempty"`     // ReduceOnly 是否只减仓（合约订单）
}

// OrderOption 订单选项函数类型
type OrderOption func(*orderOptions)

// WithClientOrderID 设置客户端订单ID（通用，所有交易所都支持）
func WithClientOrderID(id string) OrderOption {
	return func(opts *orderOptions) {
		opts.ClientOrderID = &id
	}
}

// WithPrice 设置订单价格（通用，如果设置则为限价单，未设置则为市价单）
func WithPrice(price string) OrderOption {
	return func(opts *orderOptions) {
		opts.Price = &price
	}
}

// WithAmount 设置订单购买数量（表示购买交易对真实数量）
func WithAmount(amount string) OrderOption {
	return func(opts *orderOptions) {
		opts.Amount = &amount
	}
}

// WithSize 设置订单购买张数（表示购买交易对张数数量，例如：Gate,OKX交易所使用张数）
func WithSize(size string) OrderOption {
	return func(opts *orderOptions) {
		opts.Size = &size
	}
}

// WithPositionSide 设置持仓方向（合约订单: long/short）
func WithPositionSide(positionSide PositionSide) OrderOption {
	return func(opts *orderOptions) {
		opts.PositionSide = &positionSide
	}
}

// WithTimeInForce 设置时间有效性（gtc/ioc/fok，所有交易所通用）
func WithTimeInForce(timeInForce OrderTimeInForce) OrderOption {
	return func(opts *orderOptions) {
		opts.TimeInForce = &timeInForce
	}
}

// ApplyOrderOptions 应用订单选项
func ApplyOrderOptions(opts ...OrderOption) *orderOptions {
	options := &orderOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type NewOrder struct {
	Symbol        string
	OrderId       string
	ClientOrderID string
	Timestamp     types.ExTimestamp
}

// PerpOrder 永续合约订单信息
type PerpOrder struct {
	ID               string            `json:"id"`                // ID 交易所订单唯一 ID
	ClientID         string            `json:"client_id"`         // ClientID 客户端自定义订单 ID
	Type             string            `json:"type"`              // Type 订单类型
	Side             string            `json:"side"`              // Side 订单方向
	PositionSide     string            `json:"position_side"`     // PositionSide 持仓方向
	Symbol           string            `json:"symbol"`            // Symbol 交易对 / 合约标的
	Price            types.ExDecimal   `json:"price"`             // Price 下单价格（市价单通常为 0 / 空）
	AvgPrice         types.ExDecimal   `json:"avg_price"`         // AvgPrice 实际成交均价
	Quantity         types.ExDecimal   `json:"quantity"`          // Quantity 下单数量
	ExecutedQuantity types.ExDecimal   `json:"executed_quantity"` // ExecutedQuantity 实际成交数量
	Status           string            `json:"status"`            // Status 订单最终状态
	TimeInForce      string            `json:"time_in_force"`     // TimeInForce 订单有效方式（GTC / IOC 等）
	ReduceOnly       bool              `json:"reduce_only"`       // ReduceOnly 是否只减仓
	CreateTime       types.ExTimestamp `json:"create_time"`       // CreateTime 订单创建时间
	UpdateTime       types.ExTimestamp `json:"update_time"`       // UpdateTime 订单更新时间
}
