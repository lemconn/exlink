package okx

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// OKXPerp OKX 永续合约实现
type OKXPerp struct {
	okx       *OKX
	market    *okxPerpMarket
	order     *okxPerpOrder
	hedgeMode bool
}

// NewOKXPerp 创建 OKX 永续合约实例
func NewOKXPerp(o *OKX) *OKXPerp {
	return &OKXPerp{
		okx:       o,
		market:    &okxPerpMarket{okx: o},
		order:     &okxPerpOrder{okx: o},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *OKXPerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *OKXPerp) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *OKXPerp) GetMarket(symbol string) (*types.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *OKXPerp) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *OKXPerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

func (p *OKXPerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (p *OKXPerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *OKXPerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *OKXPerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *OKXPerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

func (p *OKXPerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

func (p *OKXPerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *OKXPerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *OKXPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *OKXPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *OKXPerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *OKXPerp) IsHedgeMode() bool {
	return p.hedgeMode
}

var _ exchange.PerpExchange = (*OKXPerp)(nil)

// ========== 内部实现 ==========

type okxPerpMarket struct {
	okx *OKX
}

func (m *okxPerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *okxPerpMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxPerpMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxPerpMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxPerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxPerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type okxPerpOrder struct {
	okx *OKX
}

func (o *okxPerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *okxPerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxPerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	// TODO: 实现
	return nil
}

func (o *okxPerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// TODO: 实现
	return nil
}

