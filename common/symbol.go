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

// ParseContractSymbol 解析合约交易对 (BTC/USDT:USDT -> base, quote, settle)
func ParseContractSymbol(symbol string) (base, quote, settle string, err error) {
	// 检查是否包含结算货币（合约格式）
	if strings.Contains(symbol, ":") {
		parts := strings.Split(symbol, ":")
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid contract symbol format: %s, expected BASE/QUOTE:SETTLE", symbol)
		}
		base, quote, err = ParseSymbol(parts[0])
		if err != nil {
			return "", "", "", err
		}
		settle = strings.ToUpper(parts[1])
		return base, quote, settle, nil
	}
	// 非合约格式，只解析 base 和 quote
	base, quote, err = ParseSymbol(symbol)
	if err != nil {
		return "", "", "", err
	}
	return base, quote, "", nil
}

// ToBinanceSymbol 转换为Binance格式
// 现货: BTC/USDT -> BTCUSDT
// 合约: BTC/USDT:USDT -> BTCUSDT
func ToBinanceSymbol(symbol string) (string, error) {
	base, quote, _, err := ParseContractSymbol(symbol)
	if err != nil {
		return "", err
	}
	return base + quote, nil
}

// ToOKXSymbol 转换为OKX格式
// 现货: BTC/USDT -> BTC-USDT
// 合约: BTC/USDT:USDT -> BTC-USDT-SWAP
func ToOKXSymbol(symbol string) (string, error) {
	base, quote, settle, err := ParseContractSymbol(symbol)
	if err != nil {
		return "", err
	}
	// 如果是合约，添加 -SWAP 后缀
	if settle != "" {
		return base + "-" + quote + "-SWAP", nil
	}
	return base + "-" + quote, nil
}

// ToGateSymbol 转换为Gate格式
// 现货: BTC/USDT -> BTC_USDT
// 合约: BTC/USDT:USDT -> BTC_USDT
func ToGateSymbol(symbol string) (string, error) {
	base, quote, _, err := ParseContractSymbol(symbol)
	if err != nil {
		return "", err
	}
	return base + "_" + quote, nil
}

// ToBybitSymbol 转换为Bybit格式
// 现货: BTC/USDT -> BTCUSDT
// 合约: BTC/USDT:USDT -> BTCUSDT
func ToBybitSymbol(symbol string) (string, error) {
	base, quote, _, err := ParseContractSymbol(symbol)
	if err != nil {
		return "", err
	}
	return base + quote, nil
}
