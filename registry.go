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
)

// ExchangeOptions 交易所配置选项
type ExchangeOptions struct {
	APIKey       string
	SecretKey    string
	Passphrase   string
	Sandbox      bool
	Proxy        string
	BaseURL      string
	FetchMarkets []string
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
func WithFetchMarkets(markets []string) Option {
	return func(opts *ExchangeOptions) {
		opts.FetchMarkets = markets
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
	Register("binance", binance.NewBinance)
	Register("bybit", bybit.NewBybit)
	Register("okx", okx.NewOKX)
	Register("gate", gate.NewGate)
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
		optionsMap["fetchMarkets"] = options.FetchMarkets
	}
	if options.Passphrase != "" {
		optionsMap["passphrase"] = options.Passphrase
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
