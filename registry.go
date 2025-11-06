package exlink

import (
	"context"
	"fmt"
	"sync"
)

// ExchangeFactory 交易所工厂函数
type ExchangeFactory func(apiKey, secretKey string, options map[string]interface{}) (Exchange, error)

// Registry 交易所注册表
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ExchangeFactory
	exchanges map[string]Exchange
}

var globalRegistry = &Registry{
	factories: make(map[string]ExchangeFactory),
	exchanges: make(map[string]Exchange),
}

// Register 注册交易所
func Register(name string, factory ExchangeFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.factories[name] = factory
}

// NewExchange 创建交易所实例
func NewExchange(name string, apiKey, secretKey string, options map[string]interface{}) (Exchange, error) {
	globalRegistry.mu.RLock()
	factory, ok := globalRegistry.factories[name]
	globalRegistry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrExchangeNotSupported, name)
	}

	exchange, err := factory(apiKey, secretKey, options)
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
