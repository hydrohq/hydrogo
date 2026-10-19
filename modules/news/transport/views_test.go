package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type mockViewStore struct {
	mu     sync.Mutex
	counts map[string]int64
}

func (m *mockViewStore) IncrementContentView(ctx context.Context, tenantID, contentType, contentID, locale string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counts == nil {
		m.counts = make(map[string]int64)
	}
	m.counts[contentID]++
	return m.counts[contentID], nil
}

func (m *mockViewStore) ContentViewTotal(ctx context.Context, tenantID, contentType, contentID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[contentID], nil
}

func (m *mockViewStore) ContentViewCounts(ctx context.Context, tenantID, contentType string, contentIDs []string) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[string]int64, len(contentIDs))
	for _, id := range contentIDs {
		res[id] = m.counts[id]
	}
	return res, nil
}

func TestPublicViewEndpoints(t *testing.T) {
	store := &mockViewStore{}
	handler := Handler{
		ContentViews: store,
	}

	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}

	// 1. POST /api/v1/news/views increments view count
	body := `{"slug":"test-article","type":"news"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/news/views", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var postResp struct {
		Data struct {
			Slug  string `json:"slug"`
			Views int64  `json:"views"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&postResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if postResp.Data.Views != 1 || postResp.Data.Slug != "test-article" {
		t.Fatalf("unexpected post resp: %+v", postResp.Data)
	}

	// 2. POST /api/views (legacy/direct alias)
	req2 := httptest.NewRequest(http.MethodPost, "/api/views", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("alias expected status 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	// 3. GET /api/v1/news/views?slug=test-article
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/news/views?slug=test-article", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var getResp struct {
		Data struct {
			Slug  string `json:"slug"`
			Views int64  `json:"views"`
		} `json:"data"`
	}
	if err := json.NewDecoder(getRec.Body).Decode(&getResp); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if getResp.Data.Views != 2 {
		t.Fatalf("expected 2 views, got %d", getResp.Data.Views)
	}

	// 4. GET /api/v1/news/views?slugs=test-article,other-article
	multiReq := httptest.NewRequest(http.MethodGet, "/api/v1/news/views?slugs=test-article,other-article", nil)
	multiRec := httptest.NewRecorder()
	mux.ServeHTTP(multiRec, multiReq)
	if multiRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", multiRec.Code, multiRec.Body.String())
	}
	var multiResp struct {
		Data map[string]int64 `json:"data"`
	}
	if err := json.NewDecoder(multiRec.Body).Decode(&multiResp); err != nil {
		t.Fatalf("failed to decode multi response: %v", err)
	}
	if multiResp.Data["test-article"] != 2 || multiResp.Data["other-article"] != 0 {
		t.Fatalf("unexpected multi resp: %+v", multiResp.Data)
	}
}
