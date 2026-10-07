package generated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyMilestonePolicy struct{}

func (denyMilestonePolicy) CanRead(context.Context, string) bool   { return false }
func (denyMilestonePolicy) CanCreate(context.Context) bool         { return false }
func (denyMilestonePolicy) CanUpdate(context.Context, string) bool { return false }
func (denyMilestonePolicy) CanDelete(context.Context, string) bool { return false }

func TestMilestoneHTTPHandlerDeniesWithoutPolicyPermission(t *testing.T) {
	handler := NewMilestoneHTTPHandler(MilestoneService{}, denyMilestonePolicy{})
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, MilestoneRoute+"/id", nil))
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET denied status = %d", get.Code)
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, MilestoneRoute, nil))
	if post.Code != http.StatusForbidden {
		t.Fatalf("POST denied status = %d", post.Code)
	}
}
