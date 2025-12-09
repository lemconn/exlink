package binance

import "github.com/lemconn/exlink/types"

// ========== 现货市场模型 ==========

// binanceSpotMarketsResponse Binance 现货市场信息响应
type binanceSpotMarketsResponse struct {
	Symbols []binanceSpotSymbol `json:"symbols"`
}

// binanceSpotSymbol Binance 现货交易对信息
type binanceSpotSymbol struct {
	Symbol             string          `json:"symbol"`
	BaseAsset          string          `json:"baseAsset"`
	QuoteAsset         string          `json:"quoteAsset"`
	Status             string          `json:"status"`
	Filters            []binanceFilter `json:"filters"`
	BaseAssetPrecision int             `json:"baseAssetPrecision"`
	QuotePrecision     int             `json:"quotePrecision"`
}

// binanceFilter Binance 过滤器（现货和合约共用）
type binanceFilter struct {
	FilterType  string          `json:"filterType"`
	MinQty      types.ExDecimal `json:"minQty,omitempty"`
	MaxQty      types.ExDecimal `json:"maxQty,omitempty"`
	StepSize    types.ExDecimal `json:"stepSize,omitempty"`
	MinPrice    types.ExDecimal `json:"minPrice,omitempty"`
	MaxPrice    types.ExDecimal `json:"maxPrice,omitempty"`
	TickSize    types.ExDecimal `json:"tickSize,omitempty"`
	MinNotional types.ExDecimal `json:"minNotional,omitempty"`
}

// binanceSpotTickerResponse Binance 现货 Ticker 响应
type binanceSpotTickerResponse struct {
	Symbol             string          `json:"symbol"`
	PriceChange        types.ExDecimal `json:"priceChange"`
	PriceChangePercent types.ExDecimal `json:"priceChangePercent"`
	WeightedAvgPrice   types.ExDecimal `json:"weightedAvgPrice"`
	PrevClosePrice     types.ExDecimal `json:"prevClosePrice"`
	LastPrice          types.ExDecimal `json:"lastPrice"`
	LastQty            types.ExDecimal `json:"lastQty"`
	BidPrice           types.ExDecimal `json:"bidPrice"`
	BidQty             types.ExDecimal `json:"bidQty"`
	AskPrice           types.ExDecimal `json:"askPrice"`
	AskQty             types.ExDecimal `json:"askQty"`
	OpenPrice          types.ExDecimal `json:"openPrice"`
	HighPrice          types.ExDecimal `json:"highPrice"`
	LowPrice           types.ExDecimal `json:"lowPrice"`
	Volume             types.ExDecimal `json:"volume"`
	QuoteVolume        types.ExDecimal `json:"quoteVolume"`
	OpenTime           types.ExTimestamp `json:"openTime"`
	CloseTime          types.ExTimestamp `json:"closeTime"`
	FirstId            int64           `json:"firstId"`
	LastId             int64           `json:"lastId"`
	Count              int64           `json:"count"`
}

// ========== 永续合约市场模型 ==========

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
	Symbol             string          `json:"symbol"`
	PriceChange        types.ExDecimal `json:"priceChange"`
	PriceChangePercent types.ExDecimal `json:"priceChangePercent"`
	WeightedAvgPrice   types.ExDecimal `json:"weightedAvgPrice"`
	LastPrice          types.ExDecimal `json:"lastPrice"`
	LastQty            types.ExDecimal `json:"lastQty"`
	OpenPrice          types.ExDecimal `json:"openPrice"`
	HighPrice          types.ExDecimal `json:"highPrice"`
	LowPrice           types.ExDecimal `json:"lowPrice"`
	Volume             types.ExDecimal `json:"volume"`
	QuoteVolume        types.ExDecimal `json:"quoteVolume"`
	OpenTime           types.ExTimestamp `json:"openTime"`
	CloseTime          types.ExTimestamp `json:"closeTime"`
	FirstId            int64           `json:"firstId"`
	LastId             int64           `json:"lastId"`
	Count              int64           `json:"count"`
}
