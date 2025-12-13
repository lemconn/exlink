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

// binanceSpotCreateOrderRequest Binance 现货创建订单请求
type binanceSpotCreateOrderRequest struct {
	Symbol           string `json:"symbol"`            // 交易对
	Side             string `json:"side"`              // 订单方向 BUY/SELL
	Type             string `json:"type"`              // 订单类型 LIMIT/MARKET
	Quantity         string `json:"quantity"`          // 数量
	Price            string `json:"price,omitempty"`   // 价格（限价单必填）
	TimeInForce      string `json:"timeInForce,omitempty"` // 有效期类型
	NewClientOrderId string `json:"newClientOrderId,omitempty"` // 客户端订单ID
	Timestamp        int64  `json:"timestamp"`         // 时间戳
}

// binanceSpotCreateOrderResponse Binance 现货创建订单响应
type binanceSpotCreateOrderResponse struct {
	Symbol                  string                       `json:"symbol"`
	OrderID                 int64                        `json:"orderId"`
	ClientOrderID           string                       `json:"clientOrderId"`
	TransactTime            types.ExTimestamp            `json:"transactTime"`
	Price                   types.ExDecimal              `json:"price"`
	OrigQty                 types.ExDecimal              `json:"origQty"`
	ExecutedQty             types.ExDecimal              `json:"executedQty"`
	CummulativeQuoteQty     types.ExDecimal              `json:"cummulativeQuoteQty"`
	Status                  string                       `json:"status"`
	TimeInForce             string                       `json:"timeInForce"`
	Type                    string                       `json:"type"`
	Side                    string                       `json:"side"`
	WorkingTime             types.ExTimestamp            `json:"workingTime"`
	Fills                   []binanceSpotCreateOrderFill `json:"fills"`
	OrigQuoteOrderQty       types.ExDecimal              `json:"origQuoteOrderQty,omitempty"`
}

// binanceSpotCreateOrderFill Binance 现货订单成交明细
type binanceSpotCreateOrderFill struct {
	Price           types.ExDecimal `json:"price"`
	Qty             types.ExDecimal `json:"qty"`
	Commission      types.ExDecimal `json:"commission"`
	CommissionAsset string          `json:"commissionAsset"`
	TradeID         int64           `json:"tradeId"`
}
