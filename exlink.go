package exlink

import (
	"fmt"
	"sync"

	"github.com/lemconn/exlink/binance"
	"github.com/lemconn/exlink/bybit"
	"github.com/lemconn/exlink/exchange"
	"github.com/lemconn/exlink/gate"
	"github.com/lemconn/exlink/okx"
	"github.com/lemconn/exlink/option"
)

// 交易所名称常量
const (
	ExchangeBinance = "binance" // Binance 交易所
	ExchangeBybit   = "bybit"   // Bybit 交易所
	ExchangeOKX     = "okx"     // OKX 交易所
	ExchangeGate    = "gate"    // Gate 交易所
)

// 注意：ExchangeOptions 和 Option 相关定义已迁移到 option/init.go
// 注意：ExchangeArgsOptions 和 ArgsOption 相关定义在 option/call.go

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
func NewExchange(name string, opts ...option.Option) (exchange.Exchange, error) {
	// 初始化默认选项
	options := &option.ExchangeOptions{
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
