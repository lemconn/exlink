package binance

import (
	"fmt"
	"strings"
)

// ToBinanceSymbol 转换为Binance格式的symbol
// symbol: 标准化格式，如 "BTC/USDT"
// isContract: 是否为合约市场
// 现货: BTC/USDT -> BTCUSDT
// 合约: BTC/USDT -> BTCUSDT
func ToBinanceSymbol(symbol string, isContract bool) (string, error) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s, expected BASE/QUOTE", symbol)
	}
	base := strings.ToUpper(parts[0])
	quote := strings.ToUpper(parts[1])
	return base + quote, nil
}

// ToBinanceSide 将 option 中定义的 side 常量转换为 Binance API 所需的 side 和 positionSide 参数
// 参数:
//   - side: option 包中定义的常量 (Buy, Sell, OpenLong, OpenShort, CloseLong, CloseShort)
//
// 返回:
//   - side: Binance API 的 side 参数 (BUY 或 SELL)
//   - positionSide: Binance API 的 positionSide 参数 (LONG, SHORT, 或 "" 表示现货)
//   - error: 如果 side 参数不合法则返回错误
//
// 映射规则:
//   - Buy -> side=BUY, positionSide="" (现货买入)
//   - Sell -> side=SELL, positionSide="" (现货卖出)
//   - OpenLong -> side=BUY, positionSide=LONG (开多)
//   - OpenShort -> side=SELL, positionSide=SHORT (开空)
//   - CloseLong -> side=SELL, positionSide=LONG (平多)
//   - CloseShort -> side=BUY, positionSide=SHORT (平空)
func ToBinanceSide(side string) (string, string, error) {
	switch side {
	case "BUY":
		// 现货买入
		return "BUY", "", nil
	case "SELL":
		// 现货卖出
		return "SELL", "", nil
	case "OPEN_LONG":
		// 开多: 买入做多
		return "BUY", "LONG", nil
	case "OPEN_SHORT":
		// 开空: 卖出做空
		return "SELL", "SHORT", nil
	case "CLOSE_LONG":
		// 平多: 卖出平掉多头持仓
		return "SELL", "LONG", nil
	case "CLOSE_SHORT":
		// 平空: 买入平掉空头持仓
		return "BUY", "SHORT", nil
	default:
		return "", "", fmt.Errorf("invalid side: %s, expected one of: BUY, SELL, OPEN_LONG, OPEN_SHORT, CLOSE_LONG, CLOSE_SHORT", side)
	}
}
