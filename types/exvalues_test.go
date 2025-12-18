package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestExValues_Set_TypeInferenceAndOrder(t *testing.T) {
	v := NewExValues()

	d := decimal.NewFromFloat(12.34)
	ts := time.Date(2025, 12, 19, 1, 2, 3, 4, time.FixedZone("CST", 8*3600))

	v.Set("a", 1)
	v.Set("b", true)
	v.Set("c", d)
	v.Set("d", ts)
	v.Set("e", json.RawMessage(`{"x":1}`))

	// key order should be preserved based on first appearance
	gotQuery := v.EncodeQuery()
	wantQueryPrefix := "a=1&b=true&c=12.34&d="
	if len(gotQuery) < len(wantQueryPrefix) || gotQuery[:len(wantQueryPrefix)] != wantQueryPrefix {
		t.Fatalf("unexpected query prefix:\n got: %q\nwant prefix: %q", gotQuery, wantQueryPrefix)
	}
	if v.Get("a") != "1" {
		t.Fatalf("Get(a)=%q, want %q", v.Get("a"), "1")
	}
	if v.Get("b") != "true" {
		t.Fatalf("Get(b)=%q, want %q", v.Get("b"), "true")
	}
	if v.Get("c") != "12.34" {
		t.Fatalf("Get(c)=%q, want %q", v.Get("c"), "12.34")
	}
	// time is always formatted in UTC RFC3339Nano
	if v.Get("d") != ts.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("Get(d)=%q, want %q", v.Get("d"), ts.UTC().Format(time.RFC3339Nano))
	}
	if v.Get("e") != `{"x":1}` {
		t.Fatalf("Get(e)=%q, want %q", v.Get("e"), `{"x":1}`)
	}
}

func TestExValues_Set_SliceExpandsAndReplaces(t *testing.T) {
	v := NewExValues()

	v.Set("k", []int{1, 2, 3})
	if got := v.EncodeQuery(); got != "k=1&k=2&k=3" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=1&k=2&k=3")
	}

	// Set should replace existing values
	v.Set("k", []string{"a"})
	if got := v.EncodeQuery(); got != "k=a" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=a")
	}
}

func TestExValues_Add_AppendsAndExpands(t *testing.T) {
	v := NewExValues()

	v.Add("k", 1)
	v.Add("k", []int{2, 3})
	v.Add("k", "4")

	if got := v.EncodeQuery(); got != "k=1&k=2&k=3&k=4" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=1&k=2&k=3&k=4")
	}
}

func TestExValues_EncodeMap_AndJoinPath(t *testing.T) {
	v := NewExValues()

	v.Set("single", 1)
	v.Add("multi", "a")
	v.Add("multi", "b")

	m := v.EncodeMap()
	if got, ok := m["single"].(string); !ok || got != "1" {
		t.Fatalf("EncodeMap()[single]=(%T)%v, want string %q", m["single"], m["single"], "1")
	}
	if got, ok := m["multi"].([]string); !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("EncodeMap()[multi]=(%T)%v, want []string{a,b}", m["multi"], m["multi"])
	}

	if got := v.JoinPath("/path"); got != "/path?single=1&multi=a&multi=b" {
		t.Fatalf("JoinPath(/path)=%q, want %q", got, "/path?single=1&multi=a&multi=b")
	}
	if got := v.JoinPath("/path?x=1"); got != "/path?x=1&single=1&multi=a&multi=b" {
		t.Fatalf("JoinPath(/path?x=1)=%q, want %q", got, "/path?x=1&single=1&multi=a&multi=b")
	}
}

func TestExValues_Has_Get_Reset(t *testing.T) {
	v := NewExValues()
	if v.Has("k") {
		t.Fatalf("Has(k)=true, want false")
	}
	if v.Get("k") != "" {
		t.Fatalf("Get(k)=%q, want empty", v.Get("k"))
	}

	v.Set("k", "v")
	if !v.Has("k") {
		t.Fatalf("Has(k)=false, want true")
	}
	if v.Get("k") != "v" {
		t.Fatalf("Get(k)=%q, want %q", v.Get("k"), "v")
	}

	v.Reset()
	if v.Has("k") {
		t.Fatalf("Has(k)=true after Reset, want false")
	}
	if got := v.EncodeQuery(); got != "" {
		t.Fatalf("EncodeQuery()=%q after Reset, want empty", got)
	}
}
