package transport

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func TestWriteErrorMapsMissingRepositoryToUnavailable(t *testing.T) {
	if code := newsErrorStatus(news.ErrRepositoryRequired); code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", code, http.StatusServiceUnavailable)
	}
	if errors.Is(news.ErrRepositoryRequired, news.ErrInvalidItem) {
		t.Fatal("repository error unexpectedly classified as validation")
	}
	if code := newsErrorStatus(database.ErrDatabaseRequired); code != http.StatusServiceUnavailable {
		t.Fatalf("database status = %d, want %d", code, http.StatusServiceUnavailable)
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
