package bybit

import (
	"github.com/lemconn/exlink/types"
)

// bybitPerpMarketsResponse Bybit 永续合约市场信息响应
type bybitPerpMarketsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string        `json:"category"`
		List     []bybitSymbol `json:"list"`
	} `json:"result"`
}

// bybitPerpTickerResponse Bybit 永续合约 Ticker 响应
type bybitPerpTickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string            `json:"category"`
		List     []bybitTickerItem `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp      `json:"time"`
}

// bybitPerpKline Bybit 永续合约 Kline 数据（类型别名）
type bybitPerpKline = bybitKline

// bybitPerpKlineResponse Bybit 永续合约 Kline 响应
type bybitPerpKlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string           `json:"category"`
		Symbol   string           `json:"symbol"`
		List     []bybitPerpKline `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp      `json:"time"`
}

// bybitPerpPositionResponse Bybit 永续合约持仓响应
type bybitPerpPositionResponse struct {
	RetCode int                     `json:"retCode"`
	RetMsg  string                  `json:"retMsg"`
	Result  bybitPerpPositionResult `json:"result"`
	Time    types.ExTimestamp       `json:"time"`
}

// bybitPerpPositionResult Bybit 永续合约持仓结果
type bybitPerpPositionResult struct {
	Category string              `json:"category"`
	List     []bybitPerpPosition `json:"list"`
}

// bybitPerpPosition Bybit 永续合约持仓信息
type bybitPerpPosition struct {
	Symbol                 string            `json:"symbol"`
	Leverage               types.ExDecimal   `json:"leverage"`
	AutoAddMargin          int               `json:"autoAddMargin"`
	AvgPrice               types.ExDecimal   `json:"avgPrice"`
	LiqPrice               types.ExDecimal   `json:"liqPrice"`
	RiskLimitValue         types.ExDecimal   `json:"riskLimitValue"`
	TakeProfit             types.ExDecimal   `json:"takeProfit"`
	PositionValue          types.ExDecimal   `json:"positionValue"`
	IsReduceOnly           bool              `json:"isReduceOnly"`
	PositionIMByMp         types.ExDecimal   `json:"positionIMByMp"`
	TpslMode               string            `json:"tpslMode"`
	RiskId                 int               `json:"riskId"`
	TrailingStop           types.ExDecimal   `json:"trailingStop"`
	UnrealisedPnl          types.ExDecimal   `json:"unrealisedPnl"`
	MarkPrice              types.ExDecimal   `json:"markPrice"`
	AdlRankIndicator       int               `json:"adlRankIndicator"`
	CumRealisedPnl         types.ExDecimal   `json:"cumRealisedPnl"`
	PositionMM             types.ExDecimal   `json:"positionMM"`
	CreatedTime            types.ExTimestamp `json:"createdTime"`
	PositionIdx            int               `json:"positionIdx"`
	PositionIM             types.ExDecimal   `json:"positionIM"`
	PositionMMByMp         types.ExDecimal   `json:"positionMMByMp"`
	Seq                    int64             `json:"seq"`
	UpdatedTime            types.ExTimestamp `json:"updatedTime"`
	Side                   string            `json:"side"`
	BustPrice              types.ExDecimal   `json:"bustPrice"`
	PositionBalance        types.ExDecimal   `json:"positionBalance"`
	LeverageSysUpdatedTime types.ExTimestamp `json:"leverageSysUpdatedTime"`
	CurRealisedPnl         types.ExDecimal   `json:"curRealisedPnl"`
	Size                   types.ExDecimal   `json:"size"`
	PositionStatus         string            `json:"positionStatus"`
	MmrSysUpdatedTime      types.ExTimestamp `json:"mmrSysUpdatedTime"`
	StopLoss               types.ExDecimal   `json:"stopLoss"`
	TradeMode              int               `json:"tradeMode"`
	SessionAvgPrice        types.ExDecimal   `json:"sessionAvgPrice"`
}

// bybitPerpCreateOrderRequest Bybit 永续合约创建订单请求
type bybitPerpCreateOrderRequest struct {
	Category    string `json:"category,omitempty"`    // "linear"
	Symbol      string `json:"symbol,omitempty"`      // 交易对
	Side        string `json:"side,omitempty"`        // "Buy" / "Sell"
	OrderType   string `json:"orderType,omitempty"`   // "Limit" / "Market"
	Qty         string `json:"qty,omitempty"`         // 数量
	MarketUnit  string `json:"marketUnit,omitempty"`  // "baseCoin"
	ReduceOnly  bool   `json:"reduceOnly,omitempty"`  // 是否只减仓
	TimeInForce string `json:"timeInForce,omitempty"` // 限价单不传（实际代码中限价单会传 GTC）
	Price       string `json:"price,omitempty"`       // 限价单价格
	PositionIdx int    `json:"positionIdx,omitempty"` // 单向持仓时不传，双向持仓时 开多/平多 → positionIdx 等于 1，开空/平空 → positionIdx 等于 2
	OrderLinkID string `json:"orderLinkId,omitempty"` // 自定义 ID
}

// bybitPerpCreateOrderResponse Bybit 永续合约创建订单响应
type bybitPerpCreateOrderResponse struct {
	RetCode int    `json:"retCode"` // 返回码，0 表示成功
	RetMsg  string `json:"retMsg"`  // 返回消息
	Result  struct {
		OrderID     string `json:"orderId"`     // 系统订单号
		OrderLinkID string `json:"orderLinkId"` // 客户端订单ID
	} `json:"result"` // 订单结果
	RetExtInfo map[string]interface{} `json:"retExtInfo"` // 扩展信息
	Time       types.ExTimestamp      `json:"time"`       // 时间戳（毫秒）
}

// bybitPerpFetchOrderResponse Bybit 永续合约查询订单响应
type bybitPerpFetchOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []bybitPerpFetchOrderItem `json:"list"`
	} `json:"result"`
}

// bybitPerpFetchOrderItem Bybit 永续合约查询订单项
type bybitPerpFetchOrderItem struct {
	OrderID     string            `json:"orderId"`     // 订单ID
	OrderLinkID string            `json:"orderLinkId"` // 客户端自定义订单ID
	Symbol      string            `json:"symbol"`      // 交易对 / 合约标的
	Price       types.ExDecimal   `json:"price"`       // 下单价格
	AvgPrice    types.ExDecimal   `json:"avgPrice"`    // 成交均价
	Qty         types.ExDecimal   `json:"qty"`         // 下单数量
	CumExecQty  types.ExDecimal   `json:"cumExecQty"`  // 实际成交数量
	OrderStatus string            `json:"orderStatus"` // 订单状态
	TimeInForce string            `json:"timeInForce"` // 订单有效方式
	ReduceOnly  bool              `json:"reduceOnly"`  // 是否只减仓
	OrderType   string            `json:"orderType"`   // 订单类型
	Side        string            `json:"side"`        // 订单方向
	PositionIdx int               `json:"positionIdx"` // 单向持仓 positionIdx 等于 0，双向持仓 开多/平多 → positionIdx 等于 1，开空/平空 → positionIdx 等于 2
	CreatedTime types.ExTimestamp `json:"createdTime"` // 创建时间（毫秒）
	UpdatedTime types.ExTimestamp `json:"updatedTime"` // 更新时间（毫秒）
}
