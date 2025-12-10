package bybit

import (
	"encoding/json"
	"fmt"

	"github.com/lemconn/exlink/types"
)

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

// bybitTickerItem Bybit Ticker 数据项（现货和合约共用）
type bybitTickerItem struct {
	Symbol                 string            `json:"symbol"`
	LastPrice              types.ExDecimal   `json:"lastPrice"`
	IndexPrice             types.ExDecimal   `json:"indexPrice"`
	MarkPrice              types.ExDecimal   `json:"markPrice"`
	PrevPrice24h           types.ExDecimal   `json:"prevPrice24h"`
	Price24hPcnt           types.ExDecimal   `json:"price24hPcnt"`
	HighPrice24h           types.ExDecimal   `json:"highPrice24h"`
	LowPrice24h            types.ExDecimal   `json:"lowPrice24h"`
	PrevPrice1h            types.ExDecimal   `json:"prevPrice1h"`
	OpenInterest           types.ExDecimal   `json:"openInterest"`
	OpenInterestValue      types.ExDecimal   `json:"openInterestValue"`
	Turnover24h            types.ExDecimal   `json:"turnover24h"`
	Volume24h              types.ExDecimal   `json:"volume24h"`
	FundingRate            types.ExDecimal   `json:"fundingRate"`
	NextFundingTime        types.ExTimestamp `json:"nextFundingTime"`
	PredictedDeliveryPrice types.ExDecimal   `json:"predictedDeliveryPrice"`
	BasisRate              types.ExDecimal   `json:"basisRate"`
	DeliveryFeeRate        types.ExDecimal   `json:"deliveryFeeRate"`
	DeliveryTime           types.ExTimestamp `json:"deliveryTime"`
	Ask1Size               types.ExDecimal   `json:"ask1Size"`
	Bid1Price              types.ExDecimal   `json:"bid1Price"`
	Ask1Price              types.ExDecimal   `json:"ask1Price"`
	Bid1Size               types.ExDecimal   `json:"bid1Size"`
	Basis                  types.ExDecimal   `json:"basis"`
	PreOpenPrice           types.ExDecimal   `json:"preOpenPrice"`
	PreQty                 types.ExDecimal   `json:"preQty"`
	CurPreListingPhase     string            `json:"curPreListingPhase"`
	FundingIntervalHour    string            `json:"fundingIntervalHour"`
	BasisRateYear          types.ExDecimal   `json:"basisRateYear"`
	FundingCap             types.ExDecimal   `json:"fundingCap"`
}

// bybitKline Bybit Kline 数据（现货和合约共用）
type bybitKline struct {
	StartTime types.ExTimestamp `json:"startTime"` // startTime
	Open      types.ExDecimal   `json:"open"`      // openPrice
	High      types.ExDecimal   `json:"high"`      // highPrice
	Low       types.ExDecimal   `json:"low"`       // lowPrice
	Close     types.ExDecimal   `json:"close"`     // closePrice
	Volume    types.ExDecimal   `json:"volume"`    // volume (Trade volume USDT or USDC, contract: unit is base coin)
	Turnover  types.ExDecimal   `json:"turnover"`  // turnover (Turnover USDT or USDC, contract: unit is quote coin)
}

// UnmarshalJSON 自定义 JSON 反序列化，解析数组格式
func (k *bybitKline) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) < 7 {
		return fmt.Errorf("invalid kline array length: %d", len(arr))
	}

	// StartTime (index 0)
	if arr[0] != "" {
		k.StartTime = types.ExTimestamp{}
		if err := k.StartTime.UnmarshalJSON([]byte(arr[0])); err != nil {
			return fmt.Errorf("parse startTime: %w", err)
		}
	}

	// Open (index 1)
	if arr[1] != "" {
		k.Open = types.ExDecimal{}
		if err := k.Open.UnmarshalJSON([]byte(`"` + arr[1] + `"`)); err != nil {
			return fmt.Errorf("parse open: %w", err)
		}
	}

	// High (index 2)
	if arr[2] != "" {
		k.High = types.ExDecimal{}
		if err := k.High.UnmarshalJSON([]byte(`"` + arr[2] + `"`)); err != nil {
			return fmt.Errorf("parse high: %w", err)
		}
	}

	// Low (index 3)
	if arr[3] != "" {
		k.Low = types.ExDecimal{}
		if err := k.Low.UnmarshalJSON([]byte(`"` + arr[3] + `"`)); err != nil {
			return fmt.Errorf("parse low: %w", err)
		}
	}

	// Close (index 4)
	if arr[4] != "" {
		k.Close = types.ExDecimal{}
		if err := k.Close.UnmarshalJSON([]byte(`"` + arr[4] + `"`)); err != nil {
			return fmt.Errorf("parse close: %w", err)
		}
	}

	// Volume (index 5)
	if arr[5] != "" {
		k.Volume = types.ExDecimal{}
		if err := k.Volume.UnmarshalJSON([]byte(`"` + arr[5] + `"`)); err != nil {
			return fmt.Errorf("parse volume: %w", err)
		}
	}

	// Turnover (index 6)
	if arr[6] != "" {
		k.Turnover = types.ExDecimal{}
		if err := k.Turnover.UnmarshalJSON([]byte(`"` + arr[6] + `"`)); err != nil {
			return fmt.Errorf("parse turnover: %w", err)
		}
	}

	return nil
}
