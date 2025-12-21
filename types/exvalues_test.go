package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestExValues_SetQuery_TypeInferenceAndOrder(t *testing.T) {
	v := NewExValues()

	d := decimal.NewFromFloat(12.34)
	ts := time.Date(2025, 12, 19, 1, 2, 3, 4, time.FixedZone("CST", 8*3600))

	v.SetQuery("a", 1)
	v.SetQuery("b", true)
	v.SetQuery("c", d)
	v.SetQuery("d", ts)
	v.SetQuery("e", json.RawMessage(`{"x":1}`))

	// key order should be preserved based on first appearance
	gotQuery := v.EncodeQuery()
	wantQueryPrefix := "a=1&b=true&c=12.34&d="
	if len(gotQuery) < len(wantQueryPrefix) || gotQuery[:len(wantQueryPrefix)] != wantQueryPrefix {
		t.Fatalf("unexpected query prefix:\n got: %q\nwant prefix: %q", gotQuery, wantQueryPrefix)
	}
	if v.GetQuery("a") != "1" {
		t.Fatalf("GetQuery(a)=%q, want %q", v.GetQuery("a"), "1")
	}
	if v.GetQuery("b") != "true" {
		t.Fatalf("GetQuery(b)=%q, want %q", v.GetQuery("b"), "true")
	}
	if v.GetQuery("c") != "12.34" {
		t.Fatalf("GetQuery(c)=%q, want %q", v.GetQuery("c"), "12.34")
	}
	// time is always formatted in UTC RFC3339Nano
	if v.GetQuery("d") != ts.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("GetQuery(d)=%q, want %q", v.GetQuery("d"), ts.UTC().Format(time.RFC3339Nano))
	}
	if v.GetQuery("e") != `{"x":1}` {
		t.Fatalf("GetQuery(e)=%q, want %q", v.GetQuery("e"), `{"x":1}`)
	}
}

func TestExValues_SetQuery_SliceExpandsAndReplaces(t *testing.T) {
	v := NewExValues()

	v.SetQuery("k", []int{1, 2, 3})
	if got := v.EncodeQuery(); got != "k=1&k=2&k=3" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=1&k=2&k=3")
	}

	// SetQuery should replace existing values
	v.SetQuery("k", []string{"a"})
	if got := v.EncodeQuery(); got != "k=a" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=a")
	}
}

func TestExValues_AddQuery_AppendsAndExpands(t *testing.T) {
	v := NewExValues()

	v.AddQuery("k", 1)
	v.AddQuery("k", []int{2, 3})
	v.AddQuery("k", "4")

	if got := v.EncodeQuery(); got != "k=1&k=2&k=3&k=4" {
		t.Fatalf("EncodeQuery()=%q, want %q", got, "k=1&k=2&k=3&k=4")
	}
}

func TestExValues_EncodeHeader_AndJoinPath(t *testing.T) {
	v := NewExValues()

	v.SetHeader("single", 1)
	v.AddHeader("multi", "a")
	v.AddHeader("multi", "b")

	m := v.EncodeHeader()
	if got, ok := m["single"].(string); !ok || got != "1" {
		t.Fatalf("EncodeHeader()[single]=(%T)%v, want string %q", m["single"], m["single"], "1")
	}
	if got, ok := m["multi"].([]string); !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("EncodeHeader()[multi]=(%T)%v, want []string{a,b}", m["multi"], m["multi"])
	}

	// Test JoinPath with query parameters
	v2 := NewExValues()
	v2.SetQuery("single", 1)
	v2.AddQuery("multi", "a")
	v2.AddQuery("multi", "b")

	if got := v2.JoinPath("/path"); got != "/path?single=1&multi=a&multi=b" {
		t.Fatalf("JoinPath(/path)=%q, want %q", got, "/path?single=1&multi=a&multi=b")
	}
	if got := v2.JoinPath("/path?x=1"); got != "/path?x=1&single=1&multi=a&multi=b" {
		t.Fatalf("JoinPath(/path?x=1)=%q, want %q", got, "/path?x=1&single=1&multi=a&multi=b")
	}
}

func TestExValues_HasQuery_GetQuery_Reset(t *testing.T) {
	v := NewExValues()
	if v.HasQuery("k") {
		t.Fatalf("HasQuery(k)=true, want false")
	}
	if v.GetQuery("k") != "" {
		t.Fatalf("GetQuery(k)=%q, want empty", v.GetQuery("k"))
	}

	v.SetQuery("k", "v")
	if !v.HasQuery("k") {
		t.Fatalf("HasQuery(k)=false, want true")
	}
	if v.GetQuery("k") != "v" {
		t.Fatalf("GetQuery(k)=%q, want %q", v.GetQuery("k"), "v")
	}

	v.Reset()
	if v.HasQuery("k") {
		t.Fatalf("HasQuery(k)=true after Reset, want false")
	}
	if got := v.EncodeQuery(); got != "" {
		t.Fatalf("EncodeQuery()=%q after Reset, want empty", got)
	}
}

func TestExValues_Body(t *testing.T) {
	v := NewExValues()

	if v.HasBody("k") {
		t.Fatalf("HasBody(k)=true, want false")
	}
	if v.GetBody("k") != "" {
		t.Fatalf("GetBody(k)=%q, want empty", v.GetBody("k"))
	}

	v.SetBody("k", 123)
	if !v.HasBody("k") {
		t.Fatalf("HasBody(k)=false, want true")
	}
	if v.GetBody("k") != "123" {
		t.Fatalf("GetBody(k)=%q, want %q", v.GetBody("k"), "123")
	}

	body := v.EncodeBody()
	if got, ok := body["k"].(string); !ok || got != "123" {
		t.Fatalf("EncodeBody()[k]=(%T)%v, want string %q", body["k"], body["k"], "123")
	}

	v.AddBody("k", "value")
	body = v.EncodeBody()
	if got, ok := body["k"].([]string); !ok || len(got) != 2 || got[0] != "123" || got[1] != "value" {
		t.Fatalf("EncodeBody()[k]=(%T)%v, want []string{123,value}", body["k"], body["k"])
	}
	if v.GetBody("k") != "123" {
		t.Fatalf("GetBody(k)=%q, want %q", v.GetBody("k"), "123")
	}

	v.SetBody("single", "test")
	v.AddBody("multi", 1)
	v.AddBody("multi", 2)

	body = v.EncodeBody()
	if got, ok := body["single"].(string); !ok || got != "test" {
		t.Fatalf("EncodeBody()[single]=(%T)%v, want string %q", body["single"], body["single"], "test")
	}
	if got, ok := body["multi"].([]string); !ok || len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Fatalf("EncodeBody()[multi]=(%T)%v, want []string{1,2}", body["multi"], body["multi"])
	}

	v.Reset()
	if v.HasBody("k") {
		t.Fatalf("HasBody(k)=true after Reset, want false")
	}
	if len(v.EncodeBody()) != 0 {
		t.Fatalf("EncodeBody() should be empty after Reset")
	}
}

func TestExValues_Header(t *testing.T) {
	v := NewExValues()

	if v.HasHeader("X-Test") {
		t.Fatalf("HasHeader(X-Test)=true, want false")
	}
	if v.GetHeader("X-Test") != "" {
		t.Fatalf("GetHeader(X-Test)=%q, want empty", v.GetHeader("X-Test"))
	}

	v.SetHeader("X-Test", "value1")
	if !v.HasHeader("X-Test") {
		t.Fatalf("HasHeader(X-Test)=false, want true")
	}
	if v.GetHeader("X-Test") != "value1" {
		t.Fatalf("GetHeader(X-Test)=%q, want %q", v.GetHeader("X-Test"), "value1")
	}

	headers := v.EncodeHeader()
	if got, ok := headers["X-Test"].(string); !ok || got != "value1" {
		t.Fatalf("EncodeHeader()[X-Test]=(%T)%v, want string %q", headers["X-Test"], headers["X-Test"], "value1")
	}

	v.AddHeader("X-Test", "value2")
	headers = v.EncodeHeader()
	if got, ok := headers["X-Test"].([]string); !ok || len(got) != 2 || got[0] != "value1" || got[1] != "value2" {
		t.Fatalf("EncodeHeader()[X-Test]=(%T)%v, want []string{value1,value2}", headers["X-Test"], headers["X-Test"])
	}
	if v.GetHeader("X-Test") != "value1" {
		t.Fatalf("GetHeader(X-Test)=%q, want %q", v.GetHeader("X-Test"), "value1")
	}

	v.SetHeader("X-Single", "single")
	v.AddHeader("X-Multi", "a")
	v.AddHeader("X-Multi", "b")

	headers = v.EncodeHeader()
	if got, ok := headers["X-Single"].(string); !ok || got != "single" {
		t.Fatalf("EncodeHeader()[X-Single]=(%T)%v, want string %q", headers["X-Single"], headers["X-Single"], "single")
	}
	if got, ok := headers["X-Multi"].([]string); !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("EncodeHeader()[X-Multi]=(%T)%v, want []string{a,b}", headers["X-Multi"], headers["X-Multi"])
	}

	v.Reset()
	if v.HasHeader("X-Test") {
		t.Fatalf("HasHeader(X-Test)=true after Reset, want false")
	}
	if len(v.EncodeHeader()) != 0 {
		t.Fatalf("EncodeHeader() should be empty after Reset")
	}
}
