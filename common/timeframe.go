package common

import "strings"

// TimeframeMap 时间框架映射表
var TimeframeMap = map[string]string{
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
	"3d":  "3d",
	"1w":  "1w",
	"1M":  "1M",
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
