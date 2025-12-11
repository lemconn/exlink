package model

import (
	"github.com/lemconn/exlink/types"
)

// Position 持仓信息（用于合约）
type Position struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Side 持仓方向
	Side string `json:"side"`
	// Amount 持仓数量
	Amount types.ExDecimal `json:"amount"`
	// EntryPrice 开仓价格
	EntryPrice types.ExDecimal `json:"entry_price"`
	// MarkPrice 标记价格
	MarkPrice types.ExDecimal `json:"mark_price"`
	// LiquidationPrice 强平价格
	LiquidationPrice types.ExDecimal `json:"liquidation_price"`
	// UnrealizedPnl 未实现盈亏
	UnrealizedPnl types.ExDecimal `json:"unrealized_pnl"`
	// RealizedPnl 已实现盈亏
	RealizedPnl types.ExDecimal `json:"realized_pnl"`
	// Leverage 杠杆倍数
	Leverage types.ExDecimal `json:"leverage"`
	// Margin 保证金
	Margin types.ExDecimal `json:"margin"`
	// Percentage 持仓占比
	Percentage types.ExDecimal `json:"percentage"`
	// Timestamp 时间戳
	Timestamp types.ExTimestamp `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}

// Positions 持仓列表
type Positions []*Position
