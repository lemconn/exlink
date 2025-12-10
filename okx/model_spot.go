package okx

import (
	"github.com/lemconn/exlink/types"
)

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
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []okxTickerItem `json:"data"`
}

// okxSpotKline OKX 现货 Kline 数据（类型别名）
type okxSpotKline = okxKline

// okxSpotKlineResponse OKX 现货 Kline 响应
type okxSpotKlineResponse struct {
	Code string         `json:"code"`
	Msg  string         `json:"msg"`
	Data []okxSpotKline `json:"data"`
}

// okxSpotBalanceResponse OKX 现货余额响应
type okxSpotBalanceResponse struct {
	Code string                  `json:"code"`
	Msg  string                  `json:"msg"`
	Data []okxSpotBalanceAccount `json:"data"`
}

// okxSpotBalanceAccount OKX 现货余额账户
type okxSpotBalanceAccount struct {
	Details []okxSpotBalanceDetail `json:"details"`
}

// okxSpotBalanceDetail OKX 现货余额详情
type okxSpotBalanceDetail struct {
	AvailBal  types.ExDecimal   `json:"availBal"`
	Ccy       string            `json:"ccy"`
	Eq        types.ExDecimal   `json:"eq"`
	FrozenBal types.ExDecimal   `json:"frozenBal"`
	UTime     types.ExTimestamp `json:"uTime"`
}
