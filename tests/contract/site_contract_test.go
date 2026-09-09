package contract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestSQLiteSiteEditionContract(t *testing.T) {
	runSiteEditionContract(t, database.SQLite, filepath.Join(t.TempDir(), "site.sqlite"))
}

func TestPostgresSiteEditionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runSiteEditionContract(t, database.Postgres, dsn)
}

func runSiteEditionContract(t *testing.T, dialect database.Dialect, dsn string) {
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
	resetContractData(t, db, dialect)
	store := sitestore.Store{Dialect: dialect}
	transactor := database.NewTransactor(db)
	service := site.Service{Store: store, Transactions: transactor, Audit: auditstore.Store{Dialect: dialect}}
	if err := service.SaveSetting(ctx, site.Setting{Key: "site.name", Value: "HydroGo", Public: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.SaveSetting(ctx, site.Setting{Key: "site.name", Value: "HydroGo CMS", Public: true}); err != nil {
		t.Fatalf("repeated setting update: %v", err)
	}
	if err := service.SaveNavigation(ctx, "EN", []site.NavigationItem{{ID: "home", Label: "Home", Href: "/", Position: 0, Visible: true}}); err != nil {
		t.Fatal(err)
	}
	if err := transactor.Within(ctx, func(tx database.Executor) error {
		return store.ReplaceNavigation(ctx, tx, "en", []site.NavigationItem{
			{ID: "one", Locale: "en", Label: "One", Href: "/one", Position: 1, Visible: true},
			{ID: "two", Locale: "en", Label: "Two", Href: "/two", Position: 1, Visible: true},
		})
	}); err == nil {
		t.Fatal("duplicate navigation position was accepted")
	}
	if err := service.SaveRedirect(ctx, site.Redirect{SourcePath: "/old/", TargetPath: "/new/", Status: 301}); err != nil {
		t.Fatal(err)
	}
	if err := service.Submit(ctx, site.Submission{ID: "form-1", FormKey: "contact", Email: "user@example.com", Payload: `{"message":"hello"}`, SourceIPHash: "hash"}); err != nil {
		t.Fatal(err)
	}
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil || len(settings) != 1 || settings[0].Value != "HydroGo CMS" {
		t.Fatalf("settings = %#v/%v", settings, err)
	}
	navigation, err := store.ListNavigation(ctx, db, "en")
	if err != nil || len(navigation) != 1 || navigation[0].Href != "/" {
		t.Fatalf("navigation = %#v/%v", navigation, err)
	}
	redirect, err := store.GetRedirect(ctx, db, "/old")
	if err != nil || redirect.TargetPath != "/new" || redirect.Status != 301 {
		t.Fatalf("redirect = %#v/%v", redirect, err)
	}
	submissions, err := store.ListSubmissions(ctx, db, "contact", 10)
	if err != nil || len(submissions) != 1 || submissions[0].ID != "form-1" {
		t.Fatalf("submissions = %#v/%v", submissions, err)
	}
	if err := service.SetSubmissionStatus(ctx, "form-1", "resolved"); err != nil {
		t.Fatalf("set submission status: %v", err)
	}
	if purged, err := service.PurgeInbox(ctx, nowPlusHour()); err != nil || purged != 1 {
		t.Fatalf("purge submissions = %d/%v", purged, err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events").Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount < 6 {
		t.Fatalf("site mutations recorded %d audit events, want at least 6", auditCount)
	}
}

func nowPlusHour() time.Time { return time.Now().UTC().Add(time.Hour) }
