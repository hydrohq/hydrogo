package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
)

func TestPostgresDraftPageUsesStableBoundedLocaleScopedPagination(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	store, err := openPostgresEditorialStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close isolated PostgreSQL fixture schema: %v", err)
		}
	})
	if err := store.Reset(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixtures := make([]adminstudio.DraftDocument, 26)
	for i := range fixtures {
		fixtures[i] = adminstudio.DraftDocument{
			ID: fmt.Sprintf("zz-paged-%02d", i+1), Version: 1, State: adminstudio.StateDraft, Exists: true,
			Fields: map[string]any{"title": fmt.Sprintf("Paged draft %02d", i+1)},
		}
	}
	if err := store.SeedDraftPageFixtures(fixtures); err != nil {
		t.Fatal(err)
	}

	first, err := store.DraftPage("en", 0, 26)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 26 {
		t.Fatalf("first page/lookahead length = %d, want 26", len(first))
	}
	if first[0].ID != "zz-paged-26" || first[24].ID != "zz-paged-02" || first[25].ID != "zz-paged-01" {
		t.Fatalf("first page/lookahead order = len %d, first %q, 25th %q, lookahead %q", len(first), first[0].ID, first[24].ID, first[25].ID)
	}
	second, err := store.DraftPage("en", 25, 26)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) == 0 || second[0].ID != "zz-paged-01" {
		t.Fatalf("second page did not begin with previous page lookahead: %#v", second)
	}
	french, err := store.DraftPage("fr", 0, 26)
	if err != nil {
		t.Fatal(err)
	}
	if len(french) != 0 {
		t.Fatalf("locale filter returned drafts outside fr: %#v", french)
	}
}
