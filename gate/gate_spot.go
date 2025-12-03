package gate

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// GateSpot Gate 现货实现
type GateSpot struct {
	gate   *Gate
	market *gateSpotMarket
	order  *gateSpotOrder
}

// NewGateSpot 创建 Gate 现货实例
func NewGateSpot(g *Gate) *GateSpot {
	return &GateSpot{
		gate:   g,
		market: &gateSpotMarket{gate: g},
		order:  &gateSpotOrder{gate: g},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *GateSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *GateSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *GateSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *GateSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *GateSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *GateSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *GateSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *GateSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *GateSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *GateSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *GateSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *GateSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *GateSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *GateSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*GateSpot)(nil)

// ========== 内部实现 ==========

type gateSpotMarket struct {
	gate *Gate
}

func (m *gateSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *gateSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gateSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gateSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gateSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *gateSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type gateSpotOrder struct {
	gate *Gate
}

func (o *gateSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *gateSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *gateSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

