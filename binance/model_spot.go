package binance

import (
	"github.com/lemconn/exlink/types"
)

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

// binanceSpotTickerResponse Binance 现货 Ticker 响应
type binanceSpotTickerResponse struct {
	Symbol             string            `json:"symbol"`
	PriceChange        types.ExDecimal   `json:"priceChange"`
	PriceChangePercent types.ExDecimal   `json:"priceChangePercent"`
	WeightedAvgPrice   types.ExDecimal   `json:"weightedAvgPrice"`
	PrevClosePrice     types.ExDecimal   `json:"prevClosePrice"`
	LastPrice          types.ExDecimal   `json:"lastPrice"`
	LastQty            types.ExDecimal   `json:"lastQty"`
	BidPrice           types.ExDecimal   `json:"bidPrice"`
	BidQty             types.ExDecimal   `json:"bidQty"`
	AskPrice           types.ExDecimal   `json:"askPrice"`
	AskQty             types.ExDecimal   `json:"askQty"`
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

// binanceSpotKline Binance 现货 Kline 数据（类型别名）
type binanceSpotKline = binanceKline

// binanceSpotKlineResponse Binance 现货 Kline 响应（数组格式）
type binanceSpotKlineResponse []binanceSpotKline

// binanceSpotBalanceResponse Binance 现货余额响应
type binanceSpotBalanceResponse struct {
	UpdateTime  types.ExTimestamp        `json:"updateTime"`
	AccountType string                   `json:"accountType"`
	Balances    []binanceSpotBalanceItem `json:"balances"`
}

// binanceSpotBalanceItem Binance 现货余额项
type binanceSpotBalanceItem struct {
	Asset  string          `json:"asset"`
	Free   types.ExDecimal `json:"free"`
	Locked types.ExDecimal `json:"locked"`
}

// binanceSpotCreateOrderResponse Binance 现货订单响应
type binanceSpotCreateOrderResponse struct {
	Symbol        string            `json:"symbol"`
	OrderID       int64             `json:"orderId"`
	ClientOrderID string            `json:"clientOrderId"`
	Time          types.ExTimestamp `json:"time"`
}

// binanceSpotFetchOrderResponse Binance 现货查询订单响应
type binanceSpotFetchOrderResponse struct {
	Symbol              string            `json:"symbol"`              // 交易对
	OrderID             int64             `json:"orderId"`             // 订单ID
	OrderListID         int64             `json:"orderListId"`         // 订单列表ID
	ClientOrderID       string            `json:"clientOrderId"`       // 客户端订单ID
	Price               types.ExDecimal   `json:"price"`               // 订单价格
	OrigQty             types.ExDecimal   `json:"origQty"`             // 原始数量
	ExecutedQty         types.ExDecimal   `json:"executedQty"`         // 已成交数量
	CummulativeQuoteQty types.ExDecimal   `json:"cummulativeQuoteQty"` // 累计成交金额
	Status              string            `json:"status"`              // 订单状态
	TimeInForce         string            `json:"timeInForce"`         // 时间有效性
	Type                string            `json:"type"`                // 订单类型
	Side                string            `json:"side"`                // 订单方向
	StopPrice           types.ExDecimal   `json:"stopPrice"`           // 止损价格
	IcebergQty          types.ExDecimal   `json:"icebergQty"`          // 冰山订单数量
	Time                types.ExTimestamp `json:"time"`                // 订单创建时间
	UpdateTime          types.ExTimestamp `json:"updateTime"`          // 订单更新时间
	IsWorking           bool              `json:"isWorking"`           // 是否在工作
	WorkingTime         types.ExTimestamp `json:"workingTime"`         // 工作时间
	OrigQuoteOrderQty   types.ExDecimal   `json:"origQuoteOrderQty"`   // 原始报价订单数量
}
