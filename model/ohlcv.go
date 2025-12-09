package model

import (
	"github.com/lemconn/exlink/types"
)

// OHLCV K线数据
type OHLCV struct {
	// Timestamp 时间戳
	Timestamp types.ExTimestamp `json:"timestamp"`
	// Open 开盘价
	Open types.ExDecimal `json:"open"`
	// High 最高价
	High types.ExDecimal `json:"high"`
	// Low 最低价
	Low types.ExDecimal `json:"low"`
	// Close 收盘价
	Close types.ExDecimal `json:"close"`
	// Volume 成交量
	Volume types.ExDecimal `json:"volume"`
}

// OHLCVs K线数据数组
type OHLCVs []*OHLCV
