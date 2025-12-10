package gate

import (
	"github.com/lemconn/exlink/types"
)

// gatePerpMarketsResponse Gate 永续合约市场信息响应
type gatePerpMarketsResponse []gatePerpContract

// gatePerpContract Gate 永续合约信息
type gatePerpContract struct {
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	QuantoMultiplier string          `json:"quanto_multiplier"`
	OrderPriceRound  types.ExDecimal `json:"order_price_round"`
	OrderSizeMin     int             `json:"order_size_min"`
	OrderSizeMax     int             `json:"order_size_max"`
	InDelisting      bool            `json:"in_delisting"`
}

// gatePerpTickerResponse Gate 永续合约 Ticker 响应
type gatePerpTickerResponse []gatePerpTickerItem

// gatePerpTickerItem Gate 永续合约 Ticker 数据项
type gatePerpTickerItem struct {
	Last                  types.ExDecimal `json:"last"`
	Low24h                types.ExDecimal `json:"low_24h"`
	High24h               types.ExDecimal `json:"high_24h"`
	Volume24h             types.ExDecimal `json:"volume_24h"`
	ChangePercentage      types.ExDecimal `json:"change_percentage"`
	ChangePrice           types.ExDecimal `json:"change_price"`
	FundingRateIndicative types.ExDecimal `json:"funding_rate_indicative"`
	IndexPrice            types.ExDecimal `json:"index_price"`
	Volume24hBase         types.ExDecimal `json:"volume_24h_base"`
	Volume24hQuote        types.ExDecimal `json:"volume_24h_quote"`
	Contract              string          `json:"contract"`
	Volume24hSettle       types.ExDecimal `json:"volume_24h_settle"`
	FundingRate           types.ExDecimal `json:"funding_rate"`
	MarkPrice             types.ExDecimal `json:"mark_price"`
	TotalSize             types.ExDecimal `json:"total_size"`
	HighestBid            types.ExDecimal `json:"highest_bid"`
	HighestSize           types.ExDecimal `json:"highest_size"`
	LowestAsk             types.ExDecimal `json:"lowest_ask"`
	LowestSize            types.ExDecimal `json:"lowest_size"`
	QuantoMultiplier      types.ExDecimal `json:"quanto_multiplier"`
}

// gatePerpKlineResponse Gate 永续合约 Kline 响应（数组格式）
type gatePerpKlineResponse []gatePerpKline

// gatePerpKline Gate 永续合约 Kline 数据
type gatePerpKline struct {
	Open   types.ExDecimal   `json:"o"`   // Open price
	Volume int64             `json:"v"`   // Size volume (contract size)
	Time   types.ExTimestamp `json:"t"`   // Open time
	Close  types.ExDecimal   `json:"c"`   // Close price
	Low    types.ExDecimal   `json:"l"`   // Lowest price
	High   types.ExDecimal   `json:"h"`   // Highest price
	Sum    types.ExDecimal   `json:"sum"` // Trading volume (unit: Quote currency)
}
