package bybit

import (
	"github.com/lemconn/exlink/common"
)

const (
	bybitName       = "bybit"
	bybitBaseURL    = "https://api.bybit.com"
	bybitSandboxURL = "https://api-demo.bybit.com"
)

// Client Bybit 客户端
type Client struct {
	// HTTPClient HTTP 客户端（Bybit 使用统一的 API）
	HTTPClient *common.HTTPClient

	// APIKey API 密钥
	APIKey string

	// SecretKey 密钥
	SecretKey string

	// Sandbox 是否为模拟盘
	Sandbox bool

	// ProxyURL 代理地址
	ProxyURL string

	// Debug 是否启用调试模式
	Debug bool
}

// NewClient 创建 Bybit 客户端
func NewClient(apiKey, secretKey string, options map[string]interface{}) (*Client, error) {
	baseURL := bybitBaseURL
	sandbox := false
	proxyURL := ""
	debug := false

	if v, ok := options["baseURL"].(string); ok {
		baseURL = v
	}
	if v, ok := options["sandbox"].(bool); ok {
		sandbox = v
	}
	if v, ok := options["proxy"].(string); ok {
		proxyURL = v
	}
	if v, ok := options["debug"].(bool); ok {
		debug = v
	}

	if sandbox {
		baseURL = bybitSandboxURL
	}

	client := &Client{
		HTTPClient: common.NewHTTPClient(baseURL),
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Sandbox:    sandbox,
		ProxyURL:   proxyURL,
		Debug:      debug,
	}

	// 设置代理
	if proxyURL != "" {
		if err := client.HTTPClient.SetProxy(proxyURL); err != nil {
			return nil, err
		}
	}

	// 设置调试模式
	if debug {
		client.HTTPClient.SetDebug(true)
	}

	// 设置请求头
	if apiKey != "" {
		client.HTTPClient.SetHeader("X-BAPI-API-KEY", apiKey)
	}

	return client, nil
}

