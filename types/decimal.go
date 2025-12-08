package types

import (
	"strings"

	"github.com/shopspring/decimal"
)

// ExDecimal 支持空字符串的 decimal.Decimal 类型
// 用于 JSON 反序列化时处理空字符串或 null 值
type ExDecimal struct {
	decimal.Decimal
}

// UnmarshalJSON 自定义 JSON 反序列化，支持空字符串
func (d *ExDecimal) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(strings.Trim(string(data), `"`))
	if s == "" || s == "null" {
		d.Decimal = decimal.Zero
		return nil
	}
	return d.Decimal.UnmarshalJSON(data)
}
