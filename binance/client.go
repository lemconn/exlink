package binance

import (
	"github.com/lemconn/exlink/common"
)

const (
	binanceName           = "binance"
	binanceBaseURL        = "https://api.binance.com"
	binanceSandboxURL     = "https://demo-api.binance.com"
	binanceFapiBaseURL    = "https://fapi.binance.com"
	binanceFapiSandboxURL = "https://demo-fapi.binance.com"
)

// Client Binance 客户端，包含现货和合约的 HTTP 客户端
type Client struct {
	// SpotClient 现货 API 客户端
	SpotClient *common.HTTPClient

	// PerpClient 永续合约 API 客户端
	PerpClient *common.HTTPClient

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

// NewClient 创建 Binance 客户端
func NewClient(apiKey, secretKey string, options map[string]interface{}) (*Client, error) {
	baseURL := binanceBaseURL
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
		baseURL = binanceSandboxURL
	}

	fapiBaseURL := binanceFapiBaseURL
	if sandbox {
		fapiBaseURL = binanceFapiSandboxURL
	}

	client := &Client{
		SpotClient: common.NewHTTPClient(baseURL),
		PerpClient: common.NewHTTPClient(fapiBaseURL),
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Sandbox:    sandbox,
		ProxyURL:   proxyURL,
		Debug:      debug,
	}

	// 设置代理
	if proxyURL != "" {
		if err := client.SpotClient.SetProxy(proxyURL); err != nil {
			return nil, err
		}
		if err := client.PerpClient.SetProxy(proxyURL); err != nil {
			return nil, err
		}
	}

	// 设置调试模式
	if debug {
		client.SpotClient.SetDebug(true)
		client.PerpClient.SetDebug(true)
	}

	// 设置请求头
	if apiKey != "" {
		client.SpotClient.SetHeader("X-MBX-APIKEY", apiKey)
		client.PerpClient.SetHeader("X-MBX-APIKEY", apiKey)
	}

	return client, nil
}
