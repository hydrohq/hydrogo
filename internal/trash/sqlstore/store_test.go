package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
)

func TestTrashListMoveRestoreAndRetentionAreTenantScoped(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "trash.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, ddl := range []string{
		`CREATE TABLE pages(id TEXT PRIMARY KEY,title TEXT,deleted_at TEXT,created_at TEXT,tenant_id TEXT)`,
		`CREATE TABLE news_items(id TEXT PRIMARY KEY,title TEXT,deleted_at TEXT,created_at TEXT,tenant_id TEXT)`,
		`CREATE TABLE form_inbox(id TEXT PRIMARY KEY,form_key TEXT,deleted_at TEXT,created_at TEXT,tenant_id TEXT)`,
		`CREATE TABLE media_assets(id TEXT PRIMARY KEY,original_filename TEXT,deleted_at TEXT,created_at TEXT,tenant_id TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	stamp := now.Add(-time.Hour).Format(time.RFC3339Nano)
	expired := now.Add(-trash.Retention - time.Hour).Format(time.RFC3339Nano)
	for _, insert := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO pages VALUES(?,?,?,?,?)`, []any{"page-a", "About", nil, stamp, "alpha"}},
		{`INSERT INTO pages VALUES(?,?,?,?,?)`, []any{"page-trash", "Trashed", stamp, stamp, "alpha"}},
		{`INSERT INTO pages VALUES(?,?,?,?,?)`, []any{"page-expired", "Expired", expired, expired, "alpha"}},
		{`INSERT INTO news_items VALUES(?,?,?,?,?)`, []any{"news-b", "Other tenant", stamp, stamp, "beta"}},
		{`INSERT INTO form_inbox VALUES(?,?,?,?,?)`, []any{"form-a", "contact", nil, stamp, "alpha"}},
	} {
		if _, err := db.Exec(insert.query, insert.args...); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true, Enabled: map[trash.Kind]bool{trash.Pages: true, trash.News: true, trash.Submissions: true}}
	alpha, _ := tenant.NewScope("alpha", "alpha")
	ctx = tenant.WithScope(ctx, alpha)
	service := trash.Service{Repository: store}
	items, err := service.List(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("trash entries: %+v", items)
	}
	seen := map[string]trash.Entry{}
	for _, item := range items {
		seen[string(item.Kind)+":"+item.ID] = item
	}
	if item := seen["pages:page-trash"]; !item.Trashed || item.Label != "Trashed" {
		t.Fatalf("trashed page: %+v", item)
	}
	if item := seen["pages:page-a"]; item.Trashed {
		t.Fatalf("active page: %+v", item)
	}
	if item := seen["submissions:form-a"]; item.Trashed || item.Label != "Form submission" {
		t.Fatalf("active form: %+v", item)
	}
	if err := service.Move(ctx, trash.Pages, "page-a", now); err != nil {
		t.Fatal(err)
	}
	if err := service.Restore(ctx, trash.Pages, "page-a", now); err != nil {
		t.Fatalf("restore recent page: %v", err)
	}
	if err := service.Restore(ctx, trash.Pages, "page-expired", now); err != trash.ErrNotFound {
		t.Fatalf("expired restore error = %v", err)
	}
	if err := service.Move(ctx, trash.Pages, "page-trash", now); err != trash.ErrNotFound {
		t.Fatalf("duplicate move error = %v", err)
	}
	if err := service.Restore(ctx, trash.News, "news-b", now); err != trash.ErrNotFound {
		t.Fatalf("cross-tenant restore error = %v", err)
	}
	if err := service.Move(ctx, trash.Kind("users"), "some-id", now); err != trash.ErrNotFound {
		t.Fatalf("unrecognized kind error = %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE test_audit (id TEXT PRIMARY KEY, action TEXT, target TEXT)`); err != nil {
		t.Fatal(err)
	}
	service.Transactions = database.NewTenantTransactor(db, database.SQLite, true)
	service.Audit = recordingAudit{}
	if err := service.Move(ctx, trash.Pages, "page-a", now); err != nil {
		t.Fatal(err)
	}
	var deletedAt string
	var auditCount int
	if err := db.QueryRow(`SELECT deleted_at FROM pages WHERE id='page-a'`).Scan(&deletedAt); err != nil || deletedAt == "" {
		t.Fatalf("transactional move deleted_at=%q err=%v", deletedAt, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM test_audit`).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}
	service.Audit = recordingAudit{err: errors.New("audit unavailable")}
	if err := service.Restore(ctx, trash.Pages, "page-a", now); err == nil {
		t.Fatal("restore succeeded after audit failure")
	}
	if err := db.QueryRow(`SELECT deleted_at FROM pages WHERE id='page-a'`).Scan(&deletedAt); err != nil || deletedAt == "" {
		t.Fatalf("failed audited restore was not rolled back: deleted_at=%q err=%v", deletedAt, err)
	}
}

type recordingAudit struct{ err error }

var _ audit.Writer = recordingAudit{}

func (a recordingAudit) Record(_ context.Context, tx database.Executor, id, action, target string) error {
	if a.err != nil {
		return a.err
	}
	_, err := tx.ExecContext(context.Background(), `INSERT INTO test_audit VALUES(?,?,?)`, id, action, target)
	return err
}
