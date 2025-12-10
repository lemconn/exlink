package gate

import (
	"sync"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
)

// Gate Gate 交易所实现
type Gate struct {
	client              *Client
	signer              *Signer
	spot                *GateSpot
	perp                *GatePerp
	spotMarketsBySymbol map[string]*model.Market // 现货市场信息（标准化格式索引）
	spotMarketsByID     map[string]*model.Market // 现货市场信息（原始格式索引）
	perpMarketsBySymbol map[string]*model.Market // 合约市场信息（标准化格式索引）
	perpMarketsByID     map[string]*model.Market // 合约市场信息（原始格式索引）
	mu                  sync.RWMutex             // 保护市场信息的读写锁
}

// NewGate 创建 Gate 交易所实例
func NewGate(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)

	gate := &Gate{
		client:              client,
		signer:              signer,
		spotMarketsBySymbol: make(map[string]*model.Market),
		spotMarketsByID:     make(map[string]*model.Market),
		perpMarketsBySymbol: make(map[string]*model.Market),
		perpMarketsByID:     make(map[string]*model.Market),
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
