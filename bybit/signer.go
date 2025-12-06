package bybit

import (
	"encoding/json"
	"strconv"

	"github.com/lemconn/exlink/common"
)

// Signer Bybit 签名工具
type Signer struct {
	secretKey string
	apiKey    string
}

// NewSigner 创建签名工具
func NewSigner(secretKey string) *Signer {
	return &Signer{
		secretKey: secretKey,
	}
}

// SetAPIKey 设置 API Key（Bybit v5 签名需要）
func (s *Signer) SetAPIKey(apiKey string) {
	s.apiKey = apiKey
}

// Sign 对查询字符串进行签名（Bybit v5 API）
// method: GET, POST, DELETE
// params: 查询参数
// body: 请求体（POST 时使用）
func (s *Signer) SignRequest(method string, params map[string]interface{}, body map[string]interface{}) (signature, timestamp string) {
	timestamp = strconv.FormatInt(common.GetTimestamp(), 10)
	recvWindow := "5000" // 默认接收窗口

	// 构建查询字符串
	queryString := ""
	if len(params) > 0 {
		queryString = common.BuildQueryString(params)
	}

	// 构建请求体
	bodyStr := ""
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyStr = string(bodyBytes)
	}

	// Bybit v5 签名格式: timestamp + apiKey + recvWindow + (body or queryString)
	authBase := timestamp + s.apiKey + recvWindow
	var authFull string
	if method == "POST" || method == "PUT" {
		authFull = authBase + bodyStr
	} else {
		authFull = authBase + queryString
	}

	signature = common.SignHMAC256(authFull, s.secretKey)
	return signature, timestamp
}

// Sign 对查询字符串进行签名（兼容旧方法）
func (s *Signer) Sign(queryString string) string {
	return common.SignHMAC256(queryString, s.secretKey)
}

// BuildQueryString 构建查询字符串（用于签名）
func BuildQueryString(params map[string]interface{}) string {
	return common.BuildQueryString(params)
}
