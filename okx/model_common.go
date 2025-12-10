package okx

import (
	"encoding/json"
	"fmt"

	"github.com/lemconn/exlink/types"
)

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
