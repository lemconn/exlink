package gate

import (
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

// Sign 对消息进行签名（SHA512）
func (s *Signer) Sign(message string) string {
	return common.SignHMAC512(message, s.secretKey)
}

// BuildQueryString 构建查询字符串（用于签名）
func BuildQueryString(params map[string]interface{}) string {
	return common.BuildQueryString(params)
}

