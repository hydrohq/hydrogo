package sqlstore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func BenchmarkListRelatedPublishedSQLite(b *testing.B) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(b.TempDir(), "pages-related-benchmark.sqlite"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		b.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	pageStmt, err := tx.PrepareContext(ctx, `INSERT INTO pages (id,title,locale,slug,body,status,revision,published_revision,publication_generation,created_at,updated_at,published_locale,published_slug,published_title,published_body,tenant_id) VALUES (?,?,?,?,?,'published',1,1,1,?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		b.Fatal(err)
	}
	tagStmt, err := tx.PrepareContext(ctx, `INSERT INTO page_published_tags (tenant_id,page_id,tag) VALUES ('',?,'hydro')`)
	if err != nil {
		pageStmt.Close()
		tx.Rollback()
		b.Fatal(err)
	}
	insert := func(id string) error {
		now := "2026-09-28T00:00:00Z"
		if _, err := pageStmt.ExecContext(ctx, id, id, "en", id, "", now, now, "en", id, id, "", ""); err != nil {
			return err
		}
		_, err := tagStmt.ExecContext(ctx, id)
		return err
	}
	if err := insert("source"); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 2000; i++ {
		if err := insert(fmt.Sprintf("related-%04d", i)); err != nil {
			b.Fatal(err)
		}
	}
	pageStmt.Close()
	tagStmt.Close()
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.ListRelatedPublished(ctx, db, "source", "en", 4); err != nil {
			b.Fatal(err)
		}
	}
}
