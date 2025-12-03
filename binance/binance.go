package binance

import (
	"github.com/lemconn/exlink/exchange"
)

// Binance Binance 交易所实现
type Binance struct {
	client *Client
	signer *Signer
	spot   *BinanceSpot
	perp   *BinancePerp
}

// NewBinance 创建 Binance 交易所实例
func NewBinance(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	client, err := NewClient(apiKey, secretKey, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey)

	binance := &Binance{
		client: client,
		signer: signer,
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
