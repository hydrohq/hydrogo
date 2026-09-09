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
