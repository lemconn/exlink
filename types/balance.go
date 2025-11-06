package types

// Balance 余额信息
type Balance struct {
	Currency  string  `json:"currency"`  // 币种
	Free      float64 `json:"free"`      // 可用余额
	Used      float64 `json:"used"`      // 冻结余额
	Total     float64 `json:"total"`     // 总余额
	Available float64 `json:"available"` // 可用余额（同free）
}

// Balances 所有余额
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
