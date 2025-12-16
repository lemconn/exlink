package okx

import (
	"github.com/lemconn/exlink/types"
)

// okxPerpMarketsResponse OKX 永续合约市场信息响应
type okxPerpMarketsResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Data []okxPerpInstrument `json:"data"`
}

// okxPerpInstrument OKX 永续合约交易对信息
type okxPerpInstrument struct {
	InstType  string          `json:"instType"`
	InstID    string          `json:"instId"`
	BaseCcy   string          `json:"baseCcy"`
	QuoteCcy  string          `json:"quoteCcy"`
	SettleCcy string          `json:"settleCcy"`
	Uly       string          `json:"uly"`    // underlying，用于合约市场
	CtType    string          `json:"ctType"` // linear, inverse
	CtVal     string          `json:"ctVal"`  // 合约面值（1张合约等于多少个币）
	State     string          `json:"state"`
	MinSz     types.ExDecimal `json:"minSz"`
	MaxSz     types.ExDecimal `json:"maxSz"`
	LotSz     types.ExDecimal `json:"lotSz"`
	TickSz    types.ExDecimal `json:"tickSz"`
	MinSzVal  types.ExDecimal `json:"minSzVal"`
}

// okxPerpTickerResponse OKX 永续合约 Ticker 响应
type okxPerpTickerResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []okxTickerItem `json:"data"`
}

// okxPerpKline OKX 永续合约 Kline 数据（类型别名）
type okxPerpKline = okxKline

// okxPerpKlineResponse OKX 永续合约 Kline 响应
type okxPerpKlineResponse struct {
	Code string         `json:"code"`
	Msg  string         `json:"msg"`
	Data []okxPerpKline `json:"data"`
}

// okxPerpPositionResponse OKX 永续合约持仓响应
type okxPerpPositionResponse struct {
	Code string            `json:"code"`
	Msg  string            `json:"msg"`
	Data []okxPerpPosition `json:"data"`
}

// okxPerpPosition OKX 永续合约持仓信息
type okxPerpPosition struct {
	Adl                    types.ExDecimal   `json:"adl"`
	AvailPos               types.ExDecimal   `json:"availPos"`
	AvgPx                  types.ExDecimal   `json:"avgPx"`
	BaseBal                types.ExDecimal   `json:"baseBal"`
	BaseBorrowed           types.ExDecimal   `json:"baseBorrowed"`
	BaseInterest           types.ExDecimal   `json:"baseInterest"`
	BePx                   types.ExDecimal   `json:"bePx"`
	BizRefId               string            `json:"bizRefId"`
	BizRefType             string            `json:"bizRefType"`
	CTime                  types.ExTimestamp `json:"cTime"`
	Ccy                    string            `json:"ccy"`
	ClSpotInUseAmt         types.ExDecimal   `json:"clSpotInUseAmt"`
	CloseOrderAlgo         []interface{}     `json:"closeOrderAlgo"`
	DeltaBS                types.ExDecimal   `json:"deltaBS"`
	DeltaPA                types.ExDecimal   `json:"deltaPA"`
	Fee                    types.ExDecimal   `json:"fee"`
	FundingFee             types.ExDecimal   `json:"fundingFee"`
	GammaBS                types.ExDecimal   `json:"gammaBS"`
	GammaPA                types.ExDecimal   `json:"gammaPA"`
	HedgedPos              types.ExDecimal   `json:"hedgedPos"`
	IdxPx                  types.ExDecimal   `json:"idxPx"`
	Imr                    types.ExDecimal   `json:"imr"`
	InstID                 string            `json:"instId"`
	InstType               string            `json:"instType"`
	Interest               types.ExDecimal   `json:"interest"`
	Last                   types.ExDecimal   `json:"last"`
	Lever                  types.ExDecimal   `json:"lever"`
	Liab                   types.ExDecimal   `json:"liab"`
	LiabCcy                string            `json:"liabCcy"`
	LiqPenalty             types.ExDecimal   `json:"liqPenalty"`
	LiqPx                  types.ExDecimal   `json:"liqPx"`
	Margin                 types.ExDecimal   `json:"margin"`
	MarkPx                 types.ExDecimal   `json:"markPx"`
	MaxSpotInUseAmt        types.ExDecimal   `json:"maxSpotInUseAmt"`
	MgnMode                string            `json:"mgnMode"`
	MgnRatio               types.ExDecimal   `json:"mgnRatio"`
	Mmr                    types.ExDecimal   `json:"mmr"`
	NonSettleAvgPx         types.ExDecimal   `json:"nonSettleAvgPx"`
	NotionalUsd            types.ExDecimal   `json:"notionalUsd"`
	OptVal                 types.ExDecimal   `json:"optVal"`
	PendingCloseOrdLiabVal types.ExDecimal   `json:"pendingCloseOrdLiabVal"`
	Pnl                    types.ExDecimal   `json:"pnl"`
	Pos                    types.ExDecimal   `json:"pos"`
	PosCcy                 string            `json:"posCcy"`
	PosID                  string            `json:"posId"`
	PosSide                string            `json:"posSide"`
	QuoteBal               types.ExDecimal   `json:"quoteBal"`
	QuoteBorrowed          types.ExDecimal   `json:"quoteBorrowed"`
	QuoteInterest          types.ExDecimal   `json:"quoteInterest"`
	RealizedPnl            types.ExDecimal   `json:"realizedPnl"`
	SettledPnl             types.ExDecimal   `json:"settledPnl"`
	SpotInUseAmt           types.ExDecimal   `json:"spotInUseAmt"`
	SpotInUseCcy           string            `json:"spotInUseCcy"`
	ThetaBS                types.ExDecimal   `json:"thetaBS"`
	ThetaPA                types.ExDecimal   `json:"thetaPA"`
	TradeID                string            `json:"tradeId"`
	UTime                  types.ExTimestamp `json:"uTime"`
	Upl                    types.ExDecimal   `json:"upl"`
	UplLastPx              types.ExDecimal   `json:"uplLastPx"`
	UplRatio               types.ExDecimal   `json:"uplRatio"`
	UplRatioLastPx         types.ExDecimal   `json:"uplRatioLastPx"`
	UsdPx                  types.ExDecimal   `json:"usdPx"`
	VegaBS                 types.ExDecimal   `json:"vegaBS"`
	VegaPA                 types.ExDecimal   `json:"vegaPA"`
}

// okxPerpCreateOrderRequest OKX 永续合约创建订单请求
type okxPerpCreateOrderRequest struct {
	InstID     string `json:"instId,omitempty"`     // 交易对，如 "DOGE-USDT-SWAP"
	TdMode     string `json:"tdMode,omitempty"`     // 逐仓/全仓，如 "isolated" / "cross"
	Side       string `json:"side,omitempty"`       // "buy" / "sell"
	PosSide    string `json:"posSide,omitempty"`    // 单向持仓时不传，双向持仓时传 "long" / "short"
	OrdType    string `json:"ordType,omitempty"`    // "limit" / "market"
	Sz         string `json:"sz,omitempty"`         // 数量
	Px         string `json:"px,omitempty"`         // 限价单价格
	ReduceOnly bool   `json:"reduceOnly,omitempty"` // 是否只减仓
	ClOrdID    string `json:"clOrdId,omitempty"`    // 自定义 ID
}

// okxPerpCreateOrderResponse OKX 永续合约创建订单响应
type okxPerpCreateOrderResponse struct {
	Code string `json:"code"` // 返回码，"0" 表示成功
	Data []struct {
		ClOrdID string            `json:"clOrdId"` // 客户端订单ID
		OrdID   string            `json:"ordId"`   // 系统订单号
		TS      types.ExTimestamp `json:"ts"`      // 时间戳（毫秒）
	} `json:"data"` // 订单数据数组
	Msg string `json:"msg,omitempty"` // 返回消息
}

// okxPerpFetchOrderResponse OKX 永续合约查询订单响应
type okxPerpFetchOrderResponse struct {
	Code string                  `json:"code"`
	Msg  string                  `json:"msg"`
	Data []okxPerpFetchOrderItem `json:"data"`
}

// okxPerpFetchOrderItem OKX 永续合约查询订单项
type okxPerpFetchOrderItem struct {
	OrdID      string            `json:"ordId"`      // 订单ID
	ClOrdID    string            `json:"clOrdId"`    // 客户端自定义订单ID
	InstID     string            `json:"instId"`     // 合约标的
	Px         types.ExDecimal   `json:"px"`         // 下单价格（市价单为空）
	AvgPx      types.ExDecimal   `json:"avgPx"`      // 成交均价
	Sz         types.ExDecimal   `json:"sz"`         // 下单数量
	AccFillSz  types.ExDecimal   `json:"accFillSz"`  // 实际成交数量
	State      string            `json:"state"`      // 订单状态
	ReduceOnly string            `json:"reduceOnly"` // 是否只减仓（字符串 "true"/"false"）
	OrdType    string            `json:"ordType"`    // 订单类型
	Side       string            `json:"side"`       // 订单方向
	PosSide    string            `json:"posSide"`    // 单向持仓 net, 双向持仓 long / short
	CTime      types.ExTimestamp `json:"cTime"`      // 创建时间（毫秒）
	UTime      types.ExTimestamp `json:"uTime"`      // 更新时间（毫秒）
}
