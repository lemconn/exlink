package bybit

import (
	"github.com/lemconn/exlink/types"
)

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
