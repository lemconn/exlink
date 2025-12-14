package exchange

import (
	"context"

	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/option"
	"github.com/lemconn/exlink/types"
)

// SpotExchange 现货交易接口
type SpotExchange interface {
	// ========== 市场数据 ==========

	// LoadMarkets 加载市场信息
	LoadMarkets(ctx context.Context, reload bool) error

	// FetchMarkets 获取市场列表
	FetchMarkets(ctx context.Context) ([]*model.Market, error)

	// GetMarket 获取单个市场信息
	GetMarket(symbol string) (*model.Market, error)

	// GetMarkets 从内存中获取所有市场信息
	GetMarkets() ([]*model.Market, error)

	// FetchTicker 获取行情（单个）
	FetchTicker(ctx context.Context, symbol string) (*model.Ticker, error)

	// FetchTickers 批量获取行情
	FetchTickers(ctx context.Context) (map[string]*model.Ticker, error)

	// FetchOrderBook 获取订单簿
	// FetchOrderBook(ctx context.Context, symbol string, limit ...int) (*types.OrderBook, error)
	// TODO: 添加 OrderBook 类型到 types 包后启用

	// FetchOHLCVs 获取K线数据
	FetchOHLCVs(ctx context.Context, symbol string, timeframe string, opts ...option.ArgsOption) (model.OHLCVs, error)

	// ========== 账户信息 ==========

	// FetchBalance 获取余额
	FetchBalance(ctx context.Context) (model.Balances, error)

	// ========== 订单操作 ==========

	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, symbol string, side model.OrderSide, opts ...option.ArgsOption) (*model.Order, error)

	// CancelOrder 取消订单
	CancelOrder(ctx context.Context, orderID, symbol string) error

	// FetchOrder 查询订单
	FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error)
}
