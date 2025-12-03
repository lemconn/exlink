package okx

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// OKXSpot OKX 现货实现
type OKXSpot struct {
	okx    *OKX
	market *okxSpotMarket
	order  *okxSpotOrder
}

// NewOKXSpot 创建 OKX 现货实例
func NewOKXSpot(o *OKX) *OKXSpot {
	return &OKXSpot{
		okx:    o,
		market: &okxSpotMarket{okx: o},
		order:  &okxSpotOrder{okx: o},
	}
}

// ========== SpotExchange 接口实现 ==========

func (s *OKXSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

func (s *OKXSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

func (s *OKXSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

func (s *OKXSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

func (s *OKXSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

func (s *OKXSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

func (s *OKXSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

func (s *OKXSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

func (s *OKXSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

func (s *OKXSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

func (s *OKXSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

func (s *OKXSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

func (s *OKXSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

func (s *OKXSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

var _ exchange.SpotExchange = (*OKXSpot)(nil)

// ========== 内部实现 ==========

type okxSpotMarket struct {
	okx *OKX
}

func (m *okxSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现
	return nil
}

func (m *okxSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现
	return nil, nil
}

func (m *okxSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现
	return nil, nil
}

type okxSpotOrder struct {
	okx *OKX
}

func (o *okxSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现
	return nil
}

func (o *okxSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

func (o *okxSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现
	return nil, nil
}

