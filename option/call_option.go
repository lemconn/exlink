package option

import (
	"time"
)

// ExchangeArgsOptions 方法调用参数选项（用于 Exchange 方法调用）
type ExchangeArgsOptions struct {
	// ========== 通用查询参数 ==========
	// Limit 限制返回数量（默认值：100）
	Limit *int
	// Since 起始时间（默认值：time.Time{}，表示不限制）
	Since *time.Time
	// Symbols 交易对列表（用于 FetchPositions 等方法）
	Symbols []string

	// ========== 订单相关参数 ==========
	// OrderType 订单类型（MARKET/LIMIT）
	OrderType *OrderType
	// Price 订单价格（限价单必填，如果设置则为限价单，未设置则为市价单）
	Price *string
	// Amount 订单数量（表示购买交易对真实数量）
	Amount *string
	// ClientOrderID 客户端订单ID（所有交易所通用）
	ClientOrderID *string
	// TimeInForce 订单有效期（GTC/IOC/FOK，所有交易所通用）
	TimeInForce *TimeInForce
	// HedgeMode 是否为双向持仓模式（合约订单）
	HedgeMode *bool
}

// ArgsOption 方法调用参数选项函数类型
type ArgsOption func(*ExchangeArgsOptions)

// ========== 通用查询参数选项 ==========

// WithLimit 设置限制返回数量
func WithLimit(limit int) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.Limit = &limit
	}
}

// WithSince 设置起始时间
func WithSince(since time.Time) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.Since = &since
	}
}

// WithSymbols 设置交易对列表
func WithSymbols(symbols ...string) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.Symbols = symbols
	}
}

// ========== 订单相关参数选项 ==========

// WithOrderType 设置订单类型（MARKET/LIMIT）
func WithOrderType(orderType OrderType) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.OrderType = &orderType
	}
}

// WithPrice 设置订单价格（限价单必填，如果设置则为限价单，未设置则为市价单）
func WithPrice(price string) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.Price = &price
	}
}

// WithAmount 设置订单数量（表示购买交易对真实数量）
func WithAmount(amount string) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.Amount = &amount
	}
}

// WithClientOrderID 设置客户端订单ID（所有交易所通用）
func WithClientOrderID(clientOrderID string) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.ClientOrderID = &clientOrderID
	}
}

// WithTimeInForce 设置订单有效期（GTC/IOC/FOK，所有交易所通用）
func WithTimeInForce(timeInForce TimeInForce) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.TimeInForce = &timeInForce
	}
}

// WithHedgeMode 设置是否为双向持仓模式（合约订单）
func WithHedgeMode(hedgeMode bool) ArgsOption {
	return func(opts *ExchangeArgsOptions) {
		opts.HedgeMode = &hedgeMode
	}
}
