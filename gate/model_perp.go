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

// gatePerpPositionResponse Gate 永续合约持仓响应（数组格式）
type gatePerpPositionResponse []gatePerpPosition

// gatePerpPosition Gate 永续合约持仓信息
type gatePerpPosition struct {
	Value                  types.ExDecimal   `json:"value"`
	Leverage               types.ExDecimal   `json:"leverage"`
	Mode                   string            `json:"mode"`
	RealisedPoint          types.ExDecimal   `json:"realised_point"`
	Contract               string            `json:"contract"`
	EntryPrice             types.ExDecimal   `json:"entry_price"`
	MarkPrice              types.ExDecimal   `json:"mark_price"`
	HistoryPoint           types.ExDecimal   `json:"history_point"`
	RealisedPnl            types.ExDecimal   `json:"realised_pnl"`
	CloseOrder             interface{}       `json:"close_order"`
	Size                   types.ExDecimal   `json:"size"`
	CrossLeverageLimit     types.ExDecimal   `json:"cross_leverage_limit"`
	PendingOrders          int               `json:"pending_orders"`
	AdlRanking             int               `json:"adl_ranking"`
	MaintenanceRate        types.ExDecimal   `json:"maintenance_rate"`
	UnrealisedPnl          types.ExDecimal   `json:"unrealised_pnl"`
	PnlPnl                 types.ExDecimal   `json:"pnl_pnl"`
	PnlFee                 types.ExDecimal   `json:"pnl_fee"`
	PnlFund                types.ExDecimal   `json:"pnl_fund"`
	User                   int64             `json:"user"`
	LeverageMax            types.ExDecimal   `json:"leverage_max"`
	HistoryPnl             types.ExDecimal   `json:"history_pnl"`
	RiskLimit              types.ExDecimal   `json:"risk_limit"`
	Margin                 types.ExDecimal   `json:"margin"`
	LastClosePnl           types.ExDecimal   `json:"last_close_pnl"`
	LiqPrice               types.ExDecimal   `json:"liq_price"`
	UpdateTime             types.ExTimestamp `json:"update_time"`
	UpdateID               int64             `json:"update_id"`
	InitialMargin          types.ExDecimal   `json:"initial_margin"`
	MaintenanceMargin      types.ExDecimal   `json:"maintenance_margin"`
	TradeLongSize          types.ExDecimal   `json:"trade_long_size"`
	TradeShortSize         types.ExDecimal   `json:"trade_short_size"`
	OpenTime               types.ExTimestamp `json:"open_time"`
	RiskLimitTable         string            `json:"risk_limit_table"`
	AverageMaintenanceRate types.ExDecimal   `json:"average_maintenance_rate"`
	VoucherSize            types.ExDecimal   `json:"voucher_size"`
	VoucherMargin          types.ExDecimal   `json:"voucher_margin"`
	VoucherID              int               `json:"voucher_id"`
}

// gatePerpCreateOrderRequest Gate 永续合约创建订单请求
type gatePerpCreateOrderRequest struct {
	Contract   string `json:"contract,omitempty"`    // 交易对，如 "DOGE_USDT"
	Size       int64  `json:"size,omitempty"`        // 合约张数，正数为买、负数为卖
	Price      string `json:"price,omitempty"`       // 价格
	Tif        string `json:"tif,omitempty"`         // 市价单只能设置为 "ioc"，限价单为 "gtc"
	ReduceOnly bool   `json:"reduce_only,omitempty"` // 是否只减仓
	Text       string `json:"text,omitempty"`        // 自定义 ID
}

// gatePerpCreateOrderResponse Gate 永续合约创建订单响应
type gatePerpCreateOrderResponse struct {
	ID         string           `json:"id"`         // 系统订单号
	Text       string           `json:"text"`      // 客户端订单ID
	UpdateTime types.ExTimestamp `json:"update_time"` // 更新时间（秒级时间戳，带小数）
}
