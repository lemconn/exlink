package types

import "time"

// OHLCV K线数据
type OHLCV struct {
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Open      float64   `json:"open"`      // 开盘价
	High      float64   `json:"high"`      // 最高价
	Low       float64   `json:"low"`       // 最低价
	Close     float64   `json:"close"`     // 收盘价
	Volume    float64   `json:"volume"`    // 成交量
}

// OHLCVs K线数据数组
type OHLCVs []OHLCV
