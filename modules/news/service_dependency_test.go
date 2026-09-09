package news

import (
	"context"
	"errors"
	"testing"
)

func TestServiceRejectsMissingRepository(t *testing.T) {
	err := (Service{}).Publish(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrRepositoryRequired) {
		t.Fatalf("expected ErrRepositoryRequired, got %v", err)
	}
}
