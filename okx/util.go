package okx

import (
	"fmt"
	"strings"
)

// ToOKXSymbol 转换为OKX格式的symbol
// symbol: 标准化格式，如 "BTC/USDT"
// isContract: 是否为合约市场
// 现货: BTC/USDT -> BTC-USDT
// 合约: BTC/USDT -> BTC-USDT-SWAP
func ToOKXSymbol(symbol string, isContract bool) (string, error) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s, expected BASE/QUOTE", symbol)
	}
	base := strings.ToUpper(parts[0])
	quote := strings.ToUpper(parts[1])

	if isContract {
		return base + "-" + quote + "-SWAP", nil
	}
	return base + "-" + quote, nil
}
