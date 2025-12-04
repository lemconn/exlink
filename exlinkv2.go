package exlink

import (
	"fmt"
	"sync"

	"github.com/lemconn/exlink/binance"
	"github.com/lemconn/exlink/bybit"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/gate"
	"github.com/lemconn/exlink/model"
	"github.com/lemconn/exlink/okx"
)

// 交易所名称常量（V2版本）
const (
	ExchangeV2Binance = "binance" // Binance 交易所
	ExchangeV2Bybit   = "bybit"   // Bybit 交易所
	ExchangeV2OKX     = "okx"     // OKX 交易所
	ExchangeV2Gate    = "gate"    // Gate 交易所
)

// ExchangeOptionsV2 交易所配置选项（V2版本）
type ExchangeOptionsV2 struct {
	APIKey       string
	SecretKey    string
	Password     string // 密码（用于 OKX 等需要 password 的交易所）
	HedgeMode    bool   // 是否为双向持仓模式（用于合约下单时控制参数）
	Sandbox      bool
	Proxy        string
	BaseURL      string
	FetchMarkets []model.MarketType
	Debug        bool
	Options      map[string]interface{} // 其他自定义选项
}

// OptionV2 配置选项函数类型（V2版本）
type OptionV2 func(*ExchangeOptionsV2)

// WithAPIKeyV2 设置 API Key
func WithAPIKeyV2(apiKey string) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.APIKey = apiKey
	}
}

// WithSecretKeyV2 设置 Secret Key
func WithSecretKeyV2(secretKey string) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.SecretKey = secretKey
	}
}

// WithPasswordV2 设置 Password（用于 OKX 等需要 password 的交易所）
func WithPasswordV2(password string) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.Password = password
	}
}

// WithSandboxV2 设置是否使用模拟盘
func WithSandboxV2(sandbox bool) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.Sandbox = sandbox
	}
}

// WithProxyV2 设置代理
func WithProxyV2(proxy string) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.Proxy = proxy
	}
}

// WithBaseURLV2 设置基础 URL
func WithBaseURLV2(baseURL string) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.BaseURL = baseURL
	}
}

// WithFetchMarketsV2 设置要加载的市场类型
func WithFetchMarketsV2(markets ...model.MarketType) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.FetchMarkets = markets
	}
}

// WithDebugV2 设置是否启用调试模式
func WithDebugV2(debug bool) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.Debug = debug
	}
}

// WithHedgeModeV2 设置是否为双向持仓模式（用于合约下单时控制参数）
func WithHedgeModeV2(hedgeMode bool) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		opts.HedgeMode = hedgeMode
	}
}

// WithOptionV2 设置自定义选项
func WithOptionV2(key string, value interface{}) OptionV2 {
	return func(opts *ExchangeOptionsV2) {
		if opts.Options == nil {
			opts.Options = make(map[string]interface{})
		}
		opts.Options[key] = value
	}
}

// ExchangeFactoryV2 交易所工厂函数（V2版本）
type ExchangeFactoryV2 func(apiKey, secretKey string, options map[string]interface{}) (exchange.Exchange, error)

// RegistryV2 交易所注册表（V2版本）
type RegistryV2 struct {
	mu        sync.RWMutex
	factories map[string]ExchangeFactoryV2
}

var globalRegistryV2 = &RegistryV2{
	factories: make(map[string]ExchangeFactoryV2),
}

// initV2 初始化函数，注册所有支持的交易所（V2版本）
func init() {
	RegisterV2(ExchangeV2Binance, binance.NewBinance)
	RegisterV2(ExchangeV2Bybit, bybit.NewBybit)
	RegisterV2(ExchangeV2OKX, okx.NewOKX)
	RegisterV2(ExchangeV2Gate, gate.NewGate)
}

// RegisterV2 注册交易所（V2版本）
func RegisterV2(name string, factory ExchangeFactoryV2) {
	globalRegistryV2.mu.Lock()
	defer globalRegistryV2.mu.Unlock()
	globalRegistryV2.factories[name] = factory
}

// NewExchangeV2 创建交易所实例（V2版本，使用 Functional Options Pattern）
// 返回 exchange.Exchange 接口，提供 Spot() 和 Perp() 方法
func NewExchangeV2(name string, opts ...OptionV2) (exchange.Exchange, error) {
	// 初始化默认选项
	options := &ExchangeOptionsV2{
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
		// 转换为字符串数组
		marketStrings := make([]string, len(options.FetchMarkets))
		for i, mt := range options.FetchMarkets {
			marketStrings[i] = string(mt)
		}
		optionsMap["fetchMarkets"] = marketStrings
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

	globalRegistryV2.mu.RLock()
	factory, ok := globalRegistryV2.factories[name]
	globalRegistryV2.mu.RUnlock()

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

// GetSupportedExchangesV2 获取支持的交易所列表（V2版本）
func GetSupportedExchangesV2() []string {
	globalRegistryV2.mu.RLock()
	defer globalRegistryV2.mu.RUnlock()

	exchanges := make([]string, 0, len(globalRegistryV2.factories))
	for name := range globalRegistryV2.factories {
		exchanges = append(exchanges, name)
	}
	return exchanges
}

// IsExchangeSupportedV2 检查交易所是否支持（V2版本）
func IsExchangeSupportedV2(name string) bool {
	globalRegistryV2.mu.RLock()
	defer globalRegistryV2.mu.RUnlock()
	_, ok := globalRegistryV2.factories[name]
	return ok
}
