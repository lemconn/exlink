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
