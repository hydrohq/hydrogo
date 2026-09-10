package graphql

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTPAcceptsReadOnlyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/graphql", strings.NewReader(`{"query":"{ products { id } }"}`))
	res := httptest.NewRecorder()
	(Handler{}).ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"data"`) {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}

func TestServeHTTPRejectsMutationSchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/graphql", strings.NewReader(`{"query":"mutation { deletePage(id: \"x\") }"}`))
	res := httptest.NewRecorder()
	(Handler{}).ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "Schema is not configured for mutations") {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}
