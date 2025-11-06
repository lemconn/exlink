package okx

import "github.com/lemconn/exlink"

func init() {
	exlink.Register("okx", NewOKX)
}
