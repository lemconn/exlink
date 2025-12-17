package types

import (
	"encoding/json"
	"net/url"
	"strings"
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
func (v *ExValues) Set(key, value string) {
	if _, exists := v.values[key]; !exists {
		v.order = append(v.order, key)
	}
	v.values[key] = []string{value}
}

// Add appends a value for the given key.
// The key's order is preserved based on its first appearance.
func (v *ExValues) Add(key, value string) {
	if _, exists := v.values[key]; !exists {
		v.order = append(v.order, key)
	}
	v.values[key] = append(v.values[key], value)
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
