package search

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryProviderSearchRebuildAndPagination(t *testing.T) {
	provider := NewMemoryProvider(3)
	ctx := context.Background()
	if err := provider.Rebuild(ctx, []Document{
		{ID: "a", Locale: "tr", Title: "Go ile Haber", Body: "İçerik platformu"},
		{ID: "b", Locale: "tr", Title: "HydroGo Haber", Body: "Go performansı"},
		{ID: "c", Locale: "en", Title: "Go News", Body: "English"},
	}); err != nil {
		t.Fatal(err)
	}
	response, err := provider.Search(ctx, Query{Text: "go", Locale: "tr", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if response.Total != 2 || len(response.Items) != 1 || response.Items[0].ID != "a" {
		t.Fatalf("response = %#v", response)
	}
	if err := provider.Delete(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	response, err = provider.Search(ctx, Query{Text: "go", Locale: "tr"})
	if err != nil || len(response.Items) != 1 || response.Items[0].ID != "a" {
		t.Fatalf("after delete = %#v err=%v", response, err)
	}
}

func TestMemoryProviderBoundsAndCancellation(t *testing.T) {
	provider := NewMemoryProvider(1)
	ctx := context.Background()
	if err := provider.Upsert(ctx, Document{ID: "one", Locale: "en", Title: "One"}); err != nil {
		t.Fatal(err)
	}
	if err := provider.Upsert(ctx, Document{ID: "two", Locale: "en", Title: "Two"}); !errors.Is(err, ErrMemoryIndexFull) {
		t.Fatalf("full error = %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := provider.Search(cancelled, Query{Text: "one"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("search cancellation = %v", err)
	}
}
