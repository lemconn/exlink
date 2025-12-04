package model

// Timeframe 时间框架类型
type Timeframe string

const (
	// Timeframe1m 1分钟
	Timeframe1m Timeframe = "1m"
	// Timeframe3m 3分钟
	Timeframe3m Timeframe = "3m"
	// Timeframe5m 5分钟
	Timeframe5m Timeframe = "5m"
	// Timeframe15m 15分钟
	Timeframe15m Timeframe = "15m"
	// Timeframe30m 30分钟
	Timeframe30m Timeframe = "30m"
	// Timeframe1h 1小时
	Timeframe1h Timeframe = "1h"
	// Timeframe2h 2小时
	Timeframe2h Timeframe = "2h"
	// Timeframe4h 4小时
	Timeframe4h Timeframe = "4h"
	// Timeframe6h 6小时
	Timeframe6h Timeframe = "6h"
	// Timeframe8h 8小时
	Timeframe8h Timeframe = "8h"
	// Timeframe12h 12小时
	Timeframe12h Timeframe = "12h"
	// Timeframe1d 1天
	Timeframe1d Timeframe = "1d"
	// Timeframe3d 3天
	Timeframe3d Timeframe = "3d"
	// Timeframe1w 1周
	Timeframe1w Timeframe = "1w"
	// Timeframe1M 1月
	Timeframe1M Timeframe = "1M"
)

// ToBinance 转换为Binance格式的timeframe
// Binance使用标准格式，直接返回
func (t Timeframe) ToBinance() string {
	return string(t)
}

// ToOKX 转换为OKX格式的timeframe
// OKX需要大写某些时间框架
func (t Timeframe) ToOKX() string {
	switch t {
	case Timeframe1h:
		return "1H"
	case Timeframe2h:
		return "2H"
	case Timeframe4h:
		return "4H"
	case Timeframe6h:
		return "6H"
	case Timeframe12h:
		return "12H"
	case Timeframe1d:
		return "1D"
	case Timeframe1w:
		return "1W"
	case Timeframe1M:
		return "1M"
	case "3M":
		return "3M"
	default:
		return string(t)
	}
}

// ToBybit 转换为Bybit格式的timeframe
// Bybit v5 API 使用特殊格式：1m->1, 3m->3, 5m->5, 15m->15, 30m->30, 1h->60, 2h->120, 4h->240, 6h->360, 12h->720, 1d->D, 1w->W, 1M->M
func (t Timeframe) ToBybit() string {
	switch t {
	case Timeframe1m:
		return "1"
	case Timeframe3m:
		return "3"
	case Timeframe5m:
		return "5"
	case Timeframe15m:
		return "15"
	case Timeframe30m:
		return "30"
	case Timeframe1h:
		return "60"
	case Timeframe2h:
		return "120"
	case Timeframe4h:
		return "240"
	case Timeframe6h:
		return "360"
	case Timeframe12h:
		return "720"
	case Timeframe1d:
		return "D"
	case Timeframe1w:
		return "W"
	case Timeframe1M:
		return "M"
	default:
		return string(t)
	}
}

// ToGate 转换为Gate格式的timeframe
// Gate使用标准格式，直接返回
func (t Timeframe) ToGate() string {
	return string(t)
}
