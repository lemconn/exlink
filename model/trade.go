package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade 交易记录
type Trade struct {
	// ID 交易ID
	ID string `json:"id"`
	// OrderID 订单ID
	OrderID string `json:"order_id"`
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Type 类型
	Type string `json:"type"`
	// Side 方向
	Side string `json:"side"`
	// Amount 数量
	Amount decimal.Decimal `json:"amount"`
	// Price 价格
	Price decimal.Decimal `json:"price"`
	// Cost 成交金额
	Cost decimal.Decimal `json:"cost"`
	// Fee 手续费
	Fee *Fee `json:"fee,omitempty"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}
