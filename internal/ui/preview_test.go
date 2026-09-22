package ui

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewHandlerBoundsInputAndUsesNoStore(t *testing.T) {
	handler := PreviewHandler(func(document []byte) (template.HTML, error) {
		if string(document) != `{"blocks":[]}` {
			t.Fatalf("unexpected document: %s", document)
		}
		return template.HTML("<p>safe</p>"), nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/preview", strings.NewReader(`{"blocks":[]}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "private, no-store" || rec.Body.String() != "<p>safe</p>" {
		t.Fatalf("unexpected preview response: %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}

	tooLarge := strings.Repeat("x", int(MaxPreviewPayloadBytes)+1)
	req = httptest.NewRequest(http.MethodPost, "/api/preview", strings.NewReader(tooLarge))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected bounded payload rejection, got %d", rec.Code)
	}
}

func TestPreviewHandlerRedactsRendererErrors(t *testing.T) {
	handler := PreviewHandler(func([]byte) (template.HTML, error) { return "", errors.New("database secret") })
	req := httptest.NewRequest(http.MethodPost, "/api/preview", strings.NewReader(`{"blocks":[]}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || strings.Contains(rec.Body.String(), "database secret") {
		t.Fatalf("renderer error leaked: %d %s", rec.Code, rec.Body.String())
	}
}
