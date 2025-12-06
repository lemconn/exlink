package gate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lemconn/exlink/common"
)

// Signer Gate 签名工具
type Signer struct {
	secretKey string
}

// NewSigner 创建签名工具
func NewSigner(secretKey string) *Signer {
	return &Signer{
		secretKey: secretKey,
	}
}

// SignRequest Gate 签名方法
// method: GET, POST, DELETE
// path: API 路径
// queryString: 查询字符串
// body: 请求体（POST 时使用）
// timestamp: Unix 时间戳（秒）
func (s *Signer) SignRequest(method, path, queryString, body string, timestamp int64) string {
	bodyHash := common.HashSHA512(body)

	// 去掉路径中的 /api/v4 前缀（如果存在），因为签名格式中已经包含了
	signPath := path
	if strings.HasPrefix(path, "/api/v4") {
		signPath = strings.TrimPrefix(path, "/api/v4")
	}

	// Gate 签名格式: method\n/api/v4/path\nqueryString\nbodyHash\ntimestamp
	payload := fmt.Sprintf("%s\n/api/v4%s\n%s\n%s\n%d",
		strings.ToUpper(method), signPath, queryString, bodyHash, timestamp)

	return common.SignHMAC512(payload, s.secretKey)
}

// Sign 对消息进行签名（SHA512，兼容旧方法）
func (s *Signer) Sign(message string) string {
	return common.SignHMAC512(message, s.secretKey)
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
