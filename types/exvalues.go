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
//   - order keeps the first-seen order of keys.
//   - values stores one or more values per key.
//   - EncodeQuery preserves key order and value order.
//   - EncodeJSON / EncodeMap share the same semantic model.
type ExValues struct {
	order  []string
	values map[string][]string
}

// NewExValues creates a new ExValues instance.
func NewExValues() *ExValues {
	return &ExValues{
		order:  make([]string, 0),
		values: make(map[string][]string),
	}
}

// Set sets a single value for the given key.
// If the key appears for the first time, its position is recorded in order.
//
// It supports any value type and will infer a string representation:
//   - string / []byte / json.RawMessage
//   - bool / all ints / all uints / floats
//   - time.Time (RFC3339Nano)
//   - decimal.Decimal / *decimal.Decimal
//   - fmt.Stringer / error
//   - slice/array: expands to multiple values (replaces existing values)
func (v *ExValues) Set(key string, value any) {
	if _, exists := v.values[key]; !exists {
		v.order = append(v.order, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.values[key] = vs
}

// Add appends a value for the given key.
// The key's order is preserved based on its first appearance.
//
// It supports any value type; slice/array expands to multiple appended values.
func (v *ExValues) Add(key string, value any) {
	if _, exists := v.values[key]; !exists {
		v.order = append(v.order, key)
	}

	vs, ok := inferStrings(value)
	if !ok {
		vs = []string{""}
	}
	v.values[key] = append(v.values[key], vs...)
}

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

// EncodeQuery encodes parameters as a URL query string.
// The output preserves the original insertion order of keys.
func (v *ExValues) EncodeQuery() string {
	if len(v.order) == 0 {
		return ""
	}

	var buf strings.Builder

	for _, key := range v.order {
		vs, ok := v.values[key]
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

// EncodeMap encodes parameters into a map representation.
//
//   - single value  -> string
//   - multiple values -> []string
//
// This representation is useful for JSON, logging, or custom encoders.
func (v *ExValues) EncodeMap() map[string]any {
	m := make(map[string]any, len(v.values))

	for _, key := range v.order {
		vs := v.values[key]
		if len(vs) == 1 {
			m[key] = vs[0]
		} else if len(vs) > 1 {
			m[key] = vs
		}
	}

	return m
}

// EncodeJSON encodes parameters into a JSON byte slice.
func (v *ExValues) EncodeJSON() ([]byte, error) {
	return json.Marshal(v.EncodeMap())
}

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

// Has reports whether the given key exists.
func (v *ExValues) Has(key string) bool {
	_, ok := v.values[key]
	return ok
}

// Get returns the first value associated with the given key.
func (v *ExValues) Get(key string) string {
	if vs := v.values[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// Reset clears all stored parameters.
func (v *ExValues) Reset() {
	v.order = v.order[:0]
	v.values = make(map[string][]string)
}
