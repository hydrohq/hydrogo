package sqlstore

import (
	"context"
	"testing"
)

func TestListRejectsUnboundedLimitsBeforeDatabaseAccess(t *testing.T) {
	store := Store{}
	for _, limit := range []int{0, -1, 101, 1000} {
		if _, err := store.ListPublished(context.Background(), nil, "tr", limit); err != ErrInvalidListLimit {
			t.Fatalf("published limit %d error = %v", limit, err)
		}
		if _, err := store.ListAdmin(context.Background(), nil, limit); err != ErrInvalidListLimit {
			t.Fatalf("admin limit %d error = %v", limit, err)
		}
		if _, err := store.ListPublishedTranslations(context.Background(), nil, "slug", limit); err != ErrInvalidListLimit {
			t.Fatalf("translation limit %d error = %v", limit, err)
		}
	}
}
