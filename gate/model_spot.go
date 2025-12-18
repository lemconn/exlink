package gate

import (
	"encoding/json"
	"fmt"

	"github.com/lemconn/exlink/types"
)

// gateSpotMarketsResponse Gate 现货市场信息响应
type gateSpotMarketsResponse []gateSpotCurrencyPair

// gateSpotCurrencyPair Gate 现货交易对信息
type gateSpotCurrencyPair struct {
	ID              string          `json:"id"`
	Base            string          `json:"base"`
	Quote           string          `json:"quote"`
	Fee             string          `json:"fee"`
	MinBaseAmount   types.ExDecimal `json:"min_base_amount"`
	MinQuoteAmount  types.ExDecimal `json:"min_quote_amount"`
	MaxQuoteAmount  types.ExDecimal `json:"max_quote_amount"`
	AmountPrecision int             `json:"amount_precision"`
	Precision       int             `json:"precision"`
	TradeStatus     string          `json:"trade_status"`
}

// gateSpotTickerResponse Gate 现货 Ticker 响应
type gateSpotTickerResponse []gateSpotTickerItem

// gateSpotTickerItem Gate 现货 Ticker 数据项
type gateSpotTickerItem struct {
	CurrencyPair     string          `json:"currency_pair"`
	Last             types.ExDecimal `json:"last"`
	LowestAsk        types.ExDecimal `json:"lowest_ask"`
	LowestSize       types.ExDecimal `json:"lowest_size"`
	HighestBid       types.ExDecimal `json:"highest_bid"`
	HighestSize      types.ExDecimal `json:"highest_size"`
	ChangePercentage types.ExDecimal `json:"change_percentage"`
	BaseVolume       types.ExDecimal `json:"base_volume"`
	QuoteVolume      types.ExDecimal `json:"quote_volume"`
	High24h          types.ExDecimal `json:"high_24h"`
	Low24h           types.ExDecimal `json:"low_24h"`
}

// gateSpotKlineResponse Gate 现货 Kline 响应（数组格式）
type gateSpotKlineResponse []gateSpotKline

// gateSpotKline Gate 现货 Kline 数据
type gateSpotKline struct {
	OpenTime     types.ExTimestamp `json:"openTime"`     // Open Time
	QuoteVolume  types.ExDecimal   `json:"quoteVolume"`  // Trading volume in quote currency
	Close        types.ExDecimal   `json:"close"`        // Closing price
	High         types.ExDecimal   `json:"high"`         // Highest price
	Low          types.ExDecimal   `json:"low"`          // Lowest price
	Open         types.ExDecimal   `json:"open"`         // Opening price
	BaseVolume   types.ExDecimal   `json:"baseVolume"`   // Trading volume in base currency
	WindowClosed bool              `json:"windowClosed"` // window is closed
}

// UnmarshalJSON 自定义 JSON 反序列化，解析数组格式
func (k *gateSpotKline) UnmarshalJSON(data []byte) error {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) < 8 {
		return fmt.Errorf("invalid kline array length: %d", len(arr))
	}

	// OpenTime (index 0)
	if v, ok := arr[0].(string); ok {
		k.OpenTime = types.ExTimestamp{}
		if err := k.OpenTime.UnmarshalJSON([]byte(v)); err != nil {
			return fmt.Errorf("parse openTime: %w", err)
		}
	} else if v, ok := arr[0].(float64); ok {
		k.OpenTime = types.ExTimestamp{}
		if err := k.OpenTime.UnmarshalJSON([]byte(fmt.Sprintf("%.0f", v))); err != nil {
			return fmt.Errorf("parse openTime: %w", err)
		}
	}

	// QuoteVolume (index 1)
	if v, ok := arr[1].(string); ok {
		k.QuoteVolume = types.ExDecimal{}
		if err := k.QuoteVolume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse quoteVolume: %w", err)
		}
	}

	// Close (index 2)
	if v, ok := arr[2].(string); ok {
		k.Close = types.ExDecimal{}
		if err := k.Close.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse close: %w", err)
		}
	}

	// High (index 3)
	if v, ok := arr[3].(string); ok {
		k.High = types.ExDecimal{}
		if err := k.High.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse high: %w", err)
		}
	}

	// Low (index 4)
	if v, ok := arr[4].(string); ok {
		k.Low = types.ExDecimal{}
		if err := k.Low.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse low: %w", err)
		}
	}

	// Open (index 5)
	if v, ok := arr[5].(string); ok {
		k.Open = types.ExDecimal{}
		if err := k.Open.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse open: %w", err)
		}
	}

	// BaseVolume (index 6)
	if v, ok := arr[6].(string); ok {
		k.BaseVolume = types.ExDecimal{}
		if err := k.BaseVolume.UnmarshalJSON([]byte(`"` + v + `"`)); err != nil {
			return fmt.Errorf("parse baseVolume: %w", err)
		}
	}

	// WindowClosed (index 7)
	if v, ok := arr[7].(bool); ok {
		k.WindowClosed = v
	} else if v, ok := arr[7].(string); ok {
		k.WindowClosed = (v == "true")
	}

	return nil
}

// gateSpotBalanceResponse Gate 现货余额响应
type gateSpotBalanceResponse []gateSpotBalanceItem

// gateSpotBalanceItem Gate 现货余额项
type gateSpotBalanceItem struct {
	Currency  string          `json:"currency"`
	Available types.ExDecimal `json:"available"`
	Locked    types.ExDecimal `json:"locked"`
}

// gateSpotCreateOrderResponse Gate 现货创建订单响应
type gateSpotCreateOrderResponse struct {
	ID           string            `json:"id"`
	Text         string            `json:"text"`
	CreateTime   types.ExTimestamp `json:"create_time"`
	CreateTimeMs types.ExTimestamp `json:"create_time_ms"`
	CurrencyPair string            `json:"currency_pair"`
	Type         string            `json:"type"`
	Account      string            `json:"account"`
	Side         string            `json:"side"`
	Amount       types.ExDecimal   `json:"amount"`
	Price        types.ExDecimal   `json:"price"`
	TimeInForce  string            `json:"time_in_force"`
	Fee          types.ExDecimal   `json:"fee,omitempty"`
	FeeCurrency  string            `json:"fee_currency,omitempty"`
	FinishAs     string            `json:"finish_as,omitempty"`
}

// gateSpotFetchOrderResponse Gate 现货查询订单响应
type gateSpotFetchOrderResponse struct {
	ID             string            `json:"id"`               // 订单ID
	Text           string            `json:"text"`             // 客户端订单ID
	AmendText      string            `json:"amend_text"`       // 修改文本
	CreateTime     types.ExTimestamp `json:"create_time"`      // 创建时间（秒）
	UpdateTime     types.ExTimestamp `json:"update_time"`      // 更新时间（秒）
	CreateTimeMs   types.ExTimestamp `json:"create_time_ms"`   // 创建时间（毫秒）
	UpdateTimeMs   types.ExTimestamp `json:"update_time_ms"`   // 更新时间（毫秒）
	Status         string            `json:"status"`           // 订单状态
	CurrencyPair   string            `json:"currency_pair"`    // 交易对
	Type           string            `json:"type"`             // 订单类型
	Account        string            `json:"account"`          // 账户类型
	Side           string            `json:"side"`             // 订单方向
	Amount         types.ExDecimal   `json:"amount"`           // 订单数量
	Price          types.ExDecimal   `json:"price"`            // 订单价格
	TimeInForce    string            `json:"time_in_force"`    // 时间有效性
	Iceberg        types.ExDecimal   `json:"iceberg"`          // 冰山订单显示数量
	Left           types.ExDecimal   `json:"left"`             // 剩余数量
	FilledAmount   types.ExDecimal   `json:"filled_amount"`    // 已成交数量
	FillPrice      types.ExDecimal   `json:"fill_price"`       // 成交价格
	FilledTotal    types.ExDecimal   `json:"filled_total"`     // 成交总额
	Fee            types.ExDecimal   `json:"fee"`              // 手续费
	FeeCurrency    string            `json:"fee_currency"`     // 手续费币种
	PointFee       types.ExDecimal   `json:"point_fee"`        // 积分手续费
	GtFee          types.ExDecimal   `json:"gt_fee"`           // GT抵扣手续费
	GtMakerFee     types.ExDecimal   `json:"gt_maker_fee"`     // GT Maker手续费
	GtTakerFee     types.ExDecimal   `json:"gt_taker_fee"`     // GT Taker手续费
	GtDiscount     bool              `json:"gt_discount"`      // 是否使用GT抵扣
	RebatedFee     types.ExDecimal   `json:"rebated_fee"`      // 返还手续费
	RebatedFeeCcy  string            `json:"rebated_fee_currency"` // 返还手续费币种
	FinishAs       string            `json:"finish_as"`        // 订单完成方式
}
