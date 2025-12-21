package binance

import (
	"github.com/lemconn/exlink/common"
)

// Signer Binance 签名工具
type Signer struct {
	secretKey string
}

// NewSigner 创建签名工具
func NewSigner(secretKey string) *Signer {
	return &Signer{
		secretKey: secretKey,
	}
}

// Sign 对查询字符串进行签名
func (s *Signer) Sign(queryString string) string {
	return common.SignHMAC256(queryString, s.secretKey)
}

// BuildQueryString 构建查询字符串（用于签名）
func BuildQueryString(params map[string]interface{}) string {
	return common.BuildQueryString(params)
}
