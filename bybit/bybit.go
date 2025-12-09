package bybit

import (
	"sync"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
)

// Bybit Bybit 交易所实现
type Bybit struct {
	client      *Client
	signer      *Signer
	spot        *BybitSpot
	perp        *BybitPerp
	spotMarkets map[string]*model.Market // 现货市场信息
	perpMarkets map[string]*model.Market // 合约市场信息
	mu          sync.RWMutex             // 保护市场信息的读写锁
}

// NewBybit 创建 Bybit 交易所实例
func NewBybit(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)
	signer.SetAPIKey(apiKey) // Bybit v5 签名需要 API Key

	bybit := &Bybit{
		client:      client,
		signer:      signer,
		spotMarkets: make(map[string]*model.Market),
		perpMarkets: make(map[string]*model.Market),
	}

	// 初始化现货和合约实现
	bybit.spot = NewBybitSpot(bybit)
	bybit.perp = NewBybitPerp(bybit)

	return bybit, nil
}

// Spot 返回现货交易接口
func (b *Bybit) Spot() exchange.SpotExchange {
	return b.spot
}

// Perp 返回永续合约交易接口
func (b *Bybit) Perp() exchange.PerpExchange {
	return b.perp
}

// Name 返回交易所名称
func (b *Bybit) Name() string {
	return bybitName
}
