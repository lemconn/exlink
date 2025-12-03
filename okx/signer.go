package okx

import (
	"github.com/lemconn/exlink/common"
)

// Signer OKX 签名工具
type Signer struct {
	secretKey string
	passphrase string
}

// NewSigner 创建签名工具
func NewSigner(secretKey, passphrase string) *Signer {
	return &Signer{
		secretKey:  secretKey,
		passphrase: passphrase,
	}
}

// Sign 对消息进行签名（Base64编码）
func (s *Signer) Sign(message string) string {
	return common.SignHMAC256Base64(message, s.secretKey)
}

// BuildQueryString 构建查询字符串（用于签名）
func BuildQueryString(params map[string]interface{}) string {
	return common.BuildQueryString(params)
}

