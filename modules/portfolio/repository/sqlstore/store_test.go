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
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/portfolio"
)

func TestRelatedPublishedRanksSharedTagsAndExcludesCurrentDraftAndOtherTenant(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "portfolio-related.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"portfolio": true}); err != nil {
		t.Fatal(err)
	}
	store := New(database.SQLite)
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	items := []struct {
		ctx  context.Context
		item portfolio.Project
	}{
		{alphaCtx, project("source", "source", "published", "hydro", "go")},
		{alphaCtx, project("best", "best", "published", "hydro", "go")},
		{alphaCtx, project("next", "next", "published", "hydro")},
		{alphaCtx, project("draft", "draft", "draft", "hydro", "go")},
		{betaCtx, project("other-tenant", "other", "published", "hydro", "go")},
	}
	for _, row := range items {
		if err := store.Create(row.ctx, db, row.item); err != nil {
			t.Fatal(err)
		}
	}
	var indexPlan string
	if err := db.QueryRowContext(ctx, `EXPLAIN QUERY PLAN SELECT content_id FROM sectoral_portfolio_tags WHERE tenant_id=? AND tag=?`, "tenant-a", "tag:hydro").Scan(new(int), new(int), new(int), &indexPlan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(indexPlan, "sectoral_portfolio_tags_lookup_idx") {
		t.Fatalf("tag lookup did not use the recommendation index: %s", indexPlan)
	}
	got, err := store.ListRelatedPublished(alphaCtx, db, "source", "en", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "best" || got[1].ID != "next" {
		t.Fatalf("related project order=%v, want best then next", projectIDs(got))
	}
	updated := project("next", "next", "published", "unrelated")
	updated.Industry = "Finance"
	updated.Version = 2
	if err := store.Update(alphaCtx, db, updated); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListRelatedPublished(alphaCtx, db, "source", "en", 4)
	if err != nil || len(got) != 1 || got[0].ID != "best" {
		t.Fatalf("after tag replacement related=%v err=%v", projectIDs(got), err)
	}
	var tenantTag string
	if err := db.QueryRowContext(ctx, `SELECT tenant_id FROM sectoral_portfolio_tags WHERE content_id=? LIMIT 1`, "other-tenant").Scan(&tenantTag); err != nil || tenantTag != "tenant-b" {
		t.Fatalf("tag tenant=%q err=%v", tenantTag, err)
	}
}

func TestMySQLRelatedPublishedUsesSharedTagsAndTenantScope(t *testing.T) {
	ctx := context.Background()
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	store := New(database.MySQL)
	alpha, _ := tenant.NewScope("mysql-alpha", "alpha")
	beta, _ := tenant.NewScope("mysql-beta", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	for _, row := range []struct {
		ctx  context.Context
		item portfolio.Project
	}{
		{alphaCtx, project("mysql-source", "source", "published", "hydro", "go")},
		{alphaCtx, project("mysql-best", "best", "published", "hydro", "go")},
		{alphaCtx, project("mysql-next", "next", "published", "hydro")},
		{betaCtx, project("mysql-other", "other", "published", "hydro", "go")},
	} {
		if err := store.Create(row.ctx, db, row.item); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ListRelatedPublished(alphaCtx, db, "mysql-source", "en", 4)
	if err != nil || len(got) != 2 || got[0].ID != "mysql-best" || got[1].ID != "mysql-next" {
		t.Fatalf("related project order=%v err=%v", projectIDs(got), err)
	}
}

func TestPostgresRelatedRepositoryContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "portfolio_related_" + strconv.FormatInt(time.Now().UnixNano(), 10)
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
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, map[string]bool{"portfolio": true}); err != nil {
		t.Fatal(err)
	}
	store := New(database.Postgres)
	transactor := database.NewTenantTransactor(db, database.Postgres, true)
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	alphaCtx := tenant.WithScope(ctx, alpha)
	for _, item := range []portfolio.Project{project("source", "source", "published", "hydro"), project("candidate", "candidate", "published", "hydro")} {
		if err := transactor.Within(alphaCtx, func(tx database.Executor) error { return store.Create(alphaCtx, tx, item) }); err != nil {
			t.Fatal(err)
		}
	}
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		queryer, ok := tx.(database.QueryExecutor)
		if !ok {
			return errors.New("transaction does not support row queries")
		}
		got, err := store.ListRelatedPublished(alphaCtx, queryer, "source", "en", 4)
		if err != nil {
			return err
		}
		if len(got) != 1 || got[0].ID != "candidate" {
			return errors.New("PostgreSQL related content contract failed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func project(id, slug, status string, tags ...string) portfolio.Project {
	return portfolio.Project{Base: sectoral.Base{ID: id, Locale: "en", Slug: slug, Title: "Project " + id, Status: status, Tags: tags}, Industry: "Technology"}
}

func projectIDs(items []portfolio.Project) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
