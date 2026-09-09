package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
}
