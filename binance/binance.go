package binance

import (
	"sync"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/model"
)

// Binance Binance 交易所实现
type Binance struct {
	client              *Client
	signer              *Signer
	spot                *BinanceSpot
	perp                *BinancePerp
	spotMarketsBySymbol map[string]*model.Market // 现货市场信息（标准化格式索引）
	spotMarketsByID     map[string]*model.Market // 现货市场信息（原始格式索引）
	perpMarketsBySymbol map[string]*model.Market // 合约市场信息（标准化格式索引）
	perpMarketsByID     map[string]*model.Market // 合约市场信息（原始格式索引）
	mu                  sync.RWMutex             // 保护市场信息的读写锁
}

// NewBinance 创建 Binance 交易所实例
func NewBinance(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)

	binance := &Binance{
		client:              client,
		signer:              signer,
		spotMarketsBySymbol: make(map[string]*model.Market),
		spotMarketsByID:     make(map[string]*model.Market),
		perpMarketsBySymbol: make(map[string]*model.Market),
		perpMarketsByID:     make(map[string]*model.Market),
	}

	// 初始化现货和合约实现
	binance.spot = NewBinanceSpot(binance)
	binance.perp = NewBinancePerp(binance)

	return binance, nil
}

// Spot 返回现货交易接口
func (b *Binance) Spot() exchange.SpotExchange {
	return b.spot
}

// Perp 返回永续合约交易接口
func (b *Binance) Perp() exchange.PerpExchange {
	return b.perp
}

// Name 返回交易所名称
func (b *Binance) Name() string {
	return binanceName
}
