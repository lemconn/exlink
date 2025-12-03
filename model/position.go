package model

import "time"

// Position 持仓信息（用于合约）
type Position struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Side 持仓方向
	Side PositionSide `json:"side"`
	// Amount 持仓数量
	Amount float64 `json:"amount"`
	// EntryPrice 开仓价格
	EntryPrice float64 `json:"entry_price"`
	// MarkPrice 标记价格
	MarkPrice float64 `json:"mark_price"`
	// LiquidationPrice 强平价格
	LiquidationPrice float64 `json:"liquidation_price"`
	// UnrealizedPnl 未实现盈亏
	UnrealizedPnl float64 `json:"unrealized_pnl"`
	// RealizedPnl 已实现盈亏
	RealizedPnl float64 `json:"realized_pnl"`
	// Leverage 杠杆倍数
	Leverage float64 `json:"leverage"`
	// Margin 保证金
	Margin float64 `json:"margin"`
	// Percentage 持仓占比
	Percentage float64 `json:"percentage"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}

