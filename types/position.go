package types

import "time"

// PositionSide 持仓方向
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"  // 多头
	PositionSideShort PositionSide = "short" // 空头
)

// Position 持仓信息（用于合约）
type Position struct {
	Symbol           string                 `json:"symbol"`            // 交易对
	Side             PositionSide           `json:"side"`              // 持仓方向
	Amount           float64                `json:"amount"`            // 持仓数量
	EntryPrice       float64                `json:"entry_price"`       // 开仓价格
	MarkPrice        float64                `json:"mark_price"`        // 标记价格
	LiquidationPrice float64                `json:"liquidation_price"` // 强平价格
	UnrealizedPnl    float64                `json:"unrealized_pnl"`    // 未实现盈亏
	RealizedPnl      float64                `json:"realized_pnl"`      // 已实现盈亏
	Leverage         float64                `json:"leverage"`          // 杠杆倍数
	Margin           float64                `json:"margin"`            // 保证金
	Percentage       float64                `json:"percentage"`        // 持仓占比
	Timestamp        time.Time              `json:"timestamp"`         // 时间戳
	Info             map[string]interface{} `json:"info"`              // 交易所原始信息
}
