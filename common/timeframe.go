package common

import "strings"

// TimeframeMap 时间框架映射表
var TimeframeMap = map[string]string{
	"1s":  "1s",
	"10s": "10s",
	"1m":  "1m",
	"3m":  "3m",
	"5m":  "5m",
	"15m": "15m",
	"30m": "30m",
	"1h":  "1h",
	"2h":  "2h",
	"4h":  "4h",
	"6h":  "6h",
	"8h":  "8h",
	"12h": "12h",
	"1d":  "1d",
	"2d":  "2d",
	"3d":  "3d",
	"7d":  "1w",  // 7天标准化为1周
	"30d": "1M",  // 30天标准化为1月
	"1w":  "1w",
	"1M":  "1M",
	"3M":  "3M",
}

// NormalizeTimeframe 标准化时间框架
func NormalizeTimeframe(timeframe string) string {
	timeframe = strings.ToLower(timeframe)
	if normalized, ok := TimeframeMap[timeframe]; ok {
		return normalized
	}
	return timeframe
}

// BinanceTimeframe 转换为Binance时间框架格式
func BinanceTimeframe(timeframe string) string {
	normalized := NormalizeTimeframe(timeframe)
	// Binance使用相同格式，直接返回
	return normalized
}

// OKXTimeframe 转换为OKX时间框架格式
func OKXTimeframe(timeframe string) string {
	normalized := NormalizeTimeframe(timeframe)
	// OKX需要大写某些时间框架
	switch normalized {
	case "1h":
		return "1H"
	case "2h":
		return "2H"
	case "4h":
		return "4H"
	case "6h":
		return "6H"
	case "12h":
		return "12H"
	case "1d":
		return "1D"
	case "2d":
		return "2D"
	case "3d":
		return "3D"
	case "1w":
		return "1W"
	case "1M":
		return "1M"
	case "3M":
		return "3M"
	default:
		return normalized
	}
}

// BybitTimeframe 转换为Bybit时间框架格式
// Bybit v5 API 使用特殊格式：1m->1, 3m->3, 5m->5, 15m->15, 30m->30, 1h->60, 2h->120, 4h->240, 6h->360, 12h->720, 1d->D, 1w->W, 1M->M
func BybitTimeframe(timeframe string) string {
	normalized := NormalizeTimeframe(timeframe)
	switch normalized {
	case "1m":
		return "1"
	case "3m":
		return "3"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h":
		return "60"
	case "2h":
		return "120"
	case "4h":
		return "240"
	case "6h":
		return "360"
	case "12h":
		return "720"
	case "1d":
		return "D"
	case "1w":
		return "W"
	case "1M":
		return "M"
	default:
		return normalized
	}
}

// GateTimeframe 转换为Gate时间框架格式
// Gate需要将1w转换为7d，1M转换为30d
func GateTimeframe(timeframe string) string {
	normalized := NormalizeTimeframe(timeframe)
	switch normalized {
	case "1w":
		return "7d"
	case "1M":
		return "30d"
	default:
		return normalized
	}
}
