package bybit

import "github.com/lemconn/exlink"

func init() {
	exlink.Register(bybitName, NewBybit)
}
