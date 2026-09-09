package site

import (
	"context"
	"errors"
	"testing"
)

func TestServiceRejectsMissingStore(t *testing.T) {
	err := (Service{}).SaveSetting(context.Background(), Setting{Key: "site.title"})
	if !errors.Is(err, ErrStoreRequired) {
		t.Fatalf("expected ErrStoreRequired, got %v", err)
	}
}
