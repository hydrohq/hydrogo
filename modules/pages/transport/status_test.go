package transport

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestPageStatusMapsMissingDependenciesToUnavailable(t *testing.T) {
	if got := pageStatus(service.ErrDependencyRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("write status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := pageReadStatus(service.ErrDependencyRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("read status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := pageReadStatus(errors.New("not found")); got != http.StatusNotFound {
		t.Fatalf("not-found status = %d, want %d", got, http.StatusNotFound)
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
