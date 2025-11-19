package types

import "time"

// Ticker 行情信息
type Ticker struct {
	Symbol      string                 `json:"symbol"`       // 交易对
	Bid         string                 `json:"bid"`          // 买一价
	Ask         string                 `json:"ask"`          // 卖一价
	Last        string                 `json:"last"`         // 最新价
	Open        string                 `json:"open"`         // 开盘价
	High        string                 `json:"high"`         // 最高价
	Low         string                 `json:"low"`          // 最低价
	Volume      string                 `json:"volume"`       // 24小时成交量
	QuoteVolume string                 `json:"quote_volume"` // 24小时成交额
	Timestamp   time.Time              `json:"timestamp"`    // 时间戳
	Info        map[string]interface{} `json:"info"`         // 交易所原始信息
}
