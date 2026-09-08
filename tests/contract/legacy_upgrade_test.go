package contract_test

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
)

//go:embed testdata/legacy_customer.json
var legacyCustomerData []byte

type legacyCustomerFixture struct {
	Page struct {
		ID        string `json:"id"`
		Locale    string `json:"locale"`
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Status    string `json:"status"`
		Revision  int    `json:"revision"`
		Version   int    `json:"version"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"page"`
	Revision struct {
		ID        string `json:"id"`
		Revision  int    `json:"revision"`
		CreatedAt string `json:"created_at"`
	} `json:"revision"`
	Media struct {
		ID               string `json:"id"`
		StorageKey       string `json:"storage_key"`
		OriginalFilename string `json:"original_filename"`
		MimeType         string `json:"mime_type"`
		SizeBytes        int64  `json:"size_bytes"`
		Visibility       string `json:"visibility"`
		Status           string `json:"status"`
		CreatedAt        string `json:"created_at"`
		UpdatedAt        string `json:"updated_at"`
	} `json:"media"`
	User struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		PasswordHash string `json:"password_hash"`
		Role         string `json:"role"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	} `json:"user"`
}

func TestSQLiteLegacyCustomerUpgradePreservesPublishedContentAndMedia(t *testing.T) {
	var fixture legacyCustomerFixture
	if err := json.Unmarshal(legacyCustomerData, &fixture); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "legacy.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, database.SQLite, legacyMigrationFS(t)); err != nil {
		t.Fatalf("legacy migrations: %v", err)
	}
	seedLegacyCustomer(t, ctx, db, fixture)
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("upgrade migrations: %v", err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("compatibility after upgrade: %v", err)
	}

	store := pagesstore.Store{Dialect: database.SQLite}
	page, err := store.GetPublishedByRoute(ctx, db, "tr-tr", "anasayfa")
	if err != nil {
		t.Fatalf("legacy published route: %v", err)
	}
	if page.ID != fixture.Page.ID || page.Title != fixture.Page.Title || page.Body != fixture.Page.Body {
		t.Fatalf("published legacy page = %+v", page)
	}
	var storageKey, status string
	if err := db.QueryRowContext(ctx, "SELECT storage_key, status FROM media_assets WHERE id = ?", fixture.Media.ID).Scan(&storageKey, &status); err != nil {
		t.Fatal(err)
	}
	if storageKey != fixture.Media.StorageKey || status != "ready" {
		t.Fatalf("legacy media = %s/%s", storageKey, status)
	}
	var hash string
	if err := db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", fixture.User.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash != fixture.User.PasswordHash {
		t.Fatalf("legacy auth hash changed: %q", hash)
	}
}

func legacyMigrationFS(t *testing.T) fs.FS {
	t.Helper()
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	files := make(fstest.MapFS)
	for i, entry := range entries {
		if i >= 8 || !entry.IsDir() {
			continue
		}
		for _, name := range []string{"sqlite.sql", "postgres.sql"} {
			path := fmt.Sprintf("%s/%s", entry.Name(), name)
			content, err := fs.ReadFile(migrations.FS, path)
			if err != nil {
				t.Fatal(err)
			}
			files[path] = &fstest.MapFile{Data: content}
		}
	}
	return files
}

func seedLegacyCustomer(t *testing.T, ctx context.Context, db *sql.DB, fixture legacyCustomerFixture) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{"INSERT INTO pages (id,title,version,locale,slug,body,status,revision,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)", []any{fixture.Page.ID, fixture.Page.Title, fixture.Page.Version, fixture.Page.Locale, fixture.Page.Slug, fixture.Page.Body, fixture.Page.Status, fixture.Page.Revision, fixture.Page.CreatedAt, fixture.Page.UpdatedAt}},
		{"INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES (?,?,?,?,?,?,?,?)", []any{fixture.Revision.ID, fixture.Page.ID, fixture.Revision.Revision, fixture.Page.Locale, fixture.Page.Slug, fixture.Page.Title, fixture.Page.Body, fixture.Revision.CreatedAt}},
		{"INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,visibility,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)", []any{fixture.Media.ID, fixture.Media.StorageKey, fixture.Media.OriginalFilename, fixture.Media.MimeType, fixture.Media.SizeBytes, fixture.Media.Visibility, fixture.Media.Status, fixture.Media.CreatedAt, fixture.Media.UpdatedAt}},
		{"INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES (?,?,?,?,?,?)", []any{fixture.User.ID, fixture.User.Email, fixture.User.PasswordHash, fixture.User.Role, fixture.User.CreatedAt, fixture.User.UpdatedAt}},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed legacy customer: %v", err)
		}
	}
}
