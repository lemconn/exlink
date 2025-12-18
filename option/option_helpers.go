package option

import "time"

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
