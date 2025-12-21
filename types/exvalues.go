package types

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ExValues is a generic container for HTTP request parameters.
//
// Design notes:
//
//   - queryOrder keeps the first-seen order of keys for query parameters.
//   - queryValues stores one or more values per key for query parameters.
//   - bodyOrder keeps the first-seen order of keys for body parameters.
//   - bodyValues stores one or more values per key for body parameters.
//   - headerOrder keeps the first-seen order of keys for headers.
//   - headerValues stores one or more values per key for headers.
//   - EncodeQuery preserves key order and value order.
//   - EncodeBody returns body as map[string]any.
//   - EncodeHeader returns headers as map[string]any.
type ExValues struct {
	queryOrder   []string
	queryValues  map[string][]string
	bodyOrder    []string
	bodyValues   map[string][]string
	headerOrder  []string
	headerValues map[string][]string
}

// NewExValues creates a new ExValues instance.
func NewExValues() *ExValues {
	return &ExValues{
		queryOrder:   make([]string, 0),
		queryValues:  make(map[string][]string),
		bodyOrder:    make([]string, 0),
		bodyValues:   make(map[string][]string),
		headerOrder:  make([]string, 0),
		headerValues: make(map[string][]string),
	}
}

// ============================================================================
// Query methods
// ============================================================================

// SetQuery sets a single value for the given key.
// If the key appears for the first time, its position is recorded in queryOrder.
//
// It supports any value type and will infer a string representation:
//   - string / []byte / json.RawMessage
//   - bool / all ints / all uints / floats
//   - time.Time (RFC3339Nano)
//   - decimal.Decimal / *decimal.Decimal
//   - fmt.Stringer / error
//   - slice/array: expands to multiple values (replaces existing values)
func (v *ExValues) SetQuery(key string, value any) {
	if _, exists := v.queryValues[key]; !exists {
		v.queryOrder = append(v.queryOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.queryValues[key] = vs
}

// AddQuery appends a value for the given key.
// The key's order is preserved based on its first appearance.
//
// It supports any value type; slice/array expands to multiple appended values.
func (v *ExValues) AddQuery(key string, value any) {
	if _, exists := v.queryValues[key]; !exists {
		v.queryOrder = append(v.queryOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.queryValues[key] = append(v.queryValues[key], vs...)
}

// HasQuery reports whether the given key exists in query parameters.
func (v *ExValues) HasQuery(key string) bool {
	_, ok := v.queryValues[key]
	return ok
}

// GetQuery returns the first value associated with the given query key.
func (v *ExValues) GetQuery(key string) string {
	if vs := v.queryValues[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// EncodeQuery encodes parameters as a URL query string.
// The output preserves the original insertion order of keys.
func (v *ExValues) EncodeQuery() string {
	if len(v.queryOrder) == 0 {
		return ""
	}

	var buf strings.Builder

	for _, key := range v.queryOrder {
		vs, ok := v.queryValues[key]
		if !ok {
			continue
		}

		keyEscaped := url.QueryEscape(key)

		for _, value := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(value))
		}
	}

	return buf.String()
}

// ============================================================================
// Body methods
// ============================================================================

// SetBody sets a single value for the given key in the body.
// If the key appears for the first time, its position is recorded in bodyOrder.
//
// It supports any value type and will infer a string representation:
//   - string / []byte / json.RawMessage
//   - bool / all ints / all uints / floats
//   - time.Time (RFC3339Nano)
//   - decimal.Decimal / *decimal.Decimal
//   - fmt.Stringer / error
//   - slice/array: expands to multiple values (replaces existing values)
func (v *ExValues) SetBody(key string, value any) {
	if _, exists := v.bodyValues[key]; !exists {
		v.bodyOrder = append(v.bodyOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.bodyValues[key] = vs
}

// AddBody appends a value for the given key in the body.
// The key's order is preserved based on its first appearance.
//
// It supports any value type; slice/array expands to multiple appended values.
func (v *ExValues) AddBody(key string, value any) {
	if _, exists := v.bodyValues[key]; !exists {
		v.bodyOrder = append(v.bodyOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.bodyValues[key] = append(v.bodyValues[key], vs...)
}

// HasBody reports whether the given key exists in the body.
func (v *ExValues) HasBody(key string) bool {
	_, ok := v.bodyValues[key]
	return ok
}

// GetBody returns the first value associated with the given body key.
func (v *ExValues) GetBody(key string) string {
	if vs := v.bodyValues[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// EncodeBody encodes body parameters into a map representation.
//
//   - single value  -> string
//   - multiple values -> []string
//
// This representation is useful for JSON, logging, or custom encoders.
func (v *ExValues) EncodeBody() map[string]any {
	m := make(map[string]any, len(v.bodyValues))

	for _, key := range v.bodyOrder {
		vs := v.bodyValues[key]
		if len(vs) == 1 {
			m[key] = vs[0]
		} else if len(vs) > 1 {
			m[key] = vs
		}
	}

	return m
}

// ============================================================================
// Header methods
// ============================================================================

// SetHeader sets a single value for the given header key.
// If the key appears for the first time, its position is recorded in headerOrder.
func (v *ExValues) SetHeader(key string, value any) {
	if _, exists := v.headerValues[key]; !exists {
		v.headerOrder = append(v.headerOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.headerValues[key] = vs
}

// AddHeader appends a value for the given header key.
// The key's order is preserved based on its first appearance.
func (v *ExValues) AddHeader(key string, value any) {
	if _, exists := v.headerValues[key]; !exists {
		v.headerOrder = append(v.headerOrder, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.headerValues[key] = append(v.headerValues[key], vs...)
}

// HasHeader reports whether the given header key exists.
func (v *ExValues) HasHeader(key string) bool {
	_, ok := v.headerValues[key]
	return ok
}

// GetHeader returns the first value associated with the given header key.
func (v *ExValues) GetHeader(key string) string {
	if vs := v.headerValues[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// EncodeHeader encodes headers into a map representation.
//
//   - single value  -> string
//   - multiple values -> []string
//
// This representation is useful for JSON, logging, or custom encoders.
func (v *ExValues) EncodeHeader() map[string]any {
	m := make(map[string]any, len(v.headerValues))

	for _, key := range v.headerOrder {
		vs := v.headerValues[key]
		if len(vs) == 1 {
			m[key] = vs[0]
		} else if len(vs) > 1 {
			m[key] = vs
		}
	}

	return m
}

// ============================================================================
// Utility methods
// ============================================================================

// JoinPath joins the encoded query string to the given path.
func (v *ExValues) JoinPath(path string) string {
	query := v.EncodeQuery()
	if query == "" {
		return path
	}

	if strings.Contains(path, "?") {
		return path + "&" + query
	}

	return path + "?" + query
}

// Reset clears all stored parameters (query, body, and headers).
func (v *ExValues) Reset() {
	v.queryOrder = v.queryOrder[:0]
	v.queryValues = make(map[string][]string)
	v.bodyOrder = v.bodyOrder[:0]
	v.bodyValues = make(map[string][]string)
	v.headerOrder = v.headerOrder[:0]
	v.headerValues = make(map[string][]string)
}

// ============================================================================
// Helper functions
// ============================================================================

func inferStrings(v any) ([]string, bool) {
	if v == nil {
		return []string{""}, true
	}

	// Fast path for common, unambiguous types.
	switch x := v.(type) {
	case string:
		return []string{x}, true
	case []string:
		// Treat as multi-value.
		cp := make([]string, len(x))
		copy(cp, x)
		return cp, true
	case []byte:
		return []string{string(x)}, true
	case json.RawMessage:
		return []string{string(x)}, true
	case bool:
		return []string{strconv.FormatBool(x)}, true
	case int:
		return []string{strconv.FormatInt(int64(x), 10)}, true
	case int8:
		return []string{strconv.FormatInt(int64(x), 10)}, true
	case int16:
		return []string{strconv.FormatInt(int64(x), 10)}, true
	case int32:
		return []string{strconv.FormatInt(int64(x), 10)}, true
	case int64:
		return []string{strconv.FormatInt(x, 10)}, true
	case uint:
		return []string{strconv.FormatUint(uint64(x), 10)}, true
	case uint8:
		return []string{strconv.FormatUint(uint64(x), 10)}, true
	case uint16:
		return []string{strconv.FormatUint(uint64(x), 10)}, true
	case uint32:
		return []string{strconv.FormatUint(uint64(x), 10)}, true
	case uint64:
		return []string{strconv.FormatUint(x, 10)}, true
	case float32:
		return []string{strconv.FormatFloat(float64(x), 'f', -1, 32)}, true
	case float64:
		return []string{strconv.FormatFloat(x, 'f', -1, 64)}, true
	case time.Time:
		return []string{x.UTC().Format(time.RFC3339Nano)}, true
	case decimal.Decimal:
		return []string{x.String()}, true
	case *decimal.Decimal:
		if x == nil {
			return []string{""}, true
		}
		return []string{x.String()}, true
	case fmt.Stringer:
		return []string{x.String()}, true
	case error:
		return []string{x.Error()}, true
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return []string{""}, true
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		// Expand to multiple values.
		n := rv.Len()
		out := make([]string, 0, n)
		for i := 0; i < n; i++ {
			ss, ok := inferStrings(rv.Index(i).Interface())
			if !ok || len(ss) == 0 {
				out = append(out, "")
				continue
			}
			// If nested slice, flatten.
			out = append(out, ss...)
		}
		return out, true
	}

	// Fallback: stable, readable string form.
	return []string{fmt.Sprint(v)}, true
}
