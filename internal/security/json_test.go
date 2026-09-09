package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONStrictRejectsUnknownAndTrailingValues(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	for _, body := range []string{`{"name":"ok","role":"admin"}`, `{"name":"ok"}{"name":"again"}`} {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		res := httptest.NewRecorder()
		if err := DecodeJSONStrict(res, req, new(input), 1<<10); err == nil {
			t.Fatalf("body %q was accepted", body)
		}
	}
}

func TestDecodeJSONStrictAcceptsSingleKnownValue(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	var got input
	if err := DecodeJSONStrict(httptest.NewRecorder(), req, &got, 1<<10); err != nil {
		t.Fatal(err)
	}
	if got.Name != "ok" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestDecodeJSONStrictFailsClosedOnInvalidRequestInputs(t *testing.T) {
	valid := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	for _, tc := range []struct {
		name string
		req  *http.Request
		max  int64
	}{
		{name: "nil request", req: nil, max: 1 << 10},
		{name: "nil body", req: &http.Request{}, max: 1 << 10},
		{name: "zero limit", req: valid, max: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := DecodeJSONStrict(httptest.NewRecorder(), tc.req, &struct{}{}, tc.max); err != ErrInvalidJSON {
				t.Fatalf("error = %v, want %v", err, ErrInvalidJSON)
			}
		})
	}
}
