package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyContentBlockPolicy struct{}

func (denyContentBlockPolicy) CanRead(context.Context, string) bool   { return false }
func (denyContentBlockPolicy) CanCreate(context.Context) bool         { return false }
func (denyContentBlockPolicy) CanUpdate(context.Context, string) bool { return false }
func (denyContentBlockPolicy) CanDelete(context.Context, string) bool { return false }

func TestContentBlockHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewContentBlockHTTPHandler(ContentBlockService{}, denyContentBlockPolicy{})
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, ContentBlockRoute+"/id", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET denied status = %d", get.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, ContentBlockRoute, nil))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST denied status = %d", post.Code)
	}
}
