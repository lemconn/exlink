package model

import "time"

// Ticker 行情信息
type Ticker struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Bid 买一价
	Bid string `json:"bid"`
	// Ask 卖一价
	Ask string `json:"ask"`
	// Last 最新价
	Last string `json:"last"`
	// Open 开盘价
	Open string `json:"open"`
	// High 最高价
	High string `json:"high"`
	// Low 最低价
	Low string `json:"low"`
	// Volume 24小时成交量
	Volume string `json:"volume"`
	// QuoteVolume 24小时成交额
	QuoteVolume string `json:"quote_volume"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}
