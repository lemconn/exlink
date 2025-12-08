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
