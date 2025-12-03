package gate

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// GatePerp Gate 永续合约实现
type GatePerp struct {
	gate      *Gate
	market    *gatePerpMarket
	order     *gatePerpOrder
	hedgeMode bool
}

// NewGatePerp 创建 Gate 永续合约实例
func NewGatePerp(g *Gate) *GatePerp {
	return &GatePerp{
		gate:      g,
		market:    &gatePerpMarket{gate: g},
		order:     &gatePerpOrder{gate: g},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *GatePerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *GatePerp) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *GatePerp) GetMarket(symbol string) (*types.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *GatePerp) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *GatePerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

func (p *GatePerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (p *GatePerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *GatePerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *GatePerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *GatePerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *GatePerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

func (p *GatePerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

func (p *GatePerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *GatePerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *GatePerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *GatePerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *GatePerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *GatePerp) IsHedgeMode() bool {
	return p.hedgeMode
}

var _ exchange.PerpExchange = (*GatePerp)(nil)

// ========== 内部实现 ==========

type gatePerpMarket struct {
	gate *Gate
}

func (m *gatePerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *gatePerpMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gatePerpMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gatePerpMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gatePerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gatePerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type gatePerpOrder struct {
	gate *Gate
}

func (o *gatePerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *gatePerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gatePerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	// TODO: 实现
	return nil
}

func (o *gatePerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// TODO: 实现
	return nil
}

