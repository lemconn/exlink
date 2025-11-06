package exlink

import "errors"

var (
	// ErrMarketNotFound 市场未找到
	ErrMarketNotFound = errors.New("market not found")
	// ErrOrderNotFound 订单未找到
	ErrOrderNotFound = errors.New("order not found")
	// ErrInsufficientBalance 余额不足
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidSymbol 无效的交易对
	ErrInvalidSymbol = errors.New("invalid symbol")
	// ErrInvalidOrderType 无效的订单类型
	ErrInvalidOrderType = errors.New("invalid order type")
	// ErrExchangeNotSupported 不支持的交易所
	ErrExchangeNotSupported = errors.New("exchange not supported")
	// ErrAuthenticationRequired 需要认证
	ErrAuthenticationRequired = errors.New("authentication required")
	// ErrRateLimitExceeded 请求频率超限
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
