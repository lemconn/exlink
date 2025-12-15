package option

type SpotOrderSide string

const (
	// Buy 买入
	Buy  SpotOrderSide = "BUY"	
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