package binance

import (
	"github.com/lemconn/exlink/types"
)

// binancePerpTickerResponse Binance 永续合约 Ticker 响应
type binancePerpTickerResponse struct {
	Symbol             string            `json:"symbol"`
	PriceChange        types.ExDecimal   `json:"priceChange"`
	PriceChangePercent types.ExDecimal   `json:"priceChangePercent"`
	WeightedAvgPrice   types.ExDecimal   `json:"weightedAvgPrice"`
	LastPrice          types.ExDecimal   `json:"lastPrice"`
	LastQty            types.ExDecimal   `json:"lastQty"`
	OpenPrice          types.ExDecimal   `json:"openPrice"`
	HighPrice          types.ExDecimal   `json:"highPrice"`
	LowPrice           types.ExDecimal   `json:"lowPrice"`
	Volume             types.ExDecimal   `json:"volume"`
	QuoteVolume        types.ExDecimal   `json:"quoteVolume"`
	OpenTime           types.ExTimestamp `json:"openTime"`
	CloseTime          types.ExTimestamp `json:"closeTime"`
	FirstId            int64             `json:"firstId"`
	LastId             int64             `json:"lastId"`
	Count              int64             `json:"count"`
}
