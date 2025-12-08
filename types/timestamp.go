package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExTimestamp 支持多种格式的时间戳类型
// 用于 JSON 反序列化时处理不同格式的时间戳（秒、毫秒、微秒、纳秒、RFC3339）
type ExTimestamp struct {
	time.Time
	// sourceFormat 记录输入格式，用于序列化时保持原始格式
	// 可能的值: "s"(秒), "ms"(毫秒), "us"(微秒), "ns"(纳秒), "rfc3339"(RFC3339字符串)
	sourceFormat string
}

// UnmarshalJSON 自定义 JSON 反序列化，支持多种时间戳格式
func (t *ExTimestamp) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(strings.Trim(string(b), `"`))
	if s == "" || s == "null" {
		// 明确设置为零值
		t.Time = time.Time{}
		t.sourceFormat = ""
		return nil
	}

	// 尝试 int64（各种 timestamp）
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		switch len(s) {
		case 10:
			t.Time = time.Unix(ts, 0)
			t.sourceFormat = "s"
		case 13:
			t.Time = time.UnixMilli(ts)
			t.sourceFormat = "ms"
		case 16:
			t.Time = time.UnixMicro(ts)
			t.sourceFormat = "us"
		case 19:
			t.Time = time.Unix(0, ts)
			t.sourceFormat = "ns"
		default:
			return fmt.Errorf("unsupported timestamp length: %d (%s)", len(s), s)
		}
		return nil
	}

	// fallback: RFC3339 string
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("invalid timestamp %s: %w", s, err)
	}
	t.Time = tt
	t.sourceFormat = "rfc3339"
	return nil
}

// MarshalJSON 自定义 JSON 序列化，保持原始格式
func (t ExTimestamp) MarshalJSON() ([]byte, error) {
	switch t.sourceFormat {
	case "s":
		return []byte(strconv.FormatInt(t.Unix(), 10)), nil
	case "ms":
		return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
	case "us":
		return []byte(strconv.FormatInt(t.UnixMicro(), 10)), nil
	case "ns":
		return []byte(strconv.FormatInt(t.UnixNano(), 10)), nil
	case "rfc3339":
		return json.Marshal(t.Format(time.RFC3339))
	default:
		// 默认使用毫秒时间戳（大多数交易所API使用毫秒时间戳）
		return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
	}
}
