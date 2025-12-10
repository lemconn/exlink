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
