package gate

import (
	"fmt"
	"strings"
)

// ToGateSymbol 转换为Gate格式的symbol
// symbol: 标准化格式，如 "BTC/USDT"
// isContract: 是否为合约市场
// 现货: BTC/USDT -> BTC_USDT
// 合约: BTC/USDT -> BTC_USDT
func ToGateSymbol(symbol string, isContract bool) (string, error) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s, expected BASE/QUOTE", symbol)
	}
	base := strings.ToUpper(parts[0])
	quote := strings.ToUpper(parts[1])
	return base + "_" + quote, nil
}

