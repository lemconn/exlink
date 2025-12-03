package binance

import (
	"context"
	"time"

	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/types"
)

// BinanceSpot Binance 现货实现
type BinanceSpot struct {
	binance *Binance
	market  *binanceSpotMarket
	order   *binanceSpotOrder
}

// NewBinanceSpot 创建 Binance 现货实例
func NewBinanceSpot(b *Binance) *BinanceSpot {
	return &BinanceSpot{
		binance: b,
		market:  &binanceSpotMarket{binance: b},
		order:   &binanceSpotOrder{binance: b},
	}
}

// ========== SpotExchange 接口实现 ==========

// LoadMarkets 加载市场信息
func (s *BinanceSpot) LoadMarkets(ctx context.Context, reload bool) error {
	return s.market.LoadMarkets(ctx, reload)
}

// FetchMarkets 获取市场列表
func (s *BinanceSpot) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	return s.market.FetchMarkets(ctx)
}

// GetMarket 获取单个市场信息
func (s *BinanceSpot) GetMarket(symbol string) (*types.Market, error) {
	return s.market.GetMarket(symbol)
}

// FetchTicker 获取行情（单个）
func (s *BinanceSpot) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	return s.market.FetchTicker(ctx, symbol)
}

// FetchTickers 批量获取行情
func (s *BinanceSpot) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	return s.market.FetchTickers(ctx, symbols...)
}

// FetchOHLCV 获取K线数据
func (s *BinanceSpot) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	return s.market.FetchOHLCV(ctx, symbol, timeframe, since, limit)
}

// FetchBalance 获取余额
func (s *BinanceSpot) FetchBalance(ctx context.Context) (types.Balances, error) {
	return s.order.FetchBalance(ctx)
}

// CreateOrder 创建订单
func (s *BinanceSpot) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	return s.order.CreateOrder(ctx, symbol, side, amount, opts...)
}

// CancelOrder 取消订单
func (s *BinanceSpot) CancelOrder(ctx context.Context, orderID, symbol string) error {
	return s.order.CancelOrder(ctx, orderID, symbol)
}

// FetchOrder 查询订单
func (s *BinanceSpot) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	return s.order.FetchOrder(ctx, orderID, symbol)
}

// FetchOrders 查询订单列表
func (s *BinanceSpot) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	return s.order.FetchOrders(ctx, symbol, since, limit)
}

// FetchOpenOrders 查询未成交订单
func (s *BinanceSpot) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	return s.order.FetchOpenOrders(ctx, symbol)
}

// FetchTrades 获取交易记录（公共）
func (s *BinanceSpot) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchTrades(ctx, symbol, since, limit)
}

// FetchMyTrades 获取我的交易记录
func (s *BinanceSpot) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	return s.order.FetchMyTrades(ctx, symbol, since, limit)
}

// 确保 BinanceSpot 实现了 exchange.SpotExchange 接口
var _ exchange.SpotExchange = (*BinanceSpot)(nil)

// ========== 内部实现 ==========

// binanceSpotMarket 现货市场相关方法
type binanceSpotMarket struct {
	binance *Binance
}

// LoadMarkets 加载市场信息
func (m *binanceSpotMarket) LoadMarkets(ctx context.Context, reload bool) error {
	// TODO: 实现加载市场信息
	return nil
}

// FetchMarkets 获取市场列表
func (m *binanceSpotMarket) FetchMarkets(ctx context.Context) ([]*types.Market, error) {
	// TODO: 实现获取市场列表
	return nil, nil
}

// GetMarket 获取单个市场信息
func (m *binanceSpotMarket) GetMarket(symbol string) (*types.Market, error) {
	// TODO: 实现获取单个市场信息
	return nil, nil
}

// FetchTicker 获取行情（单个）
func (m *binanceSpotMarket) FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	// TODO: 实现获取行情
	return nil, nil
}

// FetchTickers 批量获取行情
func (m *binanceSpotMarket) FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error) {
	// TODO: 实现批量获取行情
	return nil, nil
}

// FetchOHLCV 获取K线数据
func (m *binanceSpotMarket) FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) {
	// TODO: 实现获取K线数据
	return nil, nil
}

// binanceSpotOrder 现货订单相关方法
type binanceSpotOrder struct {
	binance *Binance
}

// FetchBalance 获取余额
func (o *binanceSpotOrder) FetchBalance(ctx context.Context) (types.Balances, error) {
	// TODO: 实现获取余额
	return nil, nil
}

// CreateOrder 创建订单
func (o *binanceSpotOrder) CreateOrder(ctx context.Context, symbol string, side types.OrderSide, amount string, opts ...types.OrderOption) (*types.Order, error) {
	// TODO: 实现创建订单
	return nil, nil
}

// CancelOrder 取消订单
func (o *binanceSpotOrder) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// TODO: 实现取消订单
	return nil
}

// FetchOrder 查询订单
func (o *binanceSpotOrder) FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error) {
	// TODO: 实现查询订单
	return nil, nil
}

// FetchOrders 查询订单列表
func (o *binanceSpotOrder) FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error) {
	// TODO: 实现查询订单列表
	return nil, nil
}

// FetchOpenOrders 查询未成交订单
func (o *binanceSpotOrder) FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error) {
	// TODO: 实现查询未成交订单
	return nil, nil
}

// FetchTrades 获取交易记录（公共）
func (o *binanceSpotOrder) FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现获取交易记录
	return nil, nil
}

// FetchMyTrades 获取我的交易记录
func (o *binanceSpotOrder) FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) {
	// TODO: 实现获取我的交易记录
	return nil, nil
}

