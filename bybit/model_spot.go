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

// bybitSpotCreateOrderRequest Bybit 现货创建订单请求
type bybitSpotCreateOrderRequest struct {
	Category    string `json:"category"`              // 产品类型：spot
	Symbol      string `json:"symbol"`                // 交易对
	Side        string `json:"side"`                  // 订单方向 Buy/Sell
	OrderType   string `json:"orderType"`             // 订单类型 Market/Limit
	Qty         string `json:"qty"`                   // 数量
	Price       string `json:"price,omitempty"`       // 价格（限价单必填）
	TimeInForce string `json:"timeInForce,omitempty"` // 有效期类型
	OrderLinkId string `json:"orderLinkId,omitempty"` // 客户端订单ID
	MarketUnit  string `json:"marketUnit,omitempty"`  // 市价单位 baseCoin/quoteCoin
}

// bybitSpotCreateOrderResponse Bybit 现货创建订单响应
type bybitSpotCreateOrderResponse struct {
	RetCode    int                          `json:"retCode"`
	RetMsg     string                       `json:"retMsg"`
	Result     bybitSpotCreateOrderResult   `json:"result"`
	RetExtInfo map[string]interface{}       `json:"retExtInfo"`
	Time       types.ExTimestamp            `json:"time"`
}

// bybitSpotCreateOrderResult Bybit 现货创建订单结果
type bybitSpotCreateOrderResult struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}
