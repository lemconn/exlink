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
