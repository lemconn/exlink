package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	proxy   string
	debug   bool
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

// SetProxy 设置代理
func (c *HTTPClient) SetProxy(proxyURL string) error {
	if proxyURL == "" {
		c.client.Transport = nil
		c.proxy = ""
		return nil
	}

	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}

	if c.client.Transport != nil {
		// 保留现有的Transport设置
		if existingTransport, ok := c.client.Transport.(*http.Transport); ok {
			transport.TLSClientConfig = existingTransport.TLSClientConfig
		}
	}

	c.client.Transport = transport
	c.proxy = proxyURL
	return nil
}

// GetProxy 获取当前代理设置
func (c *HTTPClient) GetProxy() string {
	return c.proxy
}

// SetHeader 设置请求头
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetTimeout 设置超时时间
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

// SetDebug 设置是否启用调试模式
func (c *HTTPClient) SetDebug(debug bool) {
	c.debug = debug
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, path string, params map[string]interface{}) ([]byte, error) {
	return c.Request(ctx, http.MethodGet, path, params, nil)
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.Request(ctx, http.MethodPost, path, nil, data)
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, path string, params map[string]interface{}, body interface{}) ([]byte, error) {
	return c.Request(ctx, http.MethodDelete, path, params, body)
}

// Request 发送HTTP请求
func (c *HTTPClient) Request(ctx context.Context, method, path string, params map[string]interface{}, body interface{}) ([]byte, error) {
	url := c.baseURL + path

	// 构建查询参数 - 使用 BuildQueryString 确保与签名时一致（排序和URL编码）
	if len(params) > 0 {
		query := BuildQueryString(params)
		if query != "" {
			url += "?" + query
		}
	}

	// 构建请求体
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 调试输出：请求信息
	if c.debug {
		fmt.Printf("[DEBUG] Request:\n")
		fmt.Printf("  Method: %s\n", method)
		fmt.Printf("  URL: %s\n", url)
		headersJSON, _ := json.Marshal(c.headers)
		fmt.Printf("  Headers: %s\n", string(headersJSON))
		if body != nil {
			bodyBytes, _ := json.Marshal(body)
			fmt.Printf("  Body: %s\n", string(bodyBytes))
		}
		fmt.Println()
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the request
			if c.debug {
				fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
			}
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 调试输出：响应信息
	if c.debug {
		fmt.Printf("[DEBUG] Response:\n")
		fmt.Printf("  Status: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("  Body: %s\n", string(respBody))
		fmt.Println()
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
