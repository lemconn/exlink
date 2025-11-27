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

// orderOptions 订单选项
type orderOptions struct {
	// 通用选项
	ClientOrderID *string // 客户端订单ID（所有交易所通用）
	Price         *string // 订单价格（限价单，如果设置则为限价单，未设置则为市价单）

	// Binance 特定选项
	PositionSide *PositionSide // 持仓方向 (LONG/SHORT，仅 hedge mode)

	// Bybit 特定选项
	// （无特定选项）

	// OKX 特定选项
	TdMode  *string // 交易模式 (cash/cross/isolated)
	TgtCcy  *string // 目标货币 (base_ccy/quote_ccy，现货订单)
	PosSide *string // 持仓方向 (long/short，合约订单)

	// Gate 特定选项
	Size        *int64                // 合约数量（合约订单）
	TIF         *OrderTimeInForceType // 时间有效性（合约订单：gtc/ioc/fok）
	TimeInForce *OrderTimeInForceType // 时间有效性（现货订单：ioc/fok）
	Cost        *float64              // 成本（现货市价买入）

	// 通用选项（多个交易所使用）
	ReduceOnly *bool // 仅减仓（合约订单，Gate/Bybit 通用）
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

// WithPositionSide 设置持仓方向（Binance 合约订单，仅 hedge mode）
func WithPositionSide(positionSide PositionSide) OrderOption {
	return func(opts *orderOptions) {
		opts.PositionSide = &positionSide
	}
}

// WithTdMode 设置交易模式（OKX：cash/cross/isolated）
func WithTdMode(tdMode string) OrderOption {
	return func(opts *orderOptions) {
		opts.TdMode = &tdMode
	}
}

// WithTgtCcy 设置目标货币（OKX 现货订单：base_ccy/quote_ccy）
func WithTgtCcy(tgtCcy string) OrderOption {
	return func(opts *orderOptions) {
		opts.TgtCcy = &tgtCcy
	}
}

// WithPosSide 设置持仓方向（OKX 合约订单：long/short）
func WithPosSide(posSide string) OrderOption {
	return func(opts *orderOptions) {
		opts.PosSide = &posSide
	}
}

// WithSize 设置合约数量（Gate 合约订单）
func WithSize(size int64) OrderOption {
	return func(opts *orderOptions) {
		opts.Size = &size
	}
}

// WithReduceOnly 设置仅减仓（Gate 合约订单）
func WithReduceOnly(reduceOnly bool) OrderOption {
	return func(opts *orderOptions) {
		opts.ReduceOnly = &reduceOnly
	}
}

// WithTIF 设置时间有效性（Gate 合约订单：gtc/ioc/fok）
func WithTIF(tif OrderTimeInForceType) OrderOption {
	return func(opts *orderOptions) {
		opts.TIF = &tif
	}
}

// WithTimeInForce 设置时间有效性（Gate 现货订单：ioc/fok）
func WithTimeInForce(timeInForce OrderTimeInForceType) OrderOption {
	return func(opts *orderOptions) {
		opts.TimeInForce = &timeInForce
	}
}

// WithCost 设置成本（Gate 现货市价买入）
func WithCost(cost float64) OrderOption {
	return func(opts *orderOptions) {
		opts.Cost = &cost
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
