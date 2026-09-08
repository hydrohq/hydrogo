package contract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestSQLitePagesContentContract(t *testing.T) {
	runPagesContentContract(t, database.SQLite, filepath.Join(t.TempDir(), "pages.sqlite"))
}

func TestPostgresPagesContentContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runPagesContentContract(t, database.Postgres, dsn)
}

func runPagesContentContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	resetContractData(t, db, dialect)

	pagesStore := sqlstore.Store{Dialect: dialect}
	content := service.ContentService{
		Transactions: database.NewTransactor(db),
		Pages:        pagesStore,
		Audit:        auditstore.Store{Dialect: dialect},
		Outbox:       outboxstore.Store{Dialect: dialect},
		PublicCache:  cache.New[pages.Page](4, time.Minute),
	}
	page := pages.Page{ID: "content-page", Locale: "tr-TR", Slug: "/hakkimizda/", Title: "Hakkımızda", Body: "ilk"}
	if err := content.CreateDraft(ctx, page); err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := content.UpdateDraft(ctx, pages.Page{ID: page.ID, Locale: page.Locale, Slug: "hakkimizda", Title: "Hakkımızda güncel", Body: "ikinci"}, 1); err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if err := content.CreateDraft(ctx, pages.Page{ID: "second-draft", Locale: "tr-TR", Slug: "hakkimizda", Title: "İkinci taslak", Body: "taslak"}); err != nil {
		t.Fatalf("same-slug draft should be allowed: %v", err)
	}
	if err := content.Publish(ctx, page.ID, 2); err != nil {
		t.Fatalf("publish: %v", err)
	}

	var status string
	var publishedRevision int
	if err := db.QueryRowContext(ctx, "SELECT status, published_revision FROM pages WHERE id = "+placeholderForTest(dialect, 1), page.ID).Scan(&status, &publishedRevision); err != nil {
		t.Fatal(err)
	}
	if status != string(pages.StatusPublished) || publishedRevision != 2 {
		t.Fatalf("published page = %s/%d, want published/2", status, publishedRevision)
	}
	publicPage, err := content.GetPublished(ctx, db, "TR-tr", "Hakkimizda")
	if err != nil || publicPage.ID != page.ID {
		t.Fatalf("published route lookup = %+v/%v", publicPage, err)
	}
	if publicPage.Title != "Hakkımızda güncel" {
		t.Fatalf("unexpected first published title: %q", publicPage.Title)
	}
	if err := content.UpdateDraft(ctx, pages.Page{ID: page.ID, Locale: page.Locale, Slug: "hakkimizda", Title: "Yeni taslak", Body: "üçüncü"}, 3); err != nil {
		t.Fatalf("edit published page as draft: %v", err)
	}
	publicPage, err = content.GetPublished(ctx, db, "tr-tr", "hakkimizda")
	if err != nil || publicPage.Title != "Hakkımızda güncel" {
		t.Fatalf("draft leaked into public projection: %+v/%v", publicPage, err)
	}
	if err := content.Publish(ctx, page.ID, 4); err != nil {
		t.Fatalf("republish draft: %v", err)
	}
	publicPage, err = content.GetPublished(ctx, db, "tr-tr", "hakkimizda")
	if err != nil || publicPage.Title != "Yeni taslak" {
		t.Fatalf("republished projection = %+v/%v", publicPage, err)
	}
	var revisions int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_revisions WHERE page_id = "+placeholderForTest(dialect, 1), page.ID).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if revisions != 3 {
		t.Fatalf("revision count = %d, want 3", revisions)
	}
}

func placeholderForTest(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return "$1"
	}
	return "?"
}
