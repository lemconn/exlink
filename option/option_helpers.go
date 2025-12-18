package option

import "time"

//nolint:unused
func StringPresent(s *string) bool {
	return s != nil && *s != ""
}

//nolint:unused
func IntPresent(i *int) bool {
	return i != nil
}

//nolint:unused
func BoolPresent(b *bool) bool {
	return b != nil
}

//nolint:unused
func TimePresent(t *time.Time) bool {
	return t != nil && !t.IsZero()
}

//nolint:unused // GetString 返回字符串值及是否存在（非 nil 且非空）
func GetString(s *string) (string, bool) {
	if s == nil || *s == "" {
		return "", false
	}
	return *s, true
}

//nolint:unused // GetInt 返回 int 值及是否存在（非 nil）
func GetInt(i *int) (int, bool) {
	if i == nil {
		return 0, false
	}
	return *i, true
}

//nolint:unused // GetBool 返回 bool 值及是否存在（非 nil）
func GetBool(b *bool) (bool, bool) {
	if b == nil {
		return false, false
	}
	return *b, true
}

//nolint:unused // GetTime 返回 time.Time 值及是否存在（非 nil 且非零值）
func GetTime(t *time.Time) (time.Time, bool) {
	if t == nil || t.IsZero() {
		return time.Time{}, false
	}
	return *t, true
}
