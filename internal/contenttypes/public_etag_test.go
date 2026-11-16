package contenttypes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const publicArticlesYAML = `id: articles
label: Articles
public: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true}
`

func TestPublicAPIETagAndNotModified(t *testing.T) {
	service, types := newServiceFor(t, publicArticlesYAML)
	adminHandler := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})

	// Create article
	r, res := call(t, adminHandler, http.MethodPost, AdminAPIPath+"/articles", `{"title":"Performance First"}`)
	if r.Code != http.StatusCreated {
		t.Fatalf("seed: %d", r.Code)
	}
	dataMap, ok := res["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected res format: %v", res)
	}
	id, _ := dataMap["id"].(string)
	if id == "" {
		t.Fatal("missing record id")
	}

	pubHandler := baasPublicMux(PublicAPI{Service: service, Types: types})

	// 1. Initial GET /api/v1/content/articles/{id}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles/"+id, nil)
	recW := httptest.NewRecorder()
	pubHandler.ServeHTTP(recW, req)

	if recW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recW.Code)
	}
	etag := recW.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header on public GET record")
	}
	cacheCtrl := recW.Header().Get("Cache-Control")
	if !strings.Contains(cacheCtrl, "public") {
		t.Fatalf("expected public Cache-Control header, got %s", cacheCtrl)
	}

	// 2. Subsequent GET with If-None-Match matching ETag -> 304 Not Modified
	req304 := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles/"+id, nil)
	req304.Header.Set("If-None-Match", etag)
	rec304W := httptest.NewRecorder()
	pubHandler.ServeHTTP(rec304W, req304)

	if rec304W.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", rec304W.Code)
	}
	if rec304W.Body.Len() > 0 {
		t.Fatalf("304 response must have empty body, got %s", rec304W.Body.String())
	}

	// 3. Initial GET /api/v1/content/articles (List)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil)
	listW := httptest.NewRecorder()
	pubHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", listW.Code)
	}
	listETag := listW.Header().Get("ETag")
	if listETag == "" {
		t.Fatal("expected ETag header on public GET list")
	}

	// 4. Subsequent GET List with If-None-Match -> 304 Not Modified
	list304Req := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil)
	list304Req.Header.Set("If-None-Match", listETag)
	list304W := httptest.NewRecorder()
	pubHandler.ServeHTTP(list304W, list304Req)

	if list304W.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified for list, got %d", list304W.Code)
	}
}
