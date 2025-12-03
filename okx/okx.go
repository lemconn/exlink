package okx

import (
	"github.com/lemconn/exlink/exchange"
)

// OKX OKX 交易所实现
type OKX struct {
	client *Client
	signer *Signer
	spot   *OKXSpot
	perp   *OKXPerp
}

// NewOKX 创建 OKX 交易所实例
func NewOKX(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error) {
	passphrase := ""
	if v, ok := options["password"].(string); ok {
		passphrase = v
	}

	client, err := NewClient(apiKey, secretKey, passphrase, options)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(secretKey, passphrase)

	okx := &OKX{
		client: client,
		signer: signer,
	}

	// 初始化现货和合约实现
	okx.spot = NewOKXSpot(okx)
	okx.perp = NewOKXPerp(okx)

	return okx, nil
}

// Spot 返回现货交易接口
func (o *OKX) Spot() exchange.SpotExchange {
	return o.spot
}

// Perp 返回永续合约交易接口
func (o *OKX) Perp() exchange.PerpExchange {
	return o.perp
}

// Name 返回交易所名称
func (o *OKX) Name() string {
	return okxName
}
