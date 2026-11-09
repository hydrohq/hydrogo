package storefront

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderWelcomeIsNeutral(t *testing.T) {
	rec := httptest.NewRecorder()
	RenderWelcome(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Server is running") || !strings.Contains(body, `href="/admin/"`) {
		t.Fatalf("unexpected welcome page: %d %q", rec.Code, body)
	}
	for _, banned := range []string{"HydroCloud", "onboard", "Pro Plan", "₺"} {
		if strings.Contains(body, banned) {
			t.Fatalf("welcome page must stay neutral, found %q", banned)
		}
	}
}
