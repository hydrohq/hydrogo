package service

import (
	"context"
	"errors"
	"testing"
)

func TestContentServiceRejectsMissingRepository(t *testing.T) {
	_, err := (ContentService{}).Get(context.Background(), nil, "page-1")
	if !errors.Is(err, ErrDependencyRequired) {
		t.Fatalf("expected ErrDependencyRequired, got %v", err)
	}
}

func TestContentServiceRejectsIncompleteMutationDependencies(t *testing.T) {
	err := (ContentService{}).Publish(context.Background(), "page-1", 1)
	if !errors.Is(err, ErrDependencyRequired) {
		t.Fatalf("expected ErrDependencyRequired, got %v", err)
	}
}

func TestCreateDraftRejectsIncompleteDependencies(t *testing.T) {
	err := (CreateDraft{}).Execute(context.Background(), "page-1", "Title")
	if !errors.Is(err, ErrDependencyRequired) {
		t.Fatalf("expected ErrDependencyRequired, got %v", err)
	}
}
