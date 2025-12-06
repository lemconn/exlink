package gate

import (
	"sync"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// Gate Gate 交易所实现
type Gate struct {
	client      *Client
	signer      *Signer
	spot        *GateSpot
	perp        *GatePerp
	spotMarkets map[string]*types.Market // 现货市场信息
	perpMarkets map[string]*types.Market // 合约市场信息
	mu          sync.RWMutex             // 保护市场信息的读写锁
}

// NewGate 创建 Gate 交易所实例
func NewGate(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)

	gate := &Gate{
		client:      client,
		signer:      signer,
		spotMarkets: make(map[string]*types.Market),
		perpMarkets: make(map[string]*types.Market),
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
