package common

import (
	"fmt"
	"strings"
)

// NormalizeSymbol 标准化交易对格式为 BASE/QUOTE (如 BTC/USDT)
func NormalizeSymbol(base, quote string) string {
	return strings.ToUpper(base) + "/" + strings.ToUpper(quote)
}

// NormalizeContractSymbol 标准化合约交易对格式 BASE/QUOTE:SETTLE (如 BTC/USDT:USDT)
func NormalizeContractSymbol(base, quote, settle string) string {
	// 对于合约市场，总是包含结算货币
	if settle != "" {
		return strings.ToUpper(base) + "/" + strings.ToUpper(quote) + ":" + strings.ToUpper(settle)
	}
	return NormalizeSymbol(base, quote)
}

// ParseSymbol 解析标准化交易对 (BTC/USDT -> base, quote)
func ParseSymbol(symbol string) (base, quote string, err error) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid symbol format: %s, expected BASE/QUOTE", symbol)
	}
	return strings.ToUpper(parts[0]), strings.ToUpper(parts[1]), nil
}

// ToBinanceSymbol 转换为Binance格式 (BTC/USDT -> BTCUSDT)
func ToBinanceSymbol(symbol string) (string, error) {
	base, quote, err := ParseSymbol(symbol)
	if err != nil {
		return "", err
	}
	return base + quote, nil
}

// ToOKXSymbol 转换为OKX格式 (BTC/USDT -> BTC-USDT)
func ToOKXSymbol(symbol string) (string, error) {
	base, quote, err := ParseSymbol(symbol)
	if err != nil {
		return "", err
	}
	return base + "-" + quote, nil
}

// FromOKXSymbol 从OKX格式转换 (BTC-USDT -> BTC/USDT)
func FromOKXSymbol(symbol string) string {
	parts := strings.Split(symbol, "-")
	if len(parts) == 2 {
		return NormalizeSymbol(parts[0], parts[1])
	}
	return symbol
}
