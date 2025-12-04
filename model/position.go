package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Position 持仓信息（用于合约）
type Position struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Side 持仓方向
	Side PositionSide `json:"side"`
	// Amount 持仓数量
	Amount decimal.Decimal `json:"amount"`
	// EntryPrice 开仓价格
	EntryPrice decimal.Decimal `json:"entry_price"`
	// MarkPrice 标记价格
	MarkPrice decimal.Decimal `json:"mark_price"`
	// LiquidationPrice 强平价格
	LiquidationPrice decimal.Decimal `json:"liquidation_price"`
	// UnrealizedPnl 未实现盈亏
	UnrealizedPnl decimal.Decimal `json:"unrealized_pnl"`
	// RealizedPnl 已实现盈亏
	RealizedPnl decimal.Decimal `json:"realized_pnl"`
	// Leverage 杠杆倍数
	Leverage decimal.Decimal `json:"leverage"`
	// Margin 保证金
	Margin decimal.Decimal `json:"margin"`
	// Percentage 持仓占比
	Percentage decimal.Decimal `json:"percentage"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}
