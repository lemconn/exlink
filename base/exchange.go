package base

import (
	"context"
	"time"

	"github.com/lemconn/exlink/types"
)

// Exchange 交易所接口
type Exchange interface {
	// 基本信息
	Name() string                                                                         // 交易所名称
	GetMarkets(ctx context.Context, marketType types.MarketType) ([]*types.Market, error) // 获取市场列表

	// 行情数据
	FetchTicker(ctx context.Context, symbol string) (*types.Ticker, error)                                             // 获取行情
	FetchTickers(ctx context.Context, symbols ...string) (map[string]*types.Ticker, error)                             // 批量获取行情
	FetchOHLCV(ctx context.Context, symbol string, timeframe string, since time.Time, limit int) (types.OHLCVs, error) // 获取K线数据

	// 账户信息
	FetchBalance(ctx context.Context) (types.Balances, error) // 获取余额

	// 订单操作
	CreateOrder(ctx context.Context, symbol string, side types.OrderSide, orderType types.OrderType, amount, price string, params map[string]interface{}) (*types.Order, error) // 创建订单
	CancelOrder(ctx context.Context, orderID, symbol string) error                                                                                                               // 取消订单
	FetchOrder(ctx context.Context, orderID, symbol string) (*types.Order, error)                                                                                                // 查询订单
	FetchOrders(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Order, error)                                                                          // 查询订单列表
	FetchOpenOrders(ctx context.Context, symbol string) ([]*types.Order, error)                                                                                                  // 查询未成交订单

	// 交易记录
	FetchTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error)   // 获取交易记录
	FetchMyTrades(ctx context.Context, symbol string, since time.Time, limit int) ([]*types.Trade, error) // 获取我的交易记录

	// 合约相关（永续合约）
	FetchPositions(ctx context.Context, symbols ...string) ([]*types.Position, error) // 获取持仓
	SetLeverage(ctx context.Context, symbol string, leverage int) error               // 设置杠杆
	SetMarginMode(ctx context.Context, symbol string, mode string) error              // 设置保证金模式（isolated/cross）

	// 工具方法
	LoadMarkets(ctx context.Context, reload bool) error // 加载市场信息
	GetMarket(symbol string) (*types.Market, error)     // 获取市场信息
}

// BaseExchange 交易所基础实现
type BaseExchange struct {
	name     string
	markets  map[string]*types.Market
	options  map[string]interface{}
	sandbox  bool
	proxyURL string
}

// NewBaseExchange 创建基础交易所
func NewBaseExchange(name string) *BaseExchange {
	return &BaseExchange{
		name:    name,
		markets: make(map[string]*types.Market),
		options: make(map[string]interface{}),
	}
}

// Name 返回交易所名称
func (e *BaseExchange) Name() string {
	return e.name
}

// SetOption 设置选项
func (e *BaseExchange) SetOption(key string, value interface{}) {
	e.options[key] = value
}

// GetOption 获取选项
func (e *BaseExchange) GetOption(key string) interface{} {
	return e.options[key]
}

// SetSandbox 设置模拟盘模式
func (e *BaseExchange) SetSandbox(sandbox bool) {
	e.sandbox = sandbox
}

// IsSandbox 是否模拟盘模式
func (e *BaseExchange) IsSandbox() bool {
	return e.sandbox
}

// SetProxy 设置代理
func (e *BaseExchange) SetProxy(proxyURL string) {
	e.proxyURL = proxyURL
}

// GetProxy 获取代理URL
func (e *BaseExchange) GetProxy() string {
	return e.proxyURL
}

// SetMarkets 设置市场信息
func (e *BaseExchange) SetMarkets(markets []*types.Market) {
	e.markets = make(map[string]*types.Market)
	for _, market := range markets {
		e.markets[market.Symbol] = market
	}
}

// GetMarket 获取市场信息
func (e *BaseExchange) GetMarket(symbol string) (*types.Market, error) {
	market, ok := e.markets[symbol]
	if !ok {
		return nil, ErrMarketNotFound
	}
	return market, nil
}

// GetMarketsMap 获取所有市场（内部方法）
func (e *BaseExchange) GetMarketsMap() map[string]*types.Market {
	return e.markets
}

// GetMarketID 获取交易所格式的 symbol ID
// 优先从已加载的市场中查找，如果未找到则尝试使用后备转换函数
func (e *BaseExchange) GetMarketID(symbol string) (string, error) {
	// 优先从已加载的市场中查找
	market, ok := e.markets[symbol]
	if ok {
		return market.ID, nil
	}

	// 如果市场未加载，尝试使用后备转换函数
	// 这里需要根据交易所名称调用对应的转换函数
	// 由于 BaseExchange 不知道具体的转换函数，我们返回错误，让具体交易所实现
	return "", ErrMarketNotFound
}

