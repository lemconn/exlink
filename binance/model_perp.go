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

// binancePerpPositionResponse Binance 永续合约持仓响应（数组格式）
type binancePerpPositionResponse []binancePerpPosition

// binancePerpPosition Binance 永续合约持仓信息
type binancePerpPosition struct {
	Symbol           string            `json:"symbol"`
	PositionAmt      types.ExDecimal   `json:"positionAmt"`
	EntryPrice       types.ExDecimal   `json:"entryPrice"`
	BreakEvenPrice   types.ExDecimal   `json:"breakEvenPrice"`
	MarkPrice        types.ExDecimal   `json:"markPrice"`
	UnRealizedProfit types.ExDecimal   `json:"unRealizedProfit"`
	LiquidationPrice types.ExDecimal   `json:"liquidationPrice"`
	Leverage         types.ExDecimal   `json:"leverage"`
	MaxNotionalValue types.ExDecimal   `json:"maxNotionalValue"`
	MarginType       string            `json:"marginType"`
	IsolatedMargin   types.ExDecimal   `json:"isolatedMargin"`
	IsAutoAddMargin  string            `json:"isAutoAddMargin"`
	PositionSide     string            `json:"positionSide"`
	Notional         types.ExDecimal   `json:"notional"`
	IsolatedWallet   types.ExDecimal   `json:"isolatedWallet"`
	UpdateTime       types.ExTimestamp `json:"updateTime"`
	Isolated         bool              `json:"isolated"`
	AdlQuantile      int               `json:"adlQuantile"`
}

// binancePerpCreateOrderRequest Binance 永续合约创建订单请求
type binancePerpCreateOrderRequest struct {
	NewClientOrderID string `json:"newClientOrderId,omitempty"` // 自定义 ID
	PositionSide     string `json:"positionSide,omitempty"`     // 单向持仓时不传，双向持仓时传 LONG / SHORT
	Price            string `json:"price,omitempty"`            // 限价单价格
	Quantity         string `json:"quantity,omitempty"`         // 数量
	Side             string `json:"side,omitempty"`             // BUY / SELL
	ReduceOnly       string `json:"reduceOnly,omitempty"`       // "true" / "false"
	Symbol           string `json:"symbol,omitempty"`           // 交易对
	TimeInForce      string `json:"timeInForce,omitempty"`      // 限价单不传（实际代码中限价单会传 GTC）
	Type             string `json:"type,omitempty"`             // LIMIT / MARKET
}

// binancePerpCreateOrderResponse Binance 永续合约创建订单响应
type binancePerpCreateOrderResponse struct {
	OrderID       string            `json:"orderId"`       // 系统订单号
	ClientOrderID string            `json:"clientOrderId"` // 客户端订单ID
	UpdateTime    types.ExTimestamp `json:"updateTime"`    // 更新时间（毫秒时间戳）
}

// binancePerpFetchOrderResponse Binance 永续合约查询订单响应
type binancePerpFetchOrderResponse struct {
	OrderID       int64             `json:"orderId"`       // 订单ID（交易所唯一）
	ClientOrderID string            `json:"clientOrderId"` // 客户端自定义订单ID
	Symbol        string            `json:"symbol"`        // 交易对 / 合约标的
	Price         types.ExDecimal   `json:"price"`         // 下单价格（市价单通常为0）
	AvgPrice      types.ExDecimal   `json:"avgPrice"`      // 成交均价
	OrigQty       types.ExDecimal   `json:"origQty"`       // 下单数量
	ExecutedQty   types.ExDecimal   `json:"executedQty"`   // 实际成交数量
	Status        string            `json:"status"`        // 订单状态
	TimeInForce   string            `json:"timeInForce"`   // 订单有效方式
	ReduceOnly    bool              `json:"reduceOnly"`    // 是否只减仓
	Time          types.ExTimestamp `json:"time"`          // 创建时间（毫秒）
	Type          string            `json:"type"`          // 订单类型
	Side          string            `json:"side"`          // 订单方向
	PositionSide  string            `json:"positionSide"`  // 单向持仓 BOTH，双向持仓 LONG / SHORT
	UpdateTime    types.ExTimestamp `json:"updateTime"`    // 更新时间（毫秒）
}
