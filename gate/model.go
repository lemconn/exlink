package gate

import "github.com/lemconn/exlink/types"

// ========== 现货市场模型 ==========

// gateSpotMarketsResponse Gate 现货市场信息响应
type gateSpotMarketsResponse []gateSpotCurrencyPair

// gateSpotCurrencyPair Gate 现货交易对信息
type gateSpotCurrencyPair struct {
	ID              string          `json:"id"`
	Base            string          `json:"base"`
	Quote           string          `json:"quote"`
	Fee             string          `json:"fee"`
	MinBaseAmount   types.ExDecimal `json:"min_base_amount"`
	MinQuoteAmount  types.ExDecimal `json:"min_quote_amount"`
	MaxQuoteAmount  types.ExDecimal `json:"max_quote_amount"`
	AmountPrecision int             `json:"amount_precision"`
	Precision       int             `json:"precision"`
	TradeStatus     string          `json:"trade_status"`
}

// gateSpotTickerResponse Gate 现货 Ticker 响应
type gateSpotTickerResponse []gateSpotTickerItem

// gateSpotTickerItem Gate 现货 Ticker 数据项
type gateSpotTickerItem struct {
	CurrencyPair     string          `json:"currency_pair"`
	Last             types.ExDecimal `json:"last"`
	LowestAsk        types.ExDecimal `json:"lowest_ask"`
	LowestSize       types.ExDecimal `json:"lowest_size"`
	HighestBid       types.ExDecimal `json:"highest_bid"`
	HighestSize      types.ExDecimal `json:"highest_size"`
	ChangePercentage types.ExDecimal `json:"change_percentage"`
	BaseVolume       types.ExDecimal `json:"base_volume"`
	QuoteVolume      types.ExDecimal `json:"quote_volume"`
	High24h          types.ExDecimal `json:"high_24h"`
	Low24h           types.ExDecimal `json:"low_24h"`
}

// ========== 永续合约市场模型 ==========

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
	Volume24h              types.ExDecimal `json:"volume_24h"`
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
