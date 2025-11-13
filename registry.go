package exlink

import (
	"context"
	"fmt"
	"sync"

	"github.com/lemconn/exlink/base"
	"github.com/lemconn/exlink/exchanges/binance"
	"github.com/lemconn/exlink/exchanges/bybit"
	"github.com/lemconn/exlink/exchanges/gate"
	"github.com/lemconn/exlink/exchanges/okx"
	"github.com/lemconn/exlink/types"
)

// 市场类型常量别名（方便使用）
const (
	MarketSpot   = types.MarketTypeSpot   // 现货市场
	MarketSwap   = types.MarketTypeSwap   // 永续合约市场
	MarketFuture = types.MarketTypeFuture // 永续合约市场（同义于 MarketSwap）
)

// 交易所名称常量
const (
	ExchangeBinance = "binance" // Binance 交易所
	ExchangeBybit   = "bybit"   // Bybit 交易所
	ExchangeOKX     = "okx"     // OKX 交易所
	ExchangeGate    = "gate"    // Gate.io 交易所
)

// ExchangeOptions 交易所配置选项
type ExchangeOptions struct {
	APIKey       string
	SecretKey    string
	Passphrase   string
	Sandbox      bool
	Proxy        string
	BaseURL      string
	FetchMarkets []types.MarketType
	Debug        bool
	Options      map[string]interface{} // 其他自定义选项
}

// Option 配置选项函数类型
type Option func(*ExchangeOptions)

// WithAPIKey 设置 API Key
func WithAPIKey(apiKey string) Option {
	return func(opts *ExchangeOptions) {
		opts.APIKey = apiKey
	}
}

// WithSecretKey 设置 Secret Key
func WithSecretKey(secretKey string) Option {
	return func(opts *ExchangeOptions) {
		opts.SecretKey = secretKey
	}
}

// WithPassphrase 设置 Passphrase（用于 OKX 等需要 passphrase 的交易所）
func WithPassphrase(passphrase string) Option {
	return func(opts *ExchangeOptions) {
		opts.Passphrase = passphrase
	}
}

// WithSandbox 设置是否使用模拟盘
func WithSandbox(sandbox bool) Option {
	return func(opts *ExchangeOptions) {
		opts.Sandbox = sandbox
	}
}

// WithProxy 设置代理
func WithProxy(proxy string) Option {
	return func(opts *ExchangeOptions) {
		opts.Proxy = proxy
	}
}

// WithBaseURL 设置基础 URL
func WithBaseURL(baseURL string) Option {
	return func(opts *ExchangeOptions) {
		opts.BaseURL = baseURL
	}
}

// WithFetchMarkets 设置要加载的市场类型
func WithFetchMarkets(markets ...types.MarketType) Option {
	return func(opts *ExchangeOptions) {
		opts.FetchMarkets = markets
	}
}

// WithDebug 设置是否启用调试模式
func WithDebug(debug bool) Option {
	return func(opts *ExchangeOptions) {
		opts.Debug = debug
	}
}

// WithOption 设置自定义选项
func WithOption(key string, value interface{}) Option {
	return func(opts *ExchangeOptions) {
		if opts.Options == nil {
			opts.Options = make(map[string]interface{})
		}
		opts.Options[key] = value
	}
}

// ExchangeFactory 交易所工厂函数
type ExchangeFactory func(apiKey, secretKey string, options map[string]interface{}) (base.Exchange, error)

// Registry 交易所注册表
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ExchangeFactory
	exchanges map[string]base.Exchange
}

var globalRegistry = &Registry{
	factories: make(map[string]ExchangeFactory),
	exchanges: make(map[string]base.Exchange),
}

// init 初始化函数，注册所有支持的交易所
func init() {
	Register(ExchangeBinance, binance.NewBinance)
	Register(ExchangeBybit, bybit.NewBybit)
	Register(ExchangeOKX, okx.NewOKX)
	Register(ExchangeGate, gate.NewGate)
}

// Register 注册交易所
func Register(name string, factory ExchangeFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.factories[name] = factory
}

// NewExchange 创建交易所实例（使用 Functional Options Pattern）
func NewExchange(name string, opts ...Option) (base.Exchange, error) {
	// 初始化默认选项
	options := &ExchangeOptions{
		Options: make(map[string]interface{}),
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(options)
	}

	// 将选项转换为 map[string]interface{} 以兼容现有的 ExchangeFactory
	optionsMap := make(map[string]interface{})
	if options.Sandbox {
		optionsMap["sandbox"] = options.Sandbox
	}
	if options.Proxy != "" {
		optionsMap["proxy"] = options.Proxy
	}
	if options.BaseURL != "" {
		optionsMap["baseURL"] = options.BaseURL
	}
	if len(options.FetchMarkets) > 0 {
		// 转换为字符串数组以兼容现有的 ExchangeFactory
		marketStrings := make([]string, len(options.FetchMarkets))
		for i, mt := range options.FetchMarkets {
			marketStrings[i] = string(mt)
		}
		optionsMap["fetchMarkets"] = marketStrings
	}
	if options.Passphrase != "" {
		optionsMap["passphrase"] = options.Passphrase
	}
	if options.Debug {
		optionsMap["debug"] = options.Debug
	}
	// 合并自定义选项
	for k, v := range options.Options {
		optionsMap[k] = v
	}

	globalRegistry.mu.RLock()
	factory, ok := globalRegistry.factories[name]
	globalRegistry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", base.ErrExchangeNotSupported, name)
	}

	exchange, err := factory(options.APIKey, options.SecretKey, optionsMap)
	if err != nil {
		return nil, err
	}

	// 加载市场信息
	ctx := context.Background()
	if err := exchange.LoadMarkets(ctx, false); err != nil {
		return nil, fmt.Errorf("failed to load markets: %w", err)
	}

	return exchange, nil
}

// GetSupportedExchanges 获取支持的交易所列表
func GetSupportedExchanges() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	exchanges := make([]string, 0, len(globalRegistry.factories))
	for name := range globalRegistry.factories {
		exchanges = append(exchanges, name)
	}
	return exchanges
}

// IsExchangeSupported 检查交易所是否支持
func IsExchangeSupported(name string) bool {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	_, ok := globalRegistry.factories[name]
	return ok
}
