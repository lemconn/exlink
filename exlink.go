package exlink

import (
	"fmt"
	"sync"

	"github.com/lemconn/exlink/binance"
	"github.com/lemconn/exlink/bybit"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/gate"
	"github.com/lemconn/exlink/okx"
)

// 交易所名称常量
const (
	ExchangeBinance = "binance" // Binance 交易所
	ExchangeBybit   = "bybit"   // Bybit 交易所
	ExchangeOKX     = "okx"     // OKX 交易所
	ExchangeGate    = "gate"    // Gate 交易所
)

// ExchangeOptions 交易所配置选项
type ExchangeOptions struct {
	APIKey    string
	SecretKey string
	Password  string // 密码（用于 OKX 等需要 password 的交易所）
	HedgeMode bool   // 是否为双向持仓模式（用于合约下单时控制参数）
	Sandbox   bool
	Proxy     string
	BaseURL   string
	Debug     bool
	Options   map[string]interface{} // 其他自定义选项
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

// WithPassword 设置 Password（用于 OKX 等需要 password 的交易所）
func WithPassword(password string) Option {
	return func(opts *ExchangeOptions) {
		opts.Password = password
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

// WithDebug 设置是否启用调试模式
func WithDebug(debug bool) Option {
	return func(opts *ExchangeOptions) {
		opts.Debug = debug
	}
}

// WithHedgeMode 设置是否为双向持仓模式（用于合约下单时控制参数）
func WithHedgeMode(hedgeMode bool) Option {
	return func(opts *ExchangeOptions) {
		opts.HedgeMode = hedgeMode
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
type ExchangeFactory func(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error)

// Registry 交易所注册表
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ExchangeFactory
}

var globalRegistry = &Registry{
	factories: make(map[string]ExchangeFactory),
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
// 返回 exchange.Exchange 接口，提供 Spot() 和 Perp() 方法
func NewExchange(name string, opts ...Option) (exchange.Exchange, error) {
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
	if options.Password != "" {
		optionsMap["password"] = options.Password
	}
	if options.HedgeMode {
		optionsMap["hedgeMode"] = options.HedgeMode
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
		return nil, fmt.Errorf("exchange not supported: %s", name)
	}

	// 直接调用工厂函数创建 exchange.Exchange 实例
	ex, err := factory(options.APIKey, options.SecretKey, optionsMap)
	if err != nil {
		return nil, err
	}

	return ex, nil
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
