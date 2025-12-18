package types

import "time"

// Trade 交易记录
type Trade struct {
	ID        string                 `json:"id"`        // 交易ID
	OrderID   string                 `json:"order_id"`  // 订单ID
	Symbol    string                 `json:"symbol"`    // 交易对
	Type      string                 `json:"type"`      // 类型
	Side      string                 `json:"side"`      // 方向
	Amount    float64                `json:"amount"`    // 数量
	Price     float64                `json:"price"`     // 价格
	Cost      float64                `json:"cost"`      // 成交金额
	Timestamp time.Time              `json:"timestamp"` // 时间戳
	Info      map[string]interface{} `json:"info"`      // 交易所原始信息
}
