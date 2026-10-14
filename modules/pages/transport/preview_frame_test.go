package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

// The page editor's device preview loads /preview/pages/{id} in a
// same-origin iframe. That response must allow same-origin framing while
// every other response (including a rejected preview) keeps 'none'.
func TestDraftPreviewResponseIsFrameableBySameOriginOnly(t *testing.T) {
	page := pages.Page{ID: "page-1", Locale: "en", Slug: "draft", Title: "Draft title", Body: "Draft body", Status: pages.StatusDraft, Revision: 2}
	signer := &preview.Signer{Key: []byte(strings.Repeat("k", 32))}
	handler := Handler{Content: service.ContentService{Pages: aiPageRepository{page}}, PreviewSigner: signer}
	serve := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/preview/pages/page-1?token="+url.QueryEscape(token), nil)
		request.SetPathValue("id", "page-1")
		response := httptest.NewRecorder()
		security.HeadersMiddleware(security.HeaderConfig{Preset: security.HeaderPresetStrict})(http.HandlerFunc(handler.publicDraftPreview)).ServeHTTP(response, request)
		return response
	}
	token, err := signer.Issue(preview.Claims{PageID: "page-1", Revision: 2}, time.Now().UTC(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ok := serve(token)
	if ok.Code != http.StatusOK || !strings.Contains(ok.Body.String(), "Draft title") {
		t.Fatalf("preview status=%d body=%s", ok.Code, ok.Body.String())
	}
	if csp := ok.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'self'") || strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("draft preview CSP must allow same-origin framing: %s", csp)
	}
	if got := ok.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("draft preview X-Frame-Options=%q", got)
	}

	denied := serve("not-a-token")
	if denied.Code != http.StatusNotFound {
		t.Fatalf("invalid token status=%d", denied.Code)
	}
	if csp := denied.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("rejected preview must keep frame-ancestors 'none': %s", csp)
	}
	if got := denied.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("rejected preview X-Frame-Options=%q", got)
	}
}
