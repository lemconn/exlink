package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SignHMAC256 HMAC-SHA256签名（hex编码）
func SignHMAC256(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignHMAC256Base64 HMAC-SHA256签名（base64编码，用于OKX）
func SignHMAC256Base64(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// BuildQueryString 构建查询字符串
func BuildQueryString(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}

	// 排序键
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	var parts []string
	for _, k := range keys {
		v := params[k]
		var value string
		switch val := v.(type) {
		case string:
			value = val
		case int:
			value = strconv.Itoa(val)
		case int64:
			value = strconv.FormatInt(val, 10)
		case float64:
			value = strconv.FormatFloat(val, 'f', -1, 64)
		case bool:
			value = strconv.FormatBool(val)
		default:
			value = fmt.Sprintf("%v", val)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(value)))
	}
	return strings.Join(parts, "&")
}

// GetTimestamp 获取时间戳（毫秒）
func GetTimestamp() int64 {
	return time.Now().UnixMilli()
}

// GetTimestampSeconds 获取时间戳（秒）
func GetTimestampSeconds() int64 {
	return time.Now().Unix()
}

// GetISO8601Timestamp 获取ISO8601格式的时间戳（用于OKX）
func GetISO8601Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
