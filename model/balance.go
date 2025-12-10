package model

import "github.com/lemconn/exlink/types"

// Balance 余额信息
type Balance struct {
	// Currency 币种
	Currency string `json:"currency"`
	// Available 可用余额
	Available types.ExDecimal `json:"available"`
	// Locked 冻结余额
	Locked types.ExDecimal `json:"locked"`
	// Total 总余额
	Total types.ExDecimal `json:"total"`
	// Timestamp 更新时间
	UpdatedAt types.ExTimestamp `json:"updated_at"`
}

// Balances 所有余额
type Balances []*Balance
