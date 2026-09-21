package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyProjectPolicy struct{}

func (denyProjectPolicy) CanRead(context.Context, string) bool   { return false }
func (denyProjectPolicy) CanCreate(context.Context) bool         { return false }
func (denyProjectPolicy) CanUpdate(context.Context, string) bool { return false }
func (denyProjectPolicy) CanDelete(context.Context, string) bool { return false }

func TestProjectHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewProjectHTTPHandler(ProjectService{}, denyProjectPolicy{})
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, ProjectRoute+"/id", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET denied status = %d", get.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, ProjectRoute, nil))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST denied status = %d", post.Code)
	}
}
