package binance

import (
	"encoding/json"
	"fmt"

	"github.com/lemconn/exlink/types"
)

// binanceFilter Binance 过滤器（现货和合约共用）
type binanceFilter struct {
	FilterType  string          `json:"filterType"`
	MinQty      types.ExDecimal `json:"minQty,omitempty"`
	MaxQty      types.ExDecimal `json:"maxQty,omitempty"`
	StepSize    types.ExDecimal `json:"stepSize,omitempty"`
	MinPrice    types.ExDecimal `json:"minPrice,omitempty"`
	MaxPrice    types.ExDecimal `json:"maxPrice,omitempty"`
	TickSize    types.ExDecimal `json:"tickSize,omitempty"`
	MinNotional types.ExDecimal `json:"minNotional,omitempty"`
}

// binanceKline Binance Kline 数据（现货和合约共用）
type binanceKline struct {
	OpenTime            types.ExTimestamp `json:"openTime"`            // Kline open time
	Open                types.ExDecimal   `json:"open"`                // Open price
	High                types.ExDecimal   `json:"high"`                // High price
	Low                 types.ExDecimal   `json:"low"`                 // Low price
	Close               types.ExDecimal   `json:"close"`               // Close price
	Volume              types.ExDecimal   `json:"volume"`              // Volume
	CloseTime           types.ExTimestamp `json:"closeTime"`           // Kline Close time
	QuoteVolume         types.ExDecimal   `json:"quoteVolume"`         // Quote asset volume
	Trades              int64             `json:"trades"`              // Number of trades
	TakerBuyBaseVolume  types.ExDecimal   `json:"takerBuyBaseVolume"`  // Taker buy base asset volume
	TakerBuyQuoteVolume types.ExDecimal   `json:"takerBuyQuoteVolume"` // Taker buy quote asset volume
	Ignore              types.ExDecimal   `json:"ignore"`              // Unused field, ignore
}

// UnmarshalJSON 自定义 JSON 反序列化，解析数组格式
func (k *binanceKline) UnmarshalJSON(data []byte) error {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) < 12 {
		return fmt.Errorf("invalid kline array length: %d", len(arr))
	}

	// OpenTime (index 0)
	if v, ok := arr[0].(float64); ok {
		k.OpenTime = types.ExTimestamp{}
		if err := k.OpenTime.UnmarshalJSON([]byte(fmt.Sprintf("%.0f", v))); err != nil {
			return fmt.Errorf("parse openTime: %w", err)
		}
	} else if v, ok := arr[0].(json.Number); ok {
		k.OpenTime = types.ExTimestamp{}
		if err := k.OpenTime.UnmarshalJSON([]byte(v.String())); err != nil {
			return fmt.Errorf("parse openTime: %w", err)
		}
	}

	// Open (index 1)
	if v, ok := arr[1].(string); ok {
		k.Open = types.ExDecimal{}
		if err := k.Open.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse open: %w", err)
		}
	}

	// High (index 2)
	if v, ok := arr[2].(string); ok {
		k.High = types.ExDecimal{}
		if err := k.High.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse high: %w", err)
		}
	}

	// Low (index 3)
	if v, ok := arr[3].(string); ok {
		k.Low = types.ExDecimal{}
		if err := k.Low.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse low: %w", err)
		}
	}

	// Close (index 4)
	if v, ok := arr[4].(string); ok {
		k.Close = types.ExDecimal{}
		if err := k.Close.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse close: %w", err)
		}
	}

	// Volume (index 5)
	if v, ok := arr[5].(string); ok {
		k.Volume = types.ExDecimal{}
		if err := k.Volume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse volume: %w", err)
		}
	}

	// CloseTime (index 6)
	if v, ok := arr[6].(float64); ok {
		k.CloseTime = types.ExTimestamp{}
		if err := k.CloseTime.UnmarshalJSON([]byte(fmt.Sprintf("%.0f", v))); err != nil {
			return fmt.Errorf("parse closeTime: %w", err)
		}
	} else if v, ok := arr[6].(json.Number); ok {
		k.CloseTime = types.ExTimestamp{}
		if err := k.CloseTime.UnmarshalJSON([]byte(v.String())); err != nil {
			return fmt.Errorf("parse closeTime: %w", err)
		}
	}

	// QuoteVolume (index 7)
	if v, ok := arr[7].(string); ok {
		k.QuoteVolume = types.ExDecimal{}
		if err := k.QuoteVolume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse quoteVolume: %w", err)
		}
	}

	// Trades (index 8)
	if v, ok := arr[8].(float64); ok {
		k.Trades = int64(v)
	} else if v, ok := arr[8].(json.Number); ok {
		if n, err := v.Int64(); err == nil {
			k.Trades = n
		}
	}

	// TakerBuyBaseVolume (index 9)
	if v, ok := arr[9].(string); ok {
		k.TakerBuyBaseVolume = types.ExDecimal{}
		if err := k.TakerBuyBaseVolume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse takerBuyBaseVolume: %w", err)
		}
	}

	// TakerBuyQuoteVolume (index 10)
	if v, ok := arr[10].(string); ok {
		k.TakerBuyQuoteVolume = types.ExDecimal{}
		if err := k.TakerBuyQuoteVolume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse takerBuyQuoteVolume: %w", err)
		}
	}

	// Ignore (index 11)
	if v, ok := arr[11].(string); ok {
		k.Ignore = types.ExDecimal{}
		_ = k.Ignore.UnmarshalJSON([]byte(`"` + v + `"`))
	}

	return nil
}
