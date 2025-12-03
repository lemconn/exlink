package exchange

// Exchange 顶层交易所接口
type Exchange interface {
	// Spot 获取现货交易接口
	Spot() SpotExchange

	// Perp 获取永续合约交易接口
	Perp() PerpExchange

	// Name 返回交易所名称
	Name() string
}
