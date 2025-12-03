package model

import "time"

// OHLCV K线数据
type OHLCV struct {
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Open 开盘价
	Open float64 `json:"open"`
	// High 最高价
	High float64 `json:"high"`
	// Low 最低价
	Low float64 `json:"low"`
	// Close 收盘价
	Close float64 `json:"close"`
	// Volume 成交量
	Volume float64 `json:"volume"`
}

// OHLCVs K线数据数组
type OHLCVs []OHLCV

