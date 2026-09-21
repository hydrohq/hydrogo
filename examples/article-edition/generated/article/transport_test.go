package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyArticlePolicy struct{}
func (denyArticlePolicy) CanRead(context.Context, string) bool { return false }
func (denyArticlePolicy) CanCreate(context.Context) bool { return false }
func (denyArticlePolicy) CanUpdate(context.Context, string) bool { return false }
func (denyArticlePolicy) CanDelete(context.Context, string) bool { return false }

func TestArticleHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewArticleHTTPHandler(ArticleService{}, denyArticlePolicy{})
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, ArticleRoute+"/id", nil))
	if get.Code != http.StatusNotFound { t.Fatalf("GET denied status = %d", get.Code) }
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, ArticleRoute, nil))
	if post.Code != http.StatusForbidden { t.Fatalf("POST denied status = %d", post.Code) }
}
