package types

// BinanceContractOrderResponse 币安合约订单响应
type BinanceContractOrderResponse struct {
	ClientOrderID           string `json:"clientOrderId"`           // 用户自定义的订单号
	CumQty                  string `json:"cumQty"`                  // 成交量
	CumQuote                string `json:"cumQuote"`                // 成交金额
	ExecutedQty             string `json:"executedQty"`             // 成交量
	OrderID                 int64  `json:"orderId"`                 // 系统订单号
	AvgPrice                string `json:"avgPrice"`                // 平均成交价
	OrigQty                 string `json:"origQty"`                 // 原始委托数量
	Price                   string `json:"price"`                   // 委托价格
	ReduceOnly              bool   `json:"reduceOnly"`              // 仅减仓
	Side                    string `json:"side"`                    // 买卖方向
	PositionSide            string `json:"positionSide"`            // 持仓方向
	Status                  string `json:"status"`                  // 订单状态
	StopPrice               string `json:"stopPrice"`               // 触发价，对`TRAILING_STOP_MARKET`无效
	ClosePosition           bool   `json:"closePosition"`           // 是否条件全平仓
	Symbol                  string `json:"symbol"`                  // 交易对
	TimeInForce             string `json:"timeInForce"`             // 有效方法
	Type                    string `json:"type"`                    // 订单类型
	OrigType                string `json:"origType"`                // 触发前订单类型
	ActivatePrice           string `json:"activatePrice"`           // 跟踪止损激活价格, 仅`TRAILING_STOP_MARKET` 订单返回此字段
	PriceRate               string `json:"priceRate"`               // 跟踪止损回调比例, 仅`TRAILING_STOP_MARKET` 订单返回此字段
	UpdateTime              int64  `json:"updateTime"`              // 更新时间
	WorkingType             string `json:"workingType"`             // 条件价格触发类型
	PriceProtect            bool   `json:"priceProtect"`            // 是否开启条件单触发保护
	PriceMatch              string `json:"priceMatch"`              // 盘口价格下单模式
	SelfTradePreventionMode string `json:"selfTradePreventionMode"` // 订单自成交保护模式
	GoodTillDate            int64  `json:"goodTillDate"`            // 订单TIF为GTD时的自动取消时间
}

// BinanceSpotOrderFill 币安现货订单成交明细
type BinanceSpotOrderFill struct {
	Price           string `json:"price"`           // 成交价格
	Qty             string `json:"qty"`             // 成交数量
	Commission      string `json:"commission"`      // 手续费
	CommissionAsset string `json:"commissionAsset"` // 手续费币种
	TradeID         int64  `json:"tradeId"`         // 交易ID
}

// BinanceSpotOrderResponse 币安现货订单响应
type BinanceSpotOrderResponse struct {
	Symbol                  string                 `json:"symbol"`                  // 交易对
	OrderID                 int64                  `json:"orderId"`                 // 订单ID
	OrderListID             int64                  `json:"orderListId"`             // 除非此单是订单列表的一部分, 否则此值为 -1
	ClientOrderID           string                 `json:"clientOrderId"`           // 客户端订单ID
	TransactTime            int64                  `json:"transactTime"`            // 交易时间
	Price                   string                 `json:"price"`                   // 价格
	OrigQty                 string                 `json:"origQty"`                 // 原始委托数量
	ExecutedQty             string                 `json:"executedQty"`             // 成交量
	OrigQuoteOrderQty       string                 `json:"origQuoteOrderQty"`       // 原始报价数量
	CummulativeQuoteQty     string                 `json:"cummulativeQuoteQty"`     // 累计报价数量
	Status                  string                 `json:"status"`                  // 订单状态
	TimeInForce             string                 `json:"timeInForce"`             // 有效方法
	Type                    string                 `json:"type"`                    // 订单类型
	Side                    string                 `json:"side"`                    // 买卖方向
	WorkingTime             int64                  `json:"workingTime"`             // 工作时间
	SelfTradePreventionMode string                 `json:"selfTradePreventionMode"` // 订单自成交保护模式
	Fills                   []BinanceSpotOrderFill `json:"fills"`                   // 成交明细
}
