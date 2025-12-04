package model

import "github.com/shopspring/decimal"

// OrderBookEntry 订单簿条目
type OrderBookEntry struct {
	// Price 价格
	Price decimal.Decimal `json:"price"`
	// Amount 数量
	Amount decimal.Decimal `json:"amount"`
}

// OrderBook 订单簿
type OrderBook struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Bids 买单列表（价格从高到低）
	Bids []OrderBookEntry `json:"bids"`
	// Asks 卖单列表（价格从低到高）
	Asks []OrderBookEntry `json:"asks"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp"`
}
