package binance

import (
	"github.com/lemconn/exlink/types"
)

// binancePerpMarketsResponse Binance 永续合约市场信息响应
type binancePerpMarketsResponse struct {
	Symbols []binancePerpSymbol `json:"symbols"`
}

// binancePerpSymbol Binance 永续合约交易对信息
type binancePerpSymbol struct {
	Symbol            string          `json:"symbol"`
	Pair              string          `json:"pair"`
	ContractType      string          `json:"contractType"`
	BaseAsset         string          `json:"baseAsset"`
	QuoteAsset        string          `json:"quoteAsset"`
	MarginAsset       string          `json:"marginAsset"`
	Status            string          `json:"status"`
	PricePrecision    int             `json:"pricePrecision"`
	QuantityPrecision int             `json:"quantityPrecision"`
	Filters           []binanceFilter `json:"filters"`
}

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

// binancePerpKline Binance 永续合约 Kline 数据（类型别名）
type binancePerpKline = binanceKline

// binancePerpKlineResponse Binance 永续合约 Kline 响应（数组格式）
type binancePerpKlineResponse []binancePerpKline
