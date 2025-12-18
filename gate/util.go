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

// getPrecisionDigits 计算精度位数
func getPrecisionDigits(value float64) int {
	if value == 0 {
		return 8
	}
	str := fmt.Sprintf("%.10f", value)
	str = strings.TrimRight(str, "0")
	parts := strings.Split(str, ".")
	if len(parts) == 2 {
		return len(parts[1])
	}
	return 0
}

//nolint:unused // getString 从 map 中获取字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}
