package transport

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func TestSiteStatusSeparatesUnavailableStore(t *testing.T) {
	if got := siteStatus(site.ErrStoreRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("store status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := siteStatus(errors.New("invalid input")); got != http.StatusBadRequest {
		t.Fatalf("validation status = %d, want %d", got, http.StatusBadRequest)
	}
}
