package model

// OrderBookEntry 订单簿条目
type OrderBookEntry struct {
	// Price 价格
	Price float64 `json:"price"`
	// Amount 数量
	Amount float64 `json:"amount"`
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

