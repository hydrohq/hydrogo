package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type decodeTarget struct {
	Name string `json:"name"`
}

func TestDecodeJSONAcceptsAValidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if !DecodeJSON(recorder, req, &dst, 0) {
		t.Fatalf("expected success, got status %d body %q", recorder.Code, recorder.Body.String())
	}
	if dst.Name != "ok" {
		t.Fatalf("name = %q", dst.Name)
	}
}

func TestDecodeJSONRejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if DecodeJSON(recorder, req, &dst, 0) {
		t.Fatal("expected malformed JSON to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

// TestDecodeJSONRejectsTrailingGarbage covers the second Decode() call's
// purpose: a body like {"name":"ok"}{"name":"smuggled"} must not be
// silently accepted with only the first object applied -- a client that
// can get a handler to read only the first of two concatenated JSON
// objects could use this to desynchronize what the handler validated from
// what a downstream consumer of the raw body later sees.
func TestDecodeJSONRejectsTrailingGarbage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}{"name":"smuggled"}`))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if DecodeJSON(recorder, req, &dst, 0) {
		t.Fatal("expected trailing data after the JSON object to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestDecodeJSONAllowsTrailingWhitespace(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{\"name\":\"ok\"}\n\t \n"))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if !DecodeJSON(recorder, req, &dst, 0) {
		t.Fatalf("expected trailing whitespace to be allowed, got status %d body %q", recorder.Code, recorder.Body.String())
	}
}

func TestDecodeJSONRejectsABodyOverTheLimitWith413(t *testing.T) {
	body := `{"name":"` + strings.Repeat("a", 100) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if DecodeJSON(recorder, req, &dst, 10) {
		t.Fatal("expected an oversized body to be rejected")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
}

func TestDecodeJSONFallsBackToDefaultMaxBytesWhenNonPositive(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	recorder := httptest.NewRecorder()
	var dst decodeTarget
	if !DecodeJSON(recorder, req, &dst, -1) {
		t.Fatalf("expected a non-positive maxBytes to fall back to DefaultMaxBodyBytes, got status %d", recorder.Code)
	}
}

func TestWriteDataWrapsInTheStandardEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteData(recorder, http.StatusCreated, map[string]string{"id": "1"}, PaginationMeta{Total: 1})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	var decoded ResponseData
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	data, ok := decoded.Data.(map[string]any)
	if !ok || data["id"] != "1" {
		t.Fatalf("data = %+v", decoded.Data)
	}
}

func TestWriteListIncludesPaginationMetadata(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteList(recorder, []string{"a", "b"}, PaginationMeta{Limit: 2, Offset: 0, HasMore: true})
	var decoded struct {
		Data     []string       `json:"data"`
		Metadata PaginationMeta `json:"metadata"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Data) != 2 || !decoded.Metadata.HasMore || decoded.Metadata.Limit != 2 {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestWriteErrorCarriesTheRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(security.WithRequestID(req.Context(), "req-123"))
	recorder := httptest.NewRecorder()
	WriteError(recorder, req, http.StatusNotFound, "not found", nil)

	var decoded security.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RequestID != "req-123" {
		t.Fatalf("request_id = %q, want it propagated from the request context", decoded.RequestID)
	}
	if decoded.Message != "not found" {
		t.Fatalf("message = %q", decoded.Message)
	}
}

func TestWriteValidationAnswers422WithFieldCodes(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	WriteValidation(recorder, req, map[string]string{"email": "required"})

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
	var decoded security.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Fields["email"] != "required" {
		t.Fatalf("fields = %+v", decoded.Fields)
	}
}
