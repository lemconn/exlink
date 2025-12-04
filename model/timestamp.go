package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Timestamp time.Time

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	// 1) 去掉空格 & 引号
	s := strings.Trim(string(b), `" \t\n\r`)
	if s == "" || s == "null" {
		return nil
	}

	// 2) 解析为 int64（兼容字符串和数字）
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp %q: %w", s, err)
	}

	// 3) 根据长度判断单位
	var tm time.Time
	switch len(s) {
	case 10: // seconds
		tm = time.Unix(ts, 0)
	case 13: // milliseconds
		tm = time.UnixMilli(ts)
	case 16: // microseconds
		tm = time.UnixMicro(ts)
	case 19: // nanoseconds
		tm = time.Unix(0, ts)
	default:
		return fmt.Errorf("unsupported timestamp length: %d (%s)", len(s), s)
	}

	*t = Timestamp(tm)
	return nil
}

func (t Timestamp) Time() time.Time {
	return time.Time(t)
}
