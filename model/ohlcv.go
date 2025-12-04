package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// OHLCV K线数据
type OHLCV struct {
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Open 开盘价
	Open decimal.Decimal `json:"open"`
	// High 最高价
	High decimal.Decimal `json:"high"`
	// Low 最低价
	Low decimal.Decimal `json:"low"`
	// Close 收盘价
	Close decimal.Decimal `json:"close"`
	// Volume 成交量
	Volume decimal.Decimal `json:"volume"`
}

// OHLCVs K线数据数组
type OHLCVs []OHLCV
