package types

// MarketType 市场类型
type MarketType string

const (
	MarketTypeSpot   MarketType = "spot"   // 现货
	MarketTypeFuture MarketType = "future" // 永续合约
	MarketTypeSwap   MarketType = "swap"   // 合约（同义于future）
)

// Market 市场信息
type Market struct {
	ID        string     `json:"id"`                 // 市场ID，如 "BTC/USDT" 或 "BTCUSDT"
	Symbol    string     `json:"symbol"`             // 交易对符号（统一格式），如 "BTC/USDT" 或 "BTC/USDT:USDT"（永续合约）
	Base      string     `json:"base"`               // 基础货币，如 "BTC"
	Quote     string     `json:"quote"`              // 计价货币，如 "USDT"
	Settle    string     `json:"settle,omitempty"`   // 结算货币（合约市场），如 "USDT"
	Type      MarketType `json:"type"`               // 市场类型
	Active    bool       `json:"active"`             // 是否活跃
	Linear    bool       `json:"linear,omitempty"`   // 是否为线性合约（U本位）
	Inverse   bool       `json:"inverse,omitempty"`  // 是否为反向合约（币本位）
	Contract  bool       `json:"contract,omitempty"` // 是否为合约市场
	ContractMultiplier float64  `json:"contract_multiplier,omitempty"` // 合约乘数（1张合约等于多少个币），仅合约市场有效
	Precision struct {
		Amount int `json:"amount"` // 数量精度
		Price  int `json:"price"`  // 价格精度
	} `json:"precision"`
	Limits struct {
		Amount struct {
			Min float64 `json:"min"` // 最小数量
			Max float64 `json:"max"` // 最大数量
		} `json:"amount"`
		Price struct {
			Min float64 `json:"min"` // 最小价格
			Max float64 `json:"max"` // 最大价格
		} `json:"price"`
		Cost struct {
			Min float64 `json:"min"` // 最小成本
			Max float64 `json:"max"` // 最大成本
		} `json:"cost"`
	} `json:"limits"`
	Info map[string]interface{} `json:"info"` // 交易所原始信息
}
