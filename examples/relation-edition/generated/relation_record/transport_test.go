package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyRelationRecordPolicy struct{}

func (denyRelationRecordPolicy) CanRead(context.Context, string) bool   { return false }
func (denyRelationRecordPolicy) CanCreate(context.Context) bool         { return false }
func (denyRelationRecordPolicy) CanUpdate(context.Context, string) bool { return false }
func (denyRelationRecordPolicy) CanDelete(context.Context, string) bool { return false }

func TestRelationRecordHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewRelationRecordHTTPHandler(RelationRecordService{}, denyRelationRecordPolicy{})
	handler.ResolveTenant = func(*http.Request) (string, bool) { return "tenant-test", true }
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, RelationRecordRoute+"/id", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET denied status = %d", get.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, RelationRecordRoute, nil))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST denied status = %d", post.Code)
	}
}
