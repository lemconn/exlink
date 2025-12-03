package bybit

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// BybitPerp Bybit 永续合约实现
type BybitPerp struct {
	bybit     *Bybit
	market    *bybitPerpMarket
	order     *bybitPerpOrder
	hedgeMode bool
}

// NewBybitPerp 创建 Bybit 永续合约实例
func NewBybitPerp(b *Bybit) *BybitPerp {
	return &BybitPerp{
		bybit:     b,
		market:    &bybitPerpMarket{bybit: b},
		order:     &bybitPerpOrder{bybit: b},
		hedgeMode: false,
	}
}

// ========== PerpExchange 接口实现 ==========

func (p *BybitPerp) LoadMarkets(ctx context.Context, reload bool) error {
	return p.market.LoadMarkets(ctx, reload)
}

func (p *BybitPerp) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return p.market.FetchMarkets(ctx)
}

func (p *BybitPerp) GetMarket(symbol string) (*types.Market, error) {
	return p.market.GetMarket(symbol)
}

func (p *BybitPerp) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return p.market.FetchTicker(ctx, symbol)
}

func (p *BybitPerp) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return p.market.FetchTickers(ctx, symbols...)
}

func (p *BybitPerp) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return p.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (p *BybitPerp) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	return p.order.FetchPositions(ctx, symbols...)
}

func (p *BybitPerp) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return p.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (p *BybitPerp) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return p.order.CancelOrder(ctx, orderID, symbol)
}

func (p *BybitPerp) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return p.order.FetchOrder(ctx, orderID, symbol)
}

func (p *BybitPerp) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return p.order.FetchOrders(ctx, symbol, since, limit)
}

func (p *BybitPerp) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return p.order.FetchOpenOrders(ctx, symbol)
}

func (p *BybitPerp) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchTrades(ctx, symbol, since, limit)
}

func (p *BybitPerp) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return p.order.FetchMyTrades(ctx, symbol, since, limit)
}

func (p *BybitPerp) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return p.order.SetLeverage(ctx, symbol, leverage)
}

func (p *BybitPerp) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	return p.order.SetMarginMode(ctx, symbol, mode)
}

func (p *BybitPerp) SetHedgeMode(hedgeMode bool) {
	p.hedgeMode = hedgeMode
}

func (p *BybitPerp) IsHedgeMode() bool {
	return p.hedgeMode
}

var _ exchange.PerpExchange = (*BybitPerp)(nil)

// ========== 内部实现 ==========

type bybitPerpMarket struct {
	bybit *Bybit
}

func (m *bybitPerpMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *bybitPerpMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitPerpMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitPerpMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitPerpMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitPerpMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type bybitPerpOrder struct {
	bybit *Bybit
}

func (o *bybitPerpOrder) FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *bybitPerpOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitPerpOrder) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	// TODO: 实现
	return nil
}

func (o *bybitPerpOrder) SetMarginMode(ctx context.Context, symbol string, mode string) error {
	// TODO: 实现
	return nil
}

