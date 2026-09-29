package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
)

func TestAdminCanStartAndReviewTenantLinkScan(t *testing.T) {
	publicSite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<title>Home</title><a href="/missing">Missing page</a>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer publicSite.Close()
	manager := linkcheck.NewManager(linkcheck.Options{MaxPages: 10, MaxTargets: 20, RequestDelay: time.Millisecond, Client: publicSite.Client()})
	handler := Handler{SitemapOrigin: publicSite.URL, LinkScanManager: manager}
	userCtx := auth.WithUser(httptest.NewRequest(http.MethodPost, "/admin/site/link-check", nil).Context(), &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true})
	request := httptest.NewRequest(http.MethodPost, "/admin/site/link-check", nil).WithContext(userCtx)
	response := httptest.NewRecorder()
	handler.runLinkCheck(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	jobID := location.Query().Get("job")
	if jobID == "" {
		t.Fatal("job ID missing from redirect")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, ok := manager.Get("default", jobID)
		if !ok {
			t.Fatal("scan job unavailable")
		}
		if !job.Running {
			if job.Failed || len(job.Findings) != 1 {
				t.Fatalf("scan result=%#v", job)
			}
			break
		}
		time.Sleep(time.Millisecond)
	}
	job, ok := manager.Get("default", jobID)
	if !ok || job.Running {
		t.Fatal("scan did not finish")
	}
	get := httptest.NewRequest(http.MethodGet, response.Header().Get("Location"), nil).WithContext(auth.WithUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), &auth.User{ID: "admin-1", Role: auth.RoleAdmin, Active: true}))
	page := httptest.NewRecorder()
	handler.linkCheckPage(page, get)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "/missing") || !strings.Contains(page.Body.String(), "Missing page") || !strings.Contains(page.Body.String(), "Home") || !strings.Contains(page.Body.String(), "Suggested action") {
		t.Fatalf("report page status=%d body=%s", page.Code, page.Body.String())
	}
}
