package portable

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteContentExportImportRoundTripExcludesAuthState(t *testing.T) {
	ctx := context.Background()
	open := func(name string) *sql.DB {
		db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
			db.Close()
			t.Fatal(err)
		}
		return db
	}
	source := open("source.sqlite")
	defer source.Close()
	target := open("target.sqlite")
	defer target.Close()
	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p1','Home',2,'en','home','body','published',2,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('r1','p1',2,'en','home','Home','body','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,active,created_at,updated_at) VALUES ('u1','u@example.test','secret-hash','admin',1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	document, err := Export(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-hash") || strings.Contains(string(data), "password_hash") || strings.Contains(string(data), "sessions") {
		t.Fatalf("export contains auth state: %s", data)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	withUnknownField := strings.TrimSpace(string(data))
	withUnknownField = strings.TrimSuffix(withUnknownField, "}") + `,"unknown":true}`
	if _, err := Unmarshal([]byte(withUnknownField)); err == nil {
		t.Fatal("portable decoder accepted an unknown field")
	}
	if _, err := Unmarshal(append(append([]byte(nil), data...), []byte("{}")...)); err == nil {
		t.Fatal("portable decoder accepted trailing JSON")
	}
	if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error {
		return Import(ctx, tx, decoded, database.SQLite)
	}); err != nil {
		t.Fatal(err)
	}
	if err := Import(ctx, target, decoded, database.Dialect("oracle")); err == nil {
		t.Fatal("portable import accepted unsupported dialect")
	}
	var title string
	if err := target.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = 'p1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Home" {
		t.Fatalf("imported title = %q", title)
	}
	var users int
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatalf("auth users imported: %d", users)
	}
}

func TestUnmarshalReaderEnforcesByteLimit(t *testing.T) {
	if _, err := UnmarshalReader(strings.NewReader("12345"), 4); !errors.Is(err, ErrDocumentTooLarge) {
		t.Fatalf("error = %v, want %v", err, ErrDocumentTooLarge)
	}
	if _, err := UnmarshalReader(strings.NewReader("{}"), 1<<63-1); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("overflowing limit error = %v, want %v", err, ErrInvalidDocument)
	}
}

func TestImportRejectsMissingExecutor(t *testing.T) {
	err := Import(context.Background(), nil, Document{FormatVersion: 1}, database.SQLite)
	if !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("error = %v, want %v", err, database.ErrDatabaseRequired)
	}
}

func TestPortableRejectsTypedNilExecutor(t *testing.T) {
	var db *sql.DB
	if _, err := Export(context.Background(), db); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("typed nil export error = %v, want %v", err, database.ErrDatabaseRequired)
	}
	if err := Import(context.Background(), db, Document{FormatVersion: 1}, database.SQLite); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("typed nil import error = %v, want %v", err, database.ErrDatabaseRequired)
	}
}

func TestPortableAPIsNormalizeNilContext(t *testing.T) {
	db, err := database.Open(nil, database.SQLite, filepath.Join(t.TempDir(), "nil-portable-context.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(nil, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	document, err := Export(nil, db)
	if err != nil {
		t.Fatalf("export with nil context failed: %v", err)
	}
	if err := Import(nil, db, document, database.SQLite); err != nil {
		t.Fatalf("import with nil context failed: %v", err)
	}
}

func TestExportWithLimitRejectsOversizedRowsDuringRead(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "bounded-export.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('large','title',1,'en','large',?,'published',1,1,'2026-01-01','2026-01-01')`, strings.Repeat("x", 1024)); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportWithLimit(ctx, db, 256); !errors.Is(err, ErrDocumentTooLarge) {
		t.Fatalf("bounded export error = %v, want %v", err, ErrDocumentTooLarge)
	}
}

func TestPostgresContentExportImportRoundTrip(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	sourceSchema := "portable_src_" + suffix
	targetSchema := "portable_dst_" + suffix
	for _, schema := range []string{sourceSchema, targetSchema} {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
			t.Fatal(err)
		}
		for _, spec := range specs {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.%s AS TABLE public.%s WITH NO DATA", schema, spec.name, spec.name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP SCHEMA "+sourceSchema+" CASCADE")
		_, _ = db.ExecContext(ctx, "DROP SCHEMA "+targetSchema+" CASCADE")
	}()
	sourceTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, "SET LOCAL search_path TO "+sourceSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('pg-p1','Postgres',2,'en','pg','body','published',2,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('pg-r1','pg-p1',2,'en','pg','Postgres','body','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	document, err := Export(ctx, sourceTx)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourceTx.Rollback(); err != nil {
		t.Fatal(err)
	}
	targetTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targetTx.ExecContext(ctx, "SET LOCAL search_path TO "+targetSchema); err != nil {
		t.Fatal(err)
	}
	if err := Import(ctx, targetTx, document, database.Postgres); err != nil {
		t.Fatal(err)
	}
	if err := targetTx.Commit(); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := db.QueryRowContext(ctx, "SELECT title FROM "+targetSchema+".pages WHERE id = 'pg-p1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Postgres" {
		t.Fatalf("imported PostgreSQL title = %q", title)
	}
}

func TestSQLiteLargeContentFixtureRoundTripPreservesRelations(t *testing.T) {
	ctx := context.Background()
	open := func(name string) *sql.DB {
		db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
			db.Close()
			t.Fatal(err)
		}
		return db
	}
	source := open("large-source.sqlite")
	defer source.Close()
	target := open("large-target.sqlite")
	defer target.Close()
	const records = 256
	for i := 0; i < records; i++ {
		id := fmt.Sprintf("fixture-page-%03d", i)
		slug := fmt.Sprintf("fixture-%03d", i)
		now := fmt.Sprintf("2026-01-%02dT00:00:00Z", (i%28)+1)
		if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,created_at,updated_at,published_locale,published_slug,published_title,published_body) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, fmt.Sprintf("Fixture %03d", i), 2, "tr-tr", slug, "body", "published", 2, 2, int64(i), now, now, "tr-tr", slug, fmt.Sprintf("Fixture %03d", i), "body"); err != nil {
			t.Fatal(err)
		}
		if _, err := source.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES (?,?,?,?,?,?,?,?)`, id+"-r2", id, 2, "tr-tr", slug, fmt.Sprintf("Fixture %03d", i), "body", now); err != nil {
			t.Fatal(err)
		}
		if _, err := source.ExecContext(ctx, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, "fixture-media-"+fmt.Sprint(i), "fixture/"+fmt.Sprint(i), "image.png", "image/png", 64+i, 0, 0, "public", "ready", now, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES (?,?,?,?,?,?)`, "fixture-user", "fixture@example.test", "secret-hash", "admin", "2026-01-01", "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	document, err := Export(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-hash") || strings.Contains(string(encoded), "fixture-user") {
		t.Fatal("large export contains auth state")
	}
	decoded, err := Unmarshal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error {
		return Import(ctx, tx, decoded, database.SQLite)
	}); err != nil {
		t.Fatal(err)
	}
	var pages, revisions, media int
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages").Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_revisions").Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_assets").Scan(&media); err != nil {
		t.Fatal(err)
	}
	if pages != records || revisions != records || media != records {
		t.Fatalf("large round trip counts = pages:%d revisions:%d media:%d", pages, revisions, media)
	}
	var pageID, locale, slug string
	var revision int
	if err := target.QueryRowContext(ctx, "SELECT id, locale, slug, revision FROM pages WHERE id = ?", "fixture-page-127").Scan(&pageID, &locale, &slug, &revision); err != nil {
		t.Fatal(err)
	}
	if pageID != "fixture-page-127" || locale != "tr-tr" || slug != "fixture-127" || revision != 2 {
		t.Fatalf("relation sample = %s/%s/%s/%d", pageID, locale, slug, revision)
	}
	var users int
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatalf("auth users imported: %d", users)
	}
}

func TestPortableExportImportPreservesUTF8AndSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	source, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "utf8-src.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	target, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "utf8-dst.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := database.ApplyMigrations(ctx, target, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	title := "Türkçe Başlık ğüşiöç İĞÜŞÖÇ 🚀 & <script>"
	body := "Paragraf 1\n\nParagraf 2 ile \"özel\" karakterler & semboller: € £ ¥ ⚡"
	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p-utf8',?,1,'tr-tr','turkce-baslik',?,'published',1,1,'2026-01-01','2026-01-01')`, title, body); err != nil {
		t.Fatal(err)
	}

	doc, err := Export(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error {
		return Import(ctx, tx, decoded, database.SQLite)
	}); err != nil {
		t.Fatal(err)
	}

	var gotTitle, gotBody string
	if err := target.QueryRowContext(ctx, "SELECT title, body FROM pages WHERE id = 'p-utf8'").Scan(&gotTitle, &gotBody); err != nil {
		t.Fatal(err)
	}
	if gotTitle != title || gotBody != body {
		t.Fatalf("UTF-8 mismatch:\ngotTitle: %q\nwantTitle: %q\ngotBody: %q\nwantBody: %q", gotTitle, title, gotBody, body)
	}
}

func BenchmarkSQLitePortableRoundTrip256(b *testing.B) {
	ctx := context.Background()
	source, err := database.Open(ctx, database.SQLite, filepath.Join(b.TempDir(), "source.sqlite"))
	if err != nil {
		b.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.SQLite, migrations.FS); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 256; i++ {
		id := fmt.Sprintf("bench-page-%03d", i)
		if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, id, id, 1, "en", id, "body", "published", 1, i, "2026-01-01", "2026-01-01"); err != nil {
			b.Fatal(err)
		}
	}
	target, err := database.Open(ctx, database.SQLite, filepath.Join(b.TempDir(), "target.sqlite"))
	if err != nil {
		b.Fatal(err)
	}
	defer target.Close()
	if err := database.ApplyMigrations(ctx, target, database.SQLite, migrations.FS); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		document, err := Export(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
		encoded, err := Marshal(document)
		if err != nil {
			b.Fatal(err)
		}
		decoded, err := Unmarshal(encoded)
		if err != nil {
			b.Fatal(err)
		}
		if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error { return Import(ctx, tx, decoded, database.SQLite) }); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzUnmarshalPortable(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"format_version":1,"tables":[]}`),
		[]byte(`{"format_version":1,"tables":[{"name":"pages","columns":["id"],"rows":[["p1"]]}]}`),
		[]byte(`{"format_version":2,"tables":[]}`),
		[]byte(`{"malformed": true`),
		[]byte(`{}`),
		[]byte(``),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := Unmarshal(data)
		if err == nil {
			if doc.FormatVersion != 1 {
				t.Fatalf("Unmarshal allowed unsupported format version: %d", doc.FormatVersion)
			}
		}
	})
}

