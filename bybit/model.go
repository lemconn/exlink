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

// bybitLotSizeFilter Bybit 数量过滤器（现货和合约共用）
type bybitLotSizeFilter struct {
	BasePrecision  types.ExDecimal `json:"basePrecision"`
	QuotePrecision types.ExDecimal `json:"quotePrecision"`
	MinOrderQty    types.ExDecimal `json:"minOrderQty"`
	MaxOrderQty    types.ExDecimal `json:"maxOrderQty"`
	MinOrderAmt    types.ExDecimal `json:"minOrderAmt"`
	MaxOrderAmt    types.ExDecimal `json:"maxOrderAmt"`
}

// bybitPriceFilter Bybit 价格过滤器（现货和合约共用）
type bybitPriceFilter struct {
	TickSize types.ExDecimal `json:"tickSize"`
}

// bybitSpotTickerResponse Bybit 现货 Ticker 响应
type bybitSpotTickerResponse struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	Result     struct {
		Category string            `json:"category"`
		List     []bybitTickerItem `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp       `json:"time"`
}

// bybitTickerItem Bybit Ticker 数据项（现货和合约共用）
type bybitTickerItem struct {
	Symbol                 string          `json:"symbol"`
	LastPrice              types.ExDecimal `json:"lastPrice"`
	IndexPrice             types.ExDecimal `json:"indexPrice"`
	MarkPrice              types.ExDecimal `json:"markPrice"`
	PrevPrice24h           types.ExDecimal `json:"prevPrice24h"`
	Price24hPcnt           types.ExDecimal `json:"price24hPcnt"`
	HighPrice24h           types.ExDecimal `json:"highPrice24h"`
	LowPrice24h             types.ExDecimal `json:"lowPrice24h"`
	PrevPrice1h            types.ExDecimal `json:"prevPrice1h"`
	OpenInterest           types.ExDecimal `json:"openInterest"`
	OpenInterestValue      types.ExDecimal `json:"openInterestValue"`
	Turnover24h            types.ExDecimal `json:"turnover24h"`
	Volume24h               types.ExDecimal `json:"volume24h"`
	FundingRate             types.ExDecimal `json:"fundingRate"`
	NextFundingTime         types.ExTimestamp `json:"nextFundingTime"`
	PredictedDeliveryPrice  types.ExDecimal `json:"predictedDeliveryPrice"`
	BasisRate               types.ExDecimal `json:"basisRate"`
	DeliveryFeeRate         types.ExDecimal `json:"deliveryFeeRate"`
	DeliveryTime            types.ExTimestamp `json:"deliveryTime"`
	Ask1Size                types.ExDecimal `json:"ask1Size"`
	Bid1Price               types.ExDecimal `json:"bid1Price"`
	Ask1Price                types.ExDecimal `json:"ask1Price"`
	Bid1Size                 types.ExDecimal `json:"bid1Size"`
	Basis                    types.ExDecimal `json:"basis"`
	PreOpenPrice             types.ExDecimal `json:"preOpenPrice"`
	PreQty                   types.ExDecimal `json:"preQty"`
	CurPreListingPhase       string          `json:"curPreListingPhase"`
	FundingIntervalHour      string          `json:"fundingIntervalHour"`
	BasisRateYear            types.ExDecimal `json:"basisRateYear"`
	FundingCap               types.ExDecimal `json:"fundingCap"`
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

// bybitPerpTickerResponse Bybit 永续合约 Ticker 响应
type bybitPerpTickerResponse struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	Result     struct {
		Category string            `json:"category"`
		List     []bybitTickerItem `json:"list"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       types.ExTimestamp       `json:"time"`
}
