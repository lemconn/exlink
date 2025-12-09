package model

import "github.com/lemconn/exlink/types"

// Ticker 行情信息
type Ticker struct {
	// Symbol 交易对
	Symbol string `json:"symbol"`
	// Bid 买一价
	Bid types.ExDecimal `json:"bid"`
	// Ask 卖一价
	Ask types.ExDecimal `json:"ask"`
	// Last 最新价
	Last types.ExDecimal `json:"last"`
	// Open 开盘价
	Open types.ExDecimal `json:"open"`
	// High 最高价
	High types.ExDecimal `json:"high"`
	// Low 最低价
	Low types.ExDecimal `json:"low"`
	// Volume 24小时成交量
	Volume types.ExDecimal `json:"volume"`
	// QuoteVolume 24小时成交额
	QuoteVolume types.ExDecimal `json:"quote_volume"`
	// Timestamp 时间戳
	Timestamp types.ExTimestamp `json:"timestamp"`
	// Info 交易所原始信息
	Info map[string]interface{} `json:"info,omitempty"`
}

// Tickers 行情信息数组
type Tickers []*Ticker
