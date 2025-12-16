package option

import (
	"strings"
)

type SpotOrderSide string

const (
	// Buy 买入
	Buy SpotOrderSide = "BUY"
	// Sell 卖出
	Sell SpotOrderSide = "SELL"
)

func (o SpotOrderSide) ToSide() string {
	switch o {
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	}
	return ""
}

type PerpOrderSide string

const (
	// OpenLong 开多
	OpenLong PerpOrderSide = "OPEN_LONG"
	// OpenShort 开空
	OpenShort PerpOrderSide = "OPEN_SHORT"
	// CloseLong 平多
	CloseLong PerpOrderSide = "CLOSE_LONG"
	// CloseShort 平空
	CloseShort PerpOrderSide = "CLOSE_SHORT"
)

func (o PerpOrderSide) ToSide() string {
	switch o {
	case OpenLong:
		return "BUY"
	case OpenShort:
		return "SELL"
	case CloseLong:
		return "SELL"
	case CloseShort:
		return "BUY"
	}
	return ""
}

func (o PerpOrderSide) ToPositionSide() string {
	switch o {
	case OpenLong:
		return "LONG"
	case OpenShort:
		return "SHORT"
	case CloseLong:
		return "LONG"
	case CloseShort:
		return "SHORT"
	}
	return ""
}

func (o PerpOrderSide) ToReduceOnly() bool {
	switch o {
	case CloseLong:
		return true
	case CloseShort:
		return true
	}
	return false
}

// OrderType 订单类型
type OrderType string

const (
	// Market 市价单
	Market OrderType = "MARKET"
	// Limit 限价单
	Limit OrderType = "LIMIT"
)

// String 返回字符串表示
func (t OrderType) String() string {
	return string(t)
}

// Upper 返回大写字符串
func (t OrderType) Upper() string {
	return string(t)
}

// Lower 返回小写字符串
func (t OrderType) Lower() string {
	return strings.ToLower(string(t))
}

// Capitalize 返回首字母大写的字符串
func (t OrderType) Capitalize() string {
	s := strings.ToLower(string(t))
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// IsMarket 判断是否为市价单
func (t OrderType) IsMarket() bool {
	return t == Market
}

// IsLimit 判断是否为限价单
func (t OrderType) IsLimit() bool {
	return t == Limit
}

// TimeInForce 订单有效期类型
type TimeInForce string

const (
	// GTC Good Till Cancel 成交为止（下单后仅有1年有效期，1年后自动取消）
	GTC TimeInForce = "GTC"
	// IOC Immediate or Cancel 无法立即成交(吃单)的部分就撤销
	IOC TimeInForce = "IOC"
	// FOK Fill or Kill 无法全部立即成交就撤销
	FOK TimeInForce = "FOK"
)

// String 返回字符串表示
func (t TimeInForce) String() string {
	return string(t)
}

// Upper 返回大写字符串
func (t TimeInForce) Upper() string {
	return string(t)
}

// Lower 返回小写字符串
func (t TimeInForce) Lower() string {
	return strings.ToLower(string(t))
}

// IsGTC 判断是否为 GTC（成交为止）
func (t TimeInForce) IsGTC() bool {
	return t == GTC
}

// IsIOC 判断是否为 IOC（无法立即成交的部分就撤销）
func (t TimeInForce) IsIOC() bool {
	return t == IOC
}

// IsFOK 判断是否为 FOK（无法全部立即成交就撤销）
func (t TimeInForce) IsFOK() bool {
	return t == FOK
}

// MarginType 保证金类型
type MarginType string

const (
	// ISOLATED 逐仓保证金
	ISOLATED MarginType = "ISOLATED"
	// CROSSED 全仓保证金
	CROSSED MarginType = "CROSSED"
)

// String 返回字符串表示
func (m MarginType) String() string {
	return string(m)
}

// Upper 返回大写字符串
func (m MarginType) Upper() string {
	return string(m)
}

// Lower 返回小写字符串
func (m MarginType) Lower() string {
	return strings.ToLower(string(m))
}

// IsIsolated 判断是否为逐仓保证金
func (m MarginType) IsIsolated() bool {
	return m == ISOLATED
}

// IsCrossed 判断是否为全仓保证金
func (m MarginType) IsCrossed() bool {
	return m == CROSSED
}
