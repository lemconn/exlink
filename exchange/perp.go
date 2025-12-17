package exchange

import (
	"context"

	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/option"
)

// PerpExchange 永续合约交易接口
type PerpExchange interface {
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

	// FetchPositions 获取持仓
	FetchPositions(ctx context.Context, opts ...option.ArgsOption) (model.Positions, error)

	// ========== 订单操作 ==========

	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, symbol string, side option.PerpOrderSide, amount string, opts ...option.ArgsOption) (*model.NewOrder, error)

	// CancelOrder 取消订单
	CancelOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) error

	// FetchOrder 查询订单
	FetchOrder(ctx context.Context, symbol string, orderId string, opts ...option.ArgsOption) (*model.PerpOrder, error)

	// ========== 合约特有功能 ==========

	// SetLeverage 设置杠杆
	SetLeverage(ctx context.Context, symbol string, leverage int) error

	// SetMarginMode 设置保证金模式（isolated/cross）
	SetMarginMode(ctx context.Context, symbol string, mode string) error
}
