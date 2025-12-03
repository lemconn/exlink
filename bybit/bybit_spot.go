package bybit

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// BybitSpot Bybit 现货实现
type BybitSpot struct {
	bybit  *Bybit
	market *bybitSpotMarket
	order  *bybitSpotOrder
}

// NewBybitSpot 创建 Bybit 现货实例
func NewBybitSpot(b *Bybit) *BybitSpot {
	return &BybitSpot{
		bybit:  b,
		market: &bybitSpotMarket{bybit: b},
		order:  &bybitSpotOrder{bybit: b},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *BybitSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *BybitSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *BybitSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *BybitSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *BybitSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *BybitSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *BybitSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *BybitSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *BybitSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *BybitSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *BybitSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *BybitSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *BybitSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *BybitSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*BybitSpot)(nil)

// ========== 内部实现 ==========

type bybitSpotMarket struct {
	bybit *Bybit
}

func (m *bybitSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *bybitSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *bybitSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type bybitSpotOrder struct {
	bybit *Bybit
}

func (o *bybitSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *bybitSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *bybitSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

