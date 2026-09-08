package erasure

import (
	"context"
	"testing"
)

func TestGCSkipsPinnedKeysAndDeletesOnlyUnpinned(t *testing.T) {
	deleted := []string{}
	gc := GC{
		Pinned: func(_ context.Context, key string) (bool, error) { return key == "pinned", nil },
		Delete: func(_ context.Context, key string) error { deleted = append(deleted, key); return nil },
	}
	removed, err := gc.Collect(context.Background(), []string{"pinned", "free"})
	if err != nil || len(removed) != 1 || removed[0] != "free" || len(deleted) != 1 || deleted[0] != "free" {
		t.Fatalf("gc result = %#v/%v deleted=%#v", removed, err, deleted)
	}
}

func TestGCStopsOnPinProviderFailure(t *testing.T) {
	gc := GC{Pinned: func(context.Context, string) (bool, error) { return false, context.Canceled }, Delete: func(context.Context, string) error { t.Fatal("delete after pin failure"); return nil }}
	if _, err := gc.Collect(context.Background(), []string{"key"}); err != context.Canceled {
		t.Fatalf("gc error = %v", err)
	}
}
