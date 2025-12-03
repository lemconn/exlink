package okx

import (
	"github.com/lemconn/exlink/common"
)

const (
	okxName       = "okx"
	okxBaseURL    = "https://www.okx.com"
	okxSandboxURL = "https://www.okx.com" // OKX使用同一个域名，通过header区分
)

// Client OKX 客户端
type Client struct {
	// HTTPClient HTTP 客户端
	HTTPClient *common.HTTPClient

	// APIKey API 密钥
	APIKey string

	// SecretKey 密钥
	SecretKey string

	// Passphrase 密码短语
	Passphrase string

	// Sandbox 是否为模拟盘
	Sandbox bool

	// ProxyURL 代理地址
	ProxyURL string

	// Debug 是否启用调试模式
	Debug bool
}

// NewClient 创建 OKX 客户端
func NewClient(apiKey, secretKey, passphrase string, options map[string]interface{}) (*Client, error) {
	baseURL := okxBaseURL
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

	client := &Client{
		HTTPClient: common.NewHTTPClient(baseURL),
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
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

	return client, nil
}

