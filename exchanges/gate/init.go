package gate

import "github.com/lemconn/exlink"

func init() {
	exlink.Register(gateName, NewGate)
}

