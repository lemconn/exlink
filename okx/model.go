package okx

import (
	"encoding/json"
	"fmt"

	"github.com/lemconn/exlink/types"
)

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
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []okxTickerItem `json:"data"`
}

// okxTickerItem OKX Ticker 数据项（现货和合约共用）
type okxTickerItem struct {
	InstType  string            `json:"instType"`
	InstID    string            `json:"instId"`
	Last      types.ExDecimal   `json:"last"`
	LastSz    types.ExDecimal   `json:"lastSz"`
	AskPx     types.ExDecimal   `json:"askPx"`
	AskSz     types.ExDecimal   `json:"askSz"`
	BidPx     types.ExDecimal   `json:"bidPx"`
	BidSz     types.ExDecimal   `json:"bidSz"`
	Open24h   types.ExDecimal   `json:"open24h"`
	High24h   types.ExDecimal   `json:"high24h"`
	Low24h    types.ExDecimal   `json:"low24h"`
	VolCcy24h types.ExDecimal   `json:"volCcy24h"`
	Vol24h    types.ExDecimal   `json:"vol24h"`
	Ts        types.ExTimestamp `json:"ts"`
	SodUtc0   types.ExDecimal   `json:"sodUtc0"`
	SodUtc8   types.ExDecimal   `json:"sodUtc8"`
}

// okxKline OKX Kline 数据（现货和合约共用）
type okxKline struct {
	Ts             types.ExTimestamp `json:"ts"`             // ts, Open time
	Open           types.ExDecimal   `json:"open"`           // o, Open price
	High           types.ExDecimal   `json:"high"`           // h, Highest price
	Low            types.ExDecimal   `json:"low"`            // l, Lowest price
	Close          types.ExDecimal   `json:"close"`          // c, Close price
	Volume         types.ExDecimal   `json:"volume"`         // vol, Trading volume is the number of contracts for derivatives or the base currency quantity for spot/margin
	VolumeCcy      types.ExDecimal   `json:"volumeCcy"`      // volCcy, Trading volume is measured in base currency for derivatives and in quote currency for spot/margin
	VolumeCcyQuote types.ExDecimal   `json:"volumeCcyQuote"` // volCcyQuote, Trading volume is the quote currency quantity
	Confirm        types.ExDecimal   `json:"confirm"`        // confirm, Candlestick state: 0 for uncompleted, 1 for completed
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
	Code string                    `json:"code"`
	Msg  string                    `json:"msg"`
	Data []okxSpotBalanceAccount    `json:"data"`
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

// okxPerpKline OKX 永续合约 Kline 数据（类型别名）
type okxPerpKline = okxKline

// okxPerpKlineResponse OKX 永续合约 Kline 响应
type okxPerpKlineResponse struct {
	Code string         `json:"code"`
	Msg  string         `json:"msg"`
	Data []okxPerpKline `json:"data"`
}

// UnmarshalJSON 自定义 JSON 反序列化，解析数组格式
func (k *okxKline) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) < 9 {
		return fmt.Errorf("invalid kline array length: %d", len(arr))
	}

	// Ts (index 0)
	if arr[0] != "" {
		k.Ts = types.ExTimestamp{}
		if err := k.Ts.UnmarshalJSON([]byte(arr[0])); err != nil {
			return fmt.Errorf("parse ts: %w", err)
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

	// VolumeCcy (index 6)
	if arr[6] != "" {
		k.VolumeCcy = types.ExDecimal{}
		if err := k.VolumeCcy.UnmarshalJSON([]byte(`"` + arr[6] + `"`)); err != nil {
			return fmt.Errorf("parse volumeCcy: %w", err)
		}
	}

	// VolumeCcyQuote (index 7)
	if arr[7] != "" {
		k.VolumeCcyQuote = types.ExDecimal{}
		if err := k.VolumeCcyQuote.UnmarshalJSON([]byte(`"` + arr[7] + `"`)); err != nil {
			return fmt.Errorf("parse volumeCcyQuote: %w", err)
		}
	}

	// Confirm (index 8)
	if arr[8] != "" {
		k.Confirm = types.ExDecimal{}
		_ = k.Confirm.UnmarshalJSON([]byte(`"` + arr[8] + `"`))
	}

	return nil
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
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data []okxTickerItem `json:"data"`
}
