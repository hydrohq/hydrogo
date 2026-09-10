package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestParseBlockFormBuildsStructuredBlocks(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("block_count=1&block_type_0=hero&block_title_0=Welcome&block_button_url_0=%2Fstart"))
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	blocks, err := parseBlockForm(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].Type != "hero" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

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
