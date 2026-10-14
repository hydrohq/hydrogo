package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type previewNewsRepo struct {
	publicNewsRepo
}

func (r previewNewsRepo) Get(_ context.Context, _ database.Executor, id string) (news.Item, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return news.Item{}, errors.New("not found")
}

func serveNewsHTML(t *testing.T, h Handler, rawURL string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, rawURL, nil)
	request.SetPathValue("locale", "en")
	request.SetPathValue("slug", "first-story")
	response := httptest.NewRecorder()
	security.HeadersMiddleware(security.HeaderConfig{Preset: security.HeaderPresetStrict})(http.HandlerFunc(h.publicGetHTML)).ServeHTTP(response, request)
	return response
}

// The admin "Önizle" dialog embeds the token-carrying article URL in a
// same-origin iframe. Only that response may be framed; the plain public
// article keeps the strict preset's frame-ancestors 'none'.
func TestNewsPreviewResponseIsFrameableBySameOriginOnly(t *testing.T) {
	h := publicNewsHandler(t, loadShippedTheme(t, "newsroom"))
	repo := h.Service.Repository.(publicNewsRepo)
	for i := range repo.items {
		repo.items[i].Locale, repo.items[i].Status, repo.items[i].Revision = "en", news.StatusPublished, 3
	}
	h.Service.Repository = previewNewsRepo{repo}
	h.PreviewSigner = &preview.Signer{Key: []byte(strings.Repeat("k", 32))}
	token, err := h.PreviewSigner.Issue(preview.Claims{ResourceType: "news", ResourceID: "a", Revision: 3}, time.Now().UTC(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	previewed := serveNewsHTML(t, h, "/public/news/en/first-story/html?preview_token="+url.QueryEscape(token))
	if previewed.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", previewed.Code, previewed.Body.String())
	}
	if csp := previewed.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'self'") || strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("preview CSP must allow same-origin framing: %s", csp)
	}
	if got := previewed.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("preview X-Frame-Options=%q", got)
	}

	public := serveNewsHTML(t, h, "/public/news/en/first-story/html")
	if public.Code != http.StatusOK {
		t.Fatalf("public status=%d", public.Code)
	}
	if csp := public.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("public article must stay unframeable: %s", csp)
	}
	if got := public.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("public X-Frame-Options=%q", got)
	}

	stale, err := h.PreviewSigner.Issue(preview.Claims{ResourceType: "news", ResourceID: "a", Revision: 2}, time.Now().UTC(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	rejected := serveNewsHTML(t, h, "/public/news/en/first-story/html?preview_token="+url.QueryEscape(stale))
	if rejected.Code != http.StatusNotFound || strings.Contains(rejected.Header().Get("Content-Security-Policy"), "frame-ancestors 'self'") {
		t.Fatalf("stale preview token must 404 without relaxing framing: status=%d csp=%s", rejected.Code, rejected.Header().Get("Content-Security-Policy"))
	}
}
