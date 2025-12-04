package model

import "github.com/shopspring/decimal"

// MarketType 市场类型
type MarketType string

const (
	// MarketTypeSpot 现货市场
	MarketTypeSpot MarketType = "spot"
	// MarketTypeFuture 永续合约市场
	MarketTypeFuture MarketType = "future"
	// MarketTypeSwap 合约市场（同义于 MarketTypeFuture）
	MarketTypeSwap MarketType = "swap"
)

// Market 市场信息
type Market struct {
	// ID 市场ID，如 "BTC/USDT" 或 "BTCUSDT"
	ID string `json:"id"`

	// Symbol 交易对符号（统一格式），如 "BTC/USDT" 或 "BTC/USDT:USDT"（永续合约）
	Symbol string `json:"symbol"`

	// Base 基础货币，如 "BTC"
	Base string `json:"base"`

	// Quote 计价货币，如 "USDT"
	Quote string `json:"quote"`

	// Settle 结算货币（合约市场），如 "USDT"
	Settle string `json:"settle,omitempty"`

	// Type 市场类型
	Type MarketType `json:"type"`

	// Active 是否活跃
	Active bool `json:"active"`

	// Linear 是否为线性合约（U本位）
	Linear bool `json:"linear,omitempty"`

	// Inverse 是否为反向合约（币本位）
	Inverse bool `json:"inverse,omitempty"`

	// Contract 是否为合约市场
	Contract bool `json:"contract,omitempty"`

	// ContractValue 合约面值（每张合约等于多少个币），仅合约市场有效
	ContractValue string `json:"contract_value,omitempty"`

	// Precision 精度信息
	Precision struct {
		// Amount 数量精度
		Amount int `json:"amount"`
		// Price 价格精度
		Price int `json:"price"`
	} `json:"precision"`

	// Limits 限制信息
	Limits struct {
		// Amount 数量限制
		Amount struct {
			// Min 最小数量
			Min decimal.Decimal `json:"min"`
			// Max 最大数量
			Max decimal.Decimal `json:"max"`
		} `json:"amount"`
		// Price 价格限制
		Price struct {
			// Min 最小价格
			Min decimal.Decimal `json:"min"`
			// Max 最大价格
			Max decimal.Decimal `json:"max"`
		} `json:"price"`
		// Cost 成本限制
		Cost struct {
			// Min 最小成本
			Min decimal.Decimal `json:"min"`
			// Max 最大成本
			Max decimal.Decimal `json:"max"`
		} `json:"cost"`
	} `json:"limits"`

	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}
