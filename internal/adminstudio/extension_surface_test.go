package adminstudio

import (
	"net/http/httptest"
	"strings"
	"testing"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestServeExtensionSurfaceDoesNotExposeSecrets(t *testing.T) {
	manifest := extensionv1.AdminExtensionManifest{ID: "reports", Name: "Reports", Version: "1", EntrypointURL: "https://reports.example", Surfaces: []extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "/", Placement: "reports", Permissions: []string{"reports.read"}}}}
	registry, err := NewExtensionRegistry([]extensionv1.AdminExtensionManifest{manifest})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/admin/extensions/reports/home", nil)
	recorder := httptest.NewRecorder()
	registry.ServeExtensionSurface(recorder, req, "reports", "home", "tr-TR")
	body := recorder.Body.String()
	if recorder.Code != 200 || !strings.Contains(body, "https://reports.example/") || !strings.Contains(body, "reports.read") || strings.Contains(body, "Authorization") || strings.Contains(body, "hydro_session") {
		t.Fatalf("unexpected extension shell: status=%d body=%s", recorder.Code, body)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("missing security headers: %#v", recorder.Header())
	}
}
