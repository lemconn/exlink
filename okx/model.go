package okx

import "github.com/lemconn/exlink/types"

// ========== 现货市场模型 ==========

// okxSpotMarketsResponse OKX 现货市场信息响应
type okxSpotMarketsResponse struct {
	Code string              `json:"code"`
	Msg  string              `json:"msg"`
	Data []okxSpotInstrument `json:"data"`
}

// okxSpotInstrument OKX 现货交易对信息
type okxSpotInstrument struct {
	InstType string          `json:"instType"`
	InstID   string          `json:"instId"`
	BaseCcy  string          `json:"baseCcy"`
	QuoteCcy string          `json:"quoteCcy"`
	State    string          `json:"state"`
	MinSz    types.ExDecimal `json:"minSz"`
	MaxSz    types.ExDecimal `json:"maxSz"`
	LotSz    types.ExDecimal `json:"lotSz"`
	TickSz   types.ExDecimal `json:"tickSz"`
	MinSzVal types.ExDecimal `json:"minSzVal"`
}

// okxSpotTickerResponse OKX 现货 Ticker 响应
type okxSpotTickerResponse struct {
	Code string           `json:"code"`
	Msg  string           `json:"msg"`
	Data []okxTickerItem `json:"data"`
}

// okxTickerItem OKX Ticker 数据项（现货和合约共用）
type okxTickerItem struct {
	InstType  string          `json:"instType"`
	InstID    string          `json:"instId"`
	Last      types.ExDecimal `json:"last"`
	LastSz    types.ExDecimal `json:"lastSz"`
	AskPx     types.ExDecimal `json:"askPx"`
	AskSz     types.ExDecimal `json:"askSz"`
	BidPx     types.ExDecimal `json:"bidPx"`
	BidSz     types.ExDecimal `json:"bidSz"`
	Open24h   types.ExDecimal `json:"open24h"`
	High24h   types.ExDecimal `json:"high24h"`
	Low24h    types.ExDecimal `json:"low24h"`
	VolCcy24h types.ExDecimal `json:"volCcy24h"`
	Vol24h    types.ExDecimal `json:"vol24h"`
	Ts        types.ExTimestamp `json:"ts"`
	SodUtc0   types.ExDecimal `json:"sodUtc0"`
	SodUtc8   types.ExDecimal `json:"sodUtc8"`
}

// ========== 永续合约市场模型 ==========

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
	Code string           `json:"code"`
	Msg  string           `json:"msg"`
	Data []okxTickerItem `json:"data"`
}
