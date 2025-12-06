package okx

import (
	"encoding/json"

	"github.com/lemconn/exlink/common"
)

// Signer OKX 签名工具
type Signer struct {
	secretKey  string
	passphrase string
}

// NewSigner 创建签名工具
func NewSigner(secretKey, passphrase string) *Signer {
	return &Signer{
		secretKey:  secretKey,
		passphrase: passphrase,
	}
}

// SignRequest 对请求进行签名（OKX API）
// method: GET, POST, DELETE
// path: API 路径
// timestamp: ISO8601 格式时间戳
// body: 请求体（POST 时使用）
// params: 查询参数
func (s *Signer) SignRequest(method, path, timestamp, body string, params map[string]interface{}) string {
	// 构建查询字符串
	queryString := ""
	if len(params) > 0 {
		queryString = common.BuildQueryString(params)
	}

	// OKX 签名格式: timestamp + method + path + (queryString or body)
	var message string
	if method == "GET" || method == "DELETE" {
		if queryString != "" {
			message = timestamp + method + path + "?" + queryString
		} else {
			message = timestamp + method + path
		}
	} else {
		message = timestamp + method + path + body
	}

	// Base64 编码的 HMAC256 签名
	return common.SignHMAC256Base64(message, s.secretKey)
}

// Sign 对消息进行签名（Base64编码，兼容旧方法）
func (s *Signer) Sign(message string) string {
	return common.SignHMAC256Base64(message, s.secretKey)
}

// BuildQueryString 构建查询字符串（用于签名）
func BuildQueryString(params map[string]interface{}) string {
	return common.BuildQueryString(params)
}

// BuildRequestBody 构建请求体（JSON 字符串）
func BuildRequestBody(body map[string]interface{}) (string, error) {
	if body == nil {
		return "", nil
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}
