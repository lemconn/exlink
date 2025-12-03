package types

import (
	"strings"
	"time"
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

// OrderTimeInForceType 时间有效性类型
type OrderTimeInForceType string

const (
	OrderTimeInForceGTC OrderTimeInForceType = "gtc"
	OrderTimeInForceIOC OrderTimeInForceType = "ioc"
	OrderTimeInForceFOK OrderTimeInForceType = "fok"
)

func (t OrderTimeInForceType) Upper() string {
	return strings.ToUpper(string(t))
}

func (t OrderTimeInForceType) Lower() string {
	return strings.ToLower(string(t))
}

func (t OrderTimeInForceType) IsGTC() bool {
	return t == OrderTimeInForceGTC
}

func (t OrderTimeInForceType) IsIOC() bool {
	return t == OrderTimeInForceIOC
}

func (t OrderTimeInForceType) IsFOK() bool {
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

// OrderOption 订单选项函数类型
type OrderOption func(*orderOptions)

// orderOptions 订单选项（简化版，仅保留必要的通用选项）
type orderOptions struct {
	ClientOrderID *string               // 客户端订单ID（所有交易所通用）
	Price         *string               // 订单价格（限价单，如果设置则为限价单，未设置则为市价单）
	PositionSide  *PositionSide         // 持仓方向 (long/short，合约订单)
	TimeInForce   *OrderTimeInForceType // 时间有效性（gtc/ioc/fok，所有交易所通用）
}

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

// WithPositionSide 设置持仓方向（合约订单: long/short）
func WithPositionSide(positionSide PositionSide) OrderOption {
	return func(opts *orderOptions) {
		opts.PositionSide = &positionSide
	}
}

// WithTimeInForce 设置时间有效性（gtc/ioc/fok，所有交易所通用）
func WithTimeInForce(timeInForce OrderTimeInForceType) OrderOption {
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
