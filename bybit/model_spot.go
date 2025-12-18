package bybit

import (
	"github.com/lemconn/exlink/types"
)

// bybitSpotMarketsResponse Bybit 现货市场信息响应
type bybitSpotMarketsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string        `json:"category"`
		List     []bybitSymbol `json:"list"`
	} `json:"result"`
}

// bybitSpotTickerResponse Bybit 现货 Ticker 响应
type bybitSpotTickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string            `json:"category"`
		List     []bybitTickerItem `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp      `json:"time"`
}

// bybitSpotKline Bybit 现货 Kline 数据（类型别名）
type bybitSpotKline = bybitKline

// bybitSpotKlineResponse Bybit 现货 Kline 响应
type bybitSpotKlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string           `json:"category"`
		Symbol   string           `json:"symbol"`
		List     []bybitSpotKline `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp      `json:"time"`
}

// bybitSpotBalanceResponse Bybit 现货余额响应
type bybitSpotBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []bybitSpotBalanceAccount `json:"list"`
	} `json:"result"`
	Time types.ExTimestamp `json:"time"`
}

// bybitSpotBalanceAccount Bybit 现货余额账户
type bybitSpotBalanceAccount struct {
	AccountType string                 `json:"accountType"`
	Coin        []bybitSpotBalanceCoin `json:"coin"`
}

// bybitSpotBalanceCoin Bybit 现货余额币种
type bybitSpotBalanceCoin struct {
	Equity          types.ExDecimal `json:"equity"`
	TotalOrderIM    types.ExDecimal `json:"totalOrderIM"`
	TotalPositionMM types.ExDecimal `json:"totalPositionMM"`
	TotalPositionIM types.ExDecimal `json:"totalPositionIM"`
	Locked          types.ExDecimal `json:"locked"`
	Coin            string          `json:"coin"`
}

// bybitSpotCreateOrderResponse Bybit 现货创建订单响应
type bybitSpotCreateOrderResponse struct {
	RetCode    int                        `json:"retCode"`
	RetMsg     string                     `json:"retMsg"`
	Result     bybitSpotCreateOrderResult `json:"result"`
	RetExtInfo map[string]interface{}     `json:"retExtInfo"`
	Time       types.ExTimestamp          `json:"time"`
}

// bybitSpotCreateOrderResult Bybit 现货创建订单结果
type bybitSpotCreateOrderResult struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// bybitSpotFetchOrderResponse Bybit 现货查询订单响应
type bybitSpotFetchOrderResponse struct {
	RetCode    int                       `json:"retCode"`
	RetMsg     string                    `json:"retMsg"`
	Result     bybitSpotFetchOrderResult `json:"result"`
	RetExtInfo map[string]interface{}    `json:"retExtInfo"`
	Time       types.ExTimestamp         `json:"time"`
}

// bybitSpotFetchOrderResult Bybit 现货查询订单结果
type bybitSpotFetchOrderResult struct {
	NextPageCursor string                       `json:"nextPageCursor"` // 下一页游标
	Category       string                       `json:"category"`       // 产品类型
	List           []bybitSpotFetchOrderItem    `json:"list"`           // 订单列表
}

// bybitSpotFetchOrderItem Bybit 现货订单详情
type bybitSpotFetchOrderItem struct {
	Symbol         string            `json:"symbol"`         // 交易对
	OrderType      string            `json:"orderType"`      // 订单类型
	OrderLinkID    string            `json:"orderLinkId"`    // 客户端订单ID
	OrderID        string            `json:"orderId"`        // 订单ID
	OrderStatus    string            `json:"orderStatus"`    // 订单状态
	AvgPrice       types.ExDecimal   `json:"avgPrice"`       // 平均成交价格
	Price          types.ExDecimal   `json:"price"`          // 订单价格
	CreatedTime    types.ExTimestamp `json:"createdTime"`    // 创建时间
	UpdatedTime    types.ExTimestamp `json:"updatedTime"`    // 更新时间
	Side           string            `json:"side"`           // 订单方向
	TimeInForce    string            `json:"timeInForce"`    // 时间有效性
	CumExecValue   types.ExDecimal   `json:"cumExecValue"`   // 累计成交金额
	CumExecFee     types.ExDecimal   `json:"cumExecFee"`     // 累计手续费
	LeavesQty      types.ExDecimal   `json:"leavesQty"`      // 剩余数量
	CumExecQty     types.ExDecimal   `json:"cumExecQty"`     // 累计成交数量
	Qty            types.ExDecimal   `json:"qty"`            // 订单数量
	LeavesValue    types.ExDecimal   `json:"leavesValue"`    // 剩余价值
}
