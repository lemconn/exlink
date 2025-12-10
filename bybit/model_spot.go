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
