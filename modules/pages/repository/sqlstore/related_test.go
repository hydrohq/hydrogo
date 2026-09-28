package sqlstore

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestRelatedPublishedRanksSharedTagsAndExcludesDraftsAndOtherTenants(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "page-related.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	transactor := database.NewTransactor(db)
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	for _, row := range []struct {
		ctx       context.Context
		page      pages.Page
		published bool
	}{
		{alphaCtx, pages.Page{ID: "source", Locale: "en", Slug: "source", Title: "Source", Tags: []string{"hydro", "go"}}, true},
		{alphaCtx, pages.Page{ID: "best", Locale: "en", Slug: "best", Title: "Best", Tags: []string{"hydro", "go"}}, true},
		{alphaCtx, pages.Page{ID: "next", Locale: "en", Slug: "next", Title: "Next", Tags: []string{"hydro"}}, true},
		{alphaCtx, pages.Page{ID: "draft", Locale: "en", Slug: "draft", Title: "Draft", Tags: []string{"hydro", "go"}}, false},
		{betaCtx, pages.Page{ID: "other", Locale: "en", Slug: "other", Title: "Other tenant", Tags: []string{"hydro", "go"}}, true},
	} {
		if err := transactor.Within(row.ctx, func(tx database.Executor) error {
			if err := store.CreateDraftContent(row.ctx, tx, row.page); err != nil {
				return err
			}
			if row.published {
				return store.Publish(row.ctx, tx, row.page.ID, 1)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	planRows, err := db.QueryContext(ctx, `EXPLAIN QUERY PLAN SELECT p.id FROM page_published_tags source CROSS JOIN page_published_tags candidate JOIN pages p ON p.id=candidate.page_id AND p.tenant_id=candidate.tenant_id WHERE candidate.tenant_id=source.tenant_id AND candidate.tag=source.tag AND source.page_id=? AND source.tenant_id=? AND p.tenant_id=? AND p.published_locale=? AND p.status='published' AND p.published_revision IS NOT NULL AND p.id<>? GROUP BY p.id ORDER BY COUNT(source.tag) DESC,p.updated_at DESC,p.id ASC LIMIT ?`, "source", "tenant-a", "tenant-a", "en", "source", 4)
	if err != nil {
		t.Fatal(err)
	}
	var plan strings.Builder
	for planRows.Next() {
		var id, parent, notused int
		var detail string
		if err := planRows.Scan(&id, &parent, &notused, &detail); err != nil {
			planRows.Close()
			t.Fatal(err)
		}
		plan.WriteString(detail)
	}
	if err := planRows.Err(); err != nil {
		planRows.Close()
		t.Fatal(err)
	}
	planRows.Close()
	if !strings.Contains(plan.String(), "page_published_tags") {
		t.Fatalf("recommendation query did not use the published tag index: %s", plan.String())
	}
	var got []pages.Page
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		var err error
		got, err = store.ListRelatedPublished(alphaCtx, tx, "source", "en", 4)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "best" || got[1].ID != "next" {
		t.Fatalf("related page order=%v", pageIDs(got))
	}
	updated := pages.Page{ID: "next", Locale: "en", Slug: "next", Title: "Next", Tags: []string{"unrelated"}}
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error { return store.UpdateDraftContent(alphaCtx, tx, updated, 2) }); err != nil {
		t.Fatal(err)
	}
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		var err error
		got, err = store.ListRelatedPublished(alphaCtx, tx, "source", "en", 4)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "best" {
		t.Fatalf("draft edit leaked into public suggestions: %v", pageIDs(got))
	}
	var page pages.Page
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		var err error
		page, err = store.GetContent(alphaCtx, tx, "source")
		return err
	}); err != nil || len(page.Tags) != 2 {
		t.Fatalf("current tags=%v err=%v", page.Tags, err)
	}
	var public pages.Page
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		var err error
		public, err = store.GetPublishedByRoute(alphaCtx, tx, "en", "source")
		return err
	}); err != nil || len(public.Tags) != 2 {
		t.Fatalf("published tags=%v err=%v", public.Tags, err)
	}
}

func pageIDs(items []pages.Page) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestPostgresRelatedPageContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "page_related_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	defer func() { _, _ = admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); _ = admin.Close() }()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.Postgres, RequireTenantScope: true}
	transactor := database.NewTenantTransactor(db, database.Postgres, true)
	scope, _ := tenant.NewScope("tenant-a", "alpha")
	requestCtx := tenant.WithScope(ctx, scope)
	for _, page := range []pages.Page{
		{ID: "source", Locale: "en", Slug: "source", Title: "Source", Tags: []string{"hydro"}},
		{ID: "related", Locale: "en", Slug: "related", Title: "Related", Tags: []string{"hydro"}},
	} {
		if err := transactor.Within(requestCtx, func(tx database.Executor) error {
			if err := store.CreateDraftContent(requestCtx, tx, page); err != nil {
				return err
			}
			return store.Publish(requestCtx, tx, page.ID, 1)
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := transactor.Within(requestCtx, func(tx database.Executor) error {
		items, err := store.ListRelatedPublished(requestCtx, tx, "source", "en", 4)
		if err != nil {
			return err
		}
		if len(items) != 1 || items[0].ID != "related" {
			return errors.New("PostgreSQL related-page contract failed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
