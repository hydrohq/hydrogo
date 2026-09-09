package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func TestWriteErrorMapsMissingRepositoryToUnavailable(t *testing.T) {
	res := httptest.NewRecorder()
	writeError(res, news.ErrRepositoryRequired)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
	if errors.Is(news.ErrRepositoryRequired, news.ErrInvalidItem) {
		t.Fatal("repository error unexpectedly classified as validation")
	}
	res = httptest.NewRecorder()
	writeError(res, database.ErrDatabaseRequired)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("database status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestNewIDProducesUniqueNonEmptyIDs(t *testing.T) {
	seen := make(map[string]struct{}, 1024)
	for i := 0; i < 1024; i++ {
		id, err := newID()
		if err != nil || id == "" {
			t.Fatalf("newID() = %q, %v", id, err)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate generated ID %q", id)
		}
		seen[id] = struct{}{}
	}
}
