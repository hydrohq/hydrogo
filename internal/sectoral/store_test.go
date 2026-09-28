package sectoral

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type testItem struct {
	Base
	Detail string `json:"detail"`
}

type auditCall struct{ id, action, target string }
type auditRecorder struct{ calls []auditCall }

func (a *auditRecorder) Record(_ context.Context, _ database.Executor, id, action, target string) error {
	a.calls = append(a.calls, auditCall{id, action, target})
	return nil
}

func TestAuditEventsHaveDeterministicIdempotencyKeys(t *testing.T) {
	recorder := &auditRecorder{}
	for i := 0; i < 2; i++ {
		if err := RecordAudit(context.Background(), nil, recorder, "events", "created", "event-1", 1); err != nil {
			t.Fatal(err)
		}
	}
	if len(recorder.calls) != 2 || recorder.calls[0].id != recorder.calls[1].id || recorder.calls[0].action != "events.created" {
		t.Fatalf("audit calls=%+v", recorder.calls)
	}
	if err := RecordAudit(context.Background(), nil, nil, "events", "created", "event-1", 1); err != ErrAuditRequired {
		t.Fatalf("missing audit writer=%v", err)
	}
}

func TestSQLiteRepositoryCRUDOrderingAndTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "sectoral.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	enabled := map[string]bool{"locations": true}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatal(err)
	}
	for _, table := range []struct {
		name   string
		exists bool
	}{{"sectoral_locations", true}, {"sectoral_team", false}, {"sectoral_legal", false}} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table.name).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if (count == 1) != table.exists {
			t.Fatalf("inactive schema %s exists=%v", table.name, count == 1)
		}
	}
	store := Store[testItem]{Dialect: database.SQLite, Table: "sectoral_locations"}
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	item := testItem{Base: Base{ID: "location-1", Locale: "EN", Slug: "branch", Title: "Central", Status: "published", Featured: true, SortOrder: 1, Tags: []string{"Store", "store"}}, Detail: "Address"}
	if err := store.Create(alphaCtx, db, item); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(alphaCtx, db, testItem{Base: Base{ID: "location-3", Locale: "en", Slug: "branch-two", Title: "Second", Status: "published", SortOrder: 0}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(betaCtx, db, testItem{Base: Base{ID: "location-2", Locale: "en", Slug: "branch", Title: "Other", Status: "published"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(betaCtx, db, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read: %v", err)
	}
	got, err := store.Get(alphaCtx, db, item.ID)
	if err != nil || got.Detail != "Address" {
		t.Fatalf("Get()=%+v err=%v", got, err)
	}
	stale := got
	got.Title = "Renamed"
	got.Version++
	if err := store.Update(alphaCtx, db, got); err != nil {
		t.Fatal(err)
	}
	stale.Version = got.Version
	if err := store.Update(alphaCtx, db, stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	listed, err := store.ListPublished(alphaCtx, db, "en", 50)
	if err != nil || len(listed) != 2 || listed[0].Title != "Renamed" || !listed[0].Featured {
		t.Fatalf("ListPublished()=%+v err=%v", listed, err)
	}
	if err := store.Create(alphaCtx, db, testItem{Base: Base{ID: "duplicate", Locale: "en", Slug: "branch", Title: "Duplicate"}}); err == nil {
		t.Fatal("duplicate locale/slug accepted")
	}
	if err := store.Delete(alphaCtx, db, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(alphaCtx, db, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted item read: %v", err)
	}
}

func TestMySQLRepositoryCRUDOrderingAndTenantScope(t *testing.T) {
	ctx := context.Background()
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	store := Store[testItem]{Dialect: database.MySQL, Table: "sectoral_locations"}
	alpha, _ := tenant.NewScope("mysql-tenant-a", "alpha")
	beta, _ := tenant.NewScope("mysql-tenant-b", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	item := testItem{Base: Base{ID: "mysql-location-1", Locale: "EN", Slug: "branch", Title: "Central", Status: "published", Featured: true, SortOrder: 1, Tags: []string{"Store", "store"}}, Detail: "Address"}
	if err := store.Create(alphaCtx, db, item); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(betaCtx, db, testItem{Base: Base{ID: "mysql-location-2", Locale: "en", Slug: "branch", Title: "Other", Status: "published"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(betaCtx, db, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read: %v", err)
	}
	got, err := store.Get(alphaCtx, db, item.ID)
	if err != nil || got.Detail != "Address" {
		t.Fatalf("Get()=%+v err=%v", got, err)
	}
	got.Title, got.Version = "Renamed", got.Version+1
	if err := store.Update(alphaCtx, db, got); err != nil {
		t.Fatal(err)
	}
	listed, err := store.ListPublished(alphaCtx, db, "en", 50)
	if err != nil || len(listed) != 1 || listed[0].Title != "Renamed" || !listed[0].Featured {
		t.Fatalf("ListPublished()=%+v err=%v", listed, err)
	}
	if err := store.Delete(alphaCtx, db, item.ID); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRejectsUnsafeTableAndUnsupportedDialect(t *testing.T) {
	ctx := context.Background()
	if err := (Store[testItem]{Dialect: database.SQLite, Table: "records;DROP TABLE users"}).Create(ctx, nil, testItem{}); err == nil {
		t.Fatal("unsafe table name accepted")
	}
	if err := (Store[testItem]{Dialect: database.Dialect("oracle"), Table: "safe_table"}).Create(ctx, nil, testItem{}); err == nil {
		t.Fatal("unsupported dialect accepted")
	}
}

func TestPostgresTenantRepositoryContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "sectoral_" + strconv.FormatInt(time.Now().UnixNano(), 10)
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
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, map[string]bool{"locations": true}); err != nil {
		t.Fatal(err)
	}
	var rls, force bool
	if err := db.QueryRowContext(ctx, "SELECT relrowsecurity,relforcerowsecurity FROM pg_class WHERE oid='sectoral_locations'::regclass").Scan(&rls, &force); err != nil || !rls || !force {
		t.Fatalf("sectoral RLS enabled=%v forced=%v err=%v", rls, force, err)
	}
	store := Store[testItem]{Dialect: database.Postgres, Table: "sectoral_locations"}
	transactor := database.NewTenantTransactor(db, database.Postgres, true)
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	alphaCtx, betaCtx := tenant.WithScope(ctx, alpha), tenant.WithScope(ctx, beta)
	if err := transactor.Within(alphaCtx, func(tx database.Executor) error {
		return store.Create(alphaCtx, tx, testItem{Base: Base{ID: "pg-entry", Locale: "en", Slug: "branch", Title: "Alpha", Status: "published"}})
	}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ctx  context.Context
		want int
	}{{alphaCtx, 1}, {betaCtx, 0}} {
		var count int
		err := transactor.Within(tc.ctx, func(tx database.Executor) error {
			queryer, ok := tx.(database.QueryExecutor)
			if !ok {
				return errors.New("transaction does not support queries")
			}
			items, err := store.ListPublished(tc.ctx, queryer, "en", 20)
			count = len(items)
			return err
		})
		if err != nil || count != tc.want {
			t.Fatalf("tenant list count=%d want=%d err=%v", count, tc.want, err)
		}
	}
}
