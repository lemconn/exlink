package model

// Balance 余额信息
type Balance struct {
	// Currency 币种
	Currency string `json:"currency"`
	// Free 可用余额
	Free float64 `json:"free"`
	// Used 冻结余额
	Used float64 `json:"used"`
	// Total 总余额
	Total float64 `json:"total"`
	// Available 可用余额（同 Free）
	Available float64 `json:"available"`
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
		Free:      0,
		Used:      0,
		Total:     0,
		Available: 0,
	}
}

