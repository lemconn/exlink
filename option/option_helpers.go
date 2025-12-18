package option

import (
	"time"

	"github.com/shopspring/decimal"
)

func StringPresent(s *string) bool {
	return s != nil && *s != ""
}

func IntPresent(i *int) bool {
	return i != nil
}

func BoolPresent(b *bool) bool {
	return b != nil
}

func TimePresent(t *time.Time) bool {
	return t != nil && !t.IsZero()
}

// GetString 返回字符串值及是否存在（非 nil 且非空）
func GetString(s *string) (string, bool) {
	if s == nil || *s == "" {
		return "", false
	}
	return *s, true
}

func GetDecimalFromString(s *string) (decimal.Decimal, bool) {
	if s == nil || *s == "" {
		return decimal.Zero, false
	}

	d, err := decimal.NewFromString(*s)
	if err != nil {
		return decimal.Zero, false
	}

	return d, true
}

func GetDecimalFromInt64(i *int64) (decimal.Decimal, bool) {
	if i == nil {
		return decimal.Zero, false
	}

	return decimal.NewFromInt(*i), true
}

// GetInt 返回 int 值及是否存在（非 nil）
func GetInt(i *int) (int, bool) {
	if i == nil {
		return 0, false
	}
	return *i, true
}

// GetBool 返回 bool 值及是否存在（非 nil）
func GetBool(b *bool) (bool, bool) {
	if b == nil {
		return false, false
	}
	return *b, true
}

// GetTime 返回 time.Time 值及是否存在（非 nil 且非零值）
func GetTime(t *time.Time) (time.Time, bool) {
	if t == nil || t.IsZero() {
		return time.Time{}, false
	}
	return *t, true
}
