package media

import (
	"net/http"
	"testing"
)

func TestMediaStatusMapsInfrastructureErrorsToUnavailable(t *testing.T) {
	if got := mediaStatus(ErrAssetRepositoryRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("repository status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := mediaStatus(ErrUploaderRequired); got != http.StatusServiceUnavailable {
		t.Fatalf("uploader status = %d, want %d", got, http.StatusServiceUnavailable)
	}
}
