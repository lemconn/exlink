package gate

import (
	"github.com/lemconn/exlink/exchange"
)

// Gate Gate 交易所实现
type Gate struct {
	client *Client
	signer *Signer
	spot   *GateSpot
	perp   *GatePerp
}

// NewGate 创建 Gate 交易所实例
func NewGate(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)

	gate := &Gate{
		client: client,
		signer: signer,
	}

	// 初始化现货和合约实现
	gate.spot = NewGateSpot(gate)
	gate.perp = NewGatePerp(gate)

	return gate, nil
}

// Spot 返回现货交易接口
func (g *Gate) Spot() exchange.SpotExchange {
	return g.spot
}

// Perp 返回永续合约交易接口
func (g *Gate) Perp() exchange.PerpExchange {
	return g.perp
}

// Name 返回交易所名称
func (g *Gate) Name() string {
	return gateName
}
