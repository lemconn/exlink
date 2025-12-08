package bybit

import "github.com/lemconn/exlink/types"

// ========== 现货市场模型 ==========

// bybitSpotMarketsResponse Bybit 现货市场信息响应
type bybitSpotMarketsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string        `json:"category"`
		List     []bybitSymbol `json:"list"`
	} `json:"result"`
}

// bybitSymbol Bybit 交易对信息（现货和合约共用）
type bybitSymbol struct {
	Symbol        string             `json:"symbol"`
	BaseCoin      string             `json:"baseCoin"`
	QuoteCoin     string             `json:"quoteCoin"`
	Status        string             `json:"status"`
	LotSizeFilter bybitLotSizeFilter `json:"lotSizeFilter"`
	PriceFilter   bybitPriceFilter   `json:"priceFilter"`
}

// bybitLotSizeFilter Bybit 数量过滤器
type bybitLotSizeFilter struct {
	BasePrecision  types.ExDecimal `json:"basePrecision"`
	QuotePrecision types.ExDecimal `json:"quotePrecision"`
	MinOrderQty    types.ExDecimal `json:"minOrderQty"`
	MaxOrderQty    types.ExDecimal `json:"maxOrderQty"`
	MinOrderAmt    types.ExDecimal `json:"minOrderAmt"`
	MaxOrderAmt    types.ExDecimal `json:"maxOrderAmt"`
}

// bybitPriceFilter Bybit 价格过滤器
type bybitPriceFilter struct {
	TickSize types.ExDecimal `json:"tickSize"`
}

// ========== 永续合约市场模型 ==========

// bybitPerpMarketsResponse Bybit 永续合约市场信息响应
type bybitPerpMarketsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string        `json:"category"`
		List     []bybitSymbol `json:"list"`
	} `json:"result"`
}
