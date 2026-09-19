package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderPageDirectNavigationWrapsAdminShell(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	rec := httptest.NewRecorder()

	err := RenderPage(rec, req, "Test Sayfası", "Test", "dashboard", func(w io.Writer) error {
		_, err := w.Write([]byte(`<div class="test-component">İçerik</div>`))
		return err
	})
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<!doctype html>") {
		t.Errorf("expected <!doctype html> in direct navigation, got: %s", body)
	}
	if !strings.Contains(body, `data-ui-theme=`) {
		t.Errorf("expected data-ui-theme attribute in direct navigation, got: %s", body)
	}
	if !strings.Contains(body, `<div class="test-component">İçerik</div>`) {
		t.Errorf("expected component content, got: %s", body)
	}
	if !strings.Contains(body, "</html>") {
		t.Errorf("expected closing html tag, got: %s", body)
	}
}

func TestRenderPageHTMXPartialRendersBodyOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	err := RenderPage(rec, req, "Test Sayfası", "Test", "dashboard", func(w io.Writer) error {
		_, err := w.Write([]byte(`<div class="test-component">Kısmi İçerik</div>`))
		return err
	})
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Errorf("did not expect <!doctype html> in HTMX partial, got: %s", body)
	}
	if strings.Contains(body, "studio-sidebar") {
		t.Errorf("did not expect studio-sidebar in HTMX partial, got: %s", body)
	}
	if !strings.Contains(body, `<div class="test-component">Kısmi İçerik</div>`) {
		t.Errorf("expected component content, got: %s", body)
	}
}
