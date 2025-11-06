package types

import "time"

// Ticker 行情信息
type Ticker struct {
	Symbol    string    `json:"symbol"`     // 交易对
	Bid       float64   `json:"bid"`       // 买一价
	Ask       float64   `json:"ask"`       // 卖一价
	Last      float64   `json:"last"`      // 最新价
	Open      float64   `json:"open"`      // 开盘价
	High      float64   `json:"high"`      // 最高价
	Low       float64   `json:"low"`       // 最低价
	Volume    float64   `json:"volume"`    // 24小时成交量
	QuoteVolume float64 `json:"quote_volume"` // 24小时成交额
	Change    float64   `json:"change"`     // 涨跌额
	ChangePercent float64 `json:"change_percent"` // 涨跌幅
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Info      map[string]interface{} `json:"info"` // 交易所原始信息
}
