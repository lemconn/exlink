package model

import "github.com/shopspring/decimal"

// Balance 余额信息
type Balance struct {
	// Currency 币种
	Currency string `json:"currency"`
	// Free 可用余额
	Free decimal.Decimal `json:"free"`
	// Used 冻结余额
	Used decimal.Decimal `json:"used"`
	// Total 总余额
	Total decimal.Decimal `json:"total"`
	// Available 可用余额（同 Free）
	Available decimal.Decimal `json:"available"`
}

// Balances 所有余额（币种 -> 余额的映射）
type Balances map[string]*Balance

// GetBalance 获取指定币种余额
func (b Balances) GetBalance(currency string) *Balance {
	if balance, ok := b[currency]; ok {
		return balance
	}
	return &Balance{
		Currency:  currency,
		Free:      decimal.Zero,
		Used:      decimal.Zero,
		Total:     decimal.Zero,
		Available: decimal.Zero,
	}
}
