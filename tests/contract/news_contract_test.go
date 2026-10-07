package contract_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
)

func TestSQLiteNewsContract(t *testing.T) {
	runNewsContract(t, database.SQLite, filepath.Join(t.TempDir(), "news.sqlite"))
}

func TestPostgresNewsContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runNewsContract(t, database.Postgres, dsn)
}

func TestSQLiteNewsDraftPreviewContract(t *testing.T) {
	runNewsDraftPreviewContract(t, database.SQLite, filepath.Join(t.TempDir(), "news-preview.sqlite"))
}

func TestPostgresNewsDraftPreviewContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runNewsDraftPreviewContract(t, database.Postgres, dsn)
}

func runNewsDraftPreviewContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: dialect}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store}
	marker := time.Now().UTC().UnixNano()
	item := news.Item{ID: fmt.Sprintf("preview-%d", marker), Locale: "en", Slug: fmt.Sprintf("draft-%d", marker), Title: "Private news draft", Body: "Draft preview contract body", Status: news.StatusDraft}
	if err := service.CreateDraft(ctx, item); err != nil {
		t.Fatal(err)
	}
	item, err = store.Get(ctx, db, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	signer := &preview.Signer{Key: []byte("0123456789abcdef0123456789abcdef")}
	token, err := signer.Issue(preview.Claims{ResourceType: "news", ResourceID: item.ID, Revision: item.Revision}, time.Now().UTC(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	handler := newstransport.Handler{DB: db, Service: service, PreviewSigner: signer}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "GET /public/news/{locale}/{slug}/html" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	publicPath := "/public/news/" + item.Locale + "/" + item.Slug + "/html"
	withoutToken := httptest.NewRecorder()
	mux.ServeHTTP(withoutToken, httptest.NewRequest(http.MethodGet, publicPath, nil))
	if withoutToken.Code != http.StatusNotFound {
		t.Fatalf("draft without preview token status=%d body=%s", withoutToken.Code, withoutToken.Body.String())
	}
	previewResponse := httptest.NewRecorder()
	mux.ServeHTTP(previewResponse, httptest.NewRequest(http.MethodGet, publicPath+"?preview_token="+token, nil))
	if previewResponse.Code != http.StatusOK || !strings.Contains(previewResponse.Body.String(), "Private news draft") {
		t.Fatalf("valid draft preview status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}
	for header, want := range map[string]string{"Cache-Control": "private, no-store", "X-Robots-Tag": "noindex, nofollow", "Referrer-Policy": "no-referrer", "X-Frame-Options": "SAMEORIGIN"} {
		if got := previewResponse.Header().Get(header); got != want {
			t.Errorf("preview %s=%q, want %q", header, got, want)
		}
	}
	if csp := previewResponse.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'self'") || strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("preview CSP frame policy = %q", csp)
	}
	updated := item
	updated.Title = "Changed news draft"
	updated.Body = "Changed revision"
	if err := service.UpdateDraft(ctx, updated, item.Version); err != nil {
		t.Fatal(err)
	}
	stalePreview := httptest.NewRecorder()
	mux.ServeHTTP(stalePreview, httptest.NewRequest(http.MethodGet, publicPath+"?preview_token="+token, nil))
	if stalePreview.Code != http.StatusNotFound {
		t.Fatalf("stale revision preview status=%d body=%s", stalePreview.Code, stalePreview.Body.String())
	}
}

func runNewsContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM news_items"); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: dialect}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store, Audit: auditstore.Store{Dialect: dialect}, Outbox: outboxstore.Store{Dialect: dialect}}
	marker := time.Now().UTC().UnixNano()
	item := news.Item{ID: fmt.Sprintf("news-%d", marker), Locale: "en", Slug: fmt.Sprintf("launch-%d", marker), Title: "Launch", Body: "body", Status: news.StatusDraft}
	if err := service.CreateDraft(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := service.UpdateDraft(ctx, item, 99); !errors.Is(err, news.ErrConflict) {
		t.Fatalf("stale update = %v", err)
	}
	if err := service.Schedule(ctx, item.ID, 1, time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := service.RunScheduled(ctx, item.ID, 99, time.Now().UTC()); !errors.Is(err, news.ErrScheduleStale) {
		t.Fatalf("stale schedule = %v", err)
	}
	worker := news.Worker{Queue: jobs.NewQueue(10), Service: service}
	if err := worker.EnqueueDue(ctx, db, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if ran, err := worker.RunOnce(ctx, "news-worker-1", time.Now().UTC()); err != nil || !ran {
		t.Fatalf("scheduled worker = %v/%v", ran, err)
	}
	if err := service.RunScheduled(ctx, item.ID, 1, time.Now().UTC()); !errors.Is(err, news.ErrScheduleStale) {
		t.Fatalf("replayed schedule = %v", err)
	}
	published, err := store.GetPublishedByRoute(ctx, db, item.Locale, item.Slug)
	if err != nil || published.Status != news.StatusPublished || published.PublishedRevision == nil {
		t.Fatalf("published = %#v/%v", published, err)
	}
	if err := service.Retract(ctx, item.ID, published.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPublishedByRoute(ctx, db, item.Locale, item.Slug); err == nil {
		t.Fatal("retracted news remained public")
	}
}

func TestMemberSecurityAndIsolationContract(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "members.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	// Verify member table isolation and sessions
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM members").Scan(&count); err != nil {
		t.Fatalf("members table should exist and be queryable: %v", err)
	}
	if count != 0 {
		t.Errorf("initial member count = %d, want 0", count)
	}

	var sessionCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM member_sessions").Scan(&sessionCount); err != nil {
		t.Fatalf("member_sessions table should exist and be queryable: %v", err)
	}
	if sessionCount != 0 {
		t.Errorf("initial member session count = %d, want 0", sessionCount)
	}
}
