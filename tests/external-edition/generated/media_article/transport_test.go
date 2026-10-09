package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyMediaArticlePolicy struct{}

func (denyMediaArticlePolicy) CanRead(context.Context, string) bool   { return false }
func (denyMediaArticlePolicy) CanCreate(context.Context) bool         { return false }
func (denyMediaArticlePolicy) CanUpdate(context.Context, string) bool { return false }
func (denyMediaArticlePolicy) CanDelete(context.Context, string) bool { return false }

func TestMediaArticleHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewMediaArticleHTTPHandler(MediaArticleService{}, denyMediaArticlePolicy{})
	handler.ResolveTenant = func(*http.Request) (string, bool) { return "tenant-test", true }
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, MediaArticleRoute+"/id", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET denied status = %d", get.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, MediaArticleRoute, nil))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST denied status = %d", post.Code)
	}
}
