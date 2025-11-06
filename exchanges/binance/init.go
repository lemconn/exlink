package binance

import "github.com/lemconn/exlink"

func init() {
	exlink.Register("binance", NewBinance)
}
