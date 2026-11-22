package portable

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/migrations"
)

type recordedExec struct {
	query string
	args  []any
}

type recordingExecutor struct {
	execs []recordedExec
}

func (r *recordingExecutor) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	r.execs = append(r.execs, recordedExec{query: query, args: append([]any(nil), args...)})
	return nil, nil
}

func (r *recordingExecutor) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}

func (r *recordingExecutor) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func TestV1ImportSupportsMySQLDialect(t *testing.T) {
	ctx := context.Background()
	source, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p1','Home',1,'en','home','body','published',1,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	doc, err := Export(ctx, source)
	if err != nil {
		t.Fatal(err)
	}

	rec := &recordingExecutor{}
	if err := Import(ctx, rec, doc, database.MySQL); err != nil {
		t.Fatalf("Import(MySQL) failed: %v", err)
	}

	var sawPagesInsert bool
	for _, call := range rec.execs {
		if strings.HasPrefix(call.query, "INSERT INTO pages ") {
			sawPagesInsert = true
			if strings.Contains(call.query, "$1") {
				t.Fatalf("MySQL insert used Postgres placeholder: %s", call.query)
			}
			if !strings.Contains(call.query, "VALUES (?") {
				t.Fatalf("MySQL insert missing ? placeholders: %s", call.query)
			}
		}
	}
	if !sawPagesInsert {
		t.Fatal("expected INSERT INTO pages for MySQL import")
	}
}

func TestNDJSONStreamRoundTripSQLiteAndCustomBackend(t *testing.T) {
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

	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p1','Stream Title',2,'en','stream','hello','published',2,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,active,created_at,updated_at) VALUES ('u1','u@example.test','secret-hash','admin',1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	summary, err := ExportNDJSON(ctx, NewSQLRecordSource(source, database.SQLite), &buf, ScopeContent, DefaultMaxDocumentBytes)
	if err != nil {
		t.Fatalf("ExportNDJSON failed: %v", err)
	}
	if summary.Collections != len(specs) || summary.Records < 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if strings.Contains(buf.String(), "secret-hash") || strings.Contains(buf.String(), "password_hash") {
		t.Fatalf("NDJSON export leaked auth state: %s", buf.String())
	}

	// Dry-run validates without mutating target.
	if _, err := ImportNDJSON(ctx, nil, bytes.NewReader(buf.Bytes()), DefaultMaxDocumentBytes, true); err != nil {
		t.Fatalf("ImportNDJSON dry-run failed: %v", err)
	}

	if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error {
		_, err := ImportNDJSON(ctx, NewSQLRecordSink(tx, database.SQLite), bytes.NewReader(buf.Bytes()), DefaultMaxDocumentBytes, false)
		return err
	}); err != nil {
		t.Fatalf("ImportNDJSON failed: %v", err)
	}

	var gotTitle string
	if err := target.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = 'p1'").Scan(&gotTitle); err != nil {
		t.Fatal(err)
	}
	if gotTitle != "Stream Title" {
		t.Fatalf("imported title = %q, want %q", gotTitle, "Stream Title")
	}

	// Verify custom non-SQL sink (e.g. future MongoDB backend) receives the same records cleanly.
	memSink := &memoryRecordSink{dialect: "mongodb"}
	if _, err := ImportNDJSON(ctx, memSink, bytes.NewReader(buf.Bytes()), DefaultMaxDocumentBytes, false); err != nil {
		t.Fatalf("ImportNDJSON into custom sink failed: %v", err)
	}
	if !memSink.prepared || len(memSink.records["pages"]) != 1 {
		t.Fatalf("custom sink did not receive expected pages record: prepared=%v count=%d", memSink.prepared, len(memSink.records["pages"]))
	}
}

func TestNDJSONScopeAllIncludesExtraNonSecretTablesAndExcludesAuth(t *testing.T) {
	ctx := context.Background()
	source, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `CREATE TABLE ct_articles (id TEXT PRIMARY KEY, title TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO ct_articles (id, title) VALUES ('a1', 'Dynamic Content')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,active,created_at,updated_at) VALUES ('u1','u@example.test','secret-hash','admin',1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := ExportNDJSON(ctx, NewSQLRecordSource(source, database.SQLite), &buf, ScopeAll, DefaultMaxDocumentBytes); err != nil {
		t.Fatalf("ExportNDJSON(ScopeAll) failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"ct_articles"`) || !strings.Contains(out, `"Dynamic Content"`) {
		t.Fatalf("ScopeAll did not export dynamic content table ct_articles: %s", out)
	}
	if strings.Contains(out, "secret-hash") || strings.Contains(out, `"users"`) {
		t.Fatalf("ScopeAll leaked auth table or secret: %s", out)
	}
}

func TestNDJSONRejectsDenylistedAndMalformedStreams(t *testing.T) {
	ctx := context.Background()
	badStreams := []string{
		`{"type":"header","format_version":2,"scope":"content","source_dialect":"sqlite","collections":[{"name":"users","columns":["id"]}]}`,
		`{"type":"header","format_version":99,"scope":"content","source_dialect":"sqlite","collections":[]}`,
		"{\"type\":\"header\",\"format_version\":2,\"scope\":\"content\",\"source_dialect\":\"sqlite\",\"collections\":[{\"name\":\"pages\",\"columns\":[\"id\"]}]}\n{\"type\":\"record\",\"collection\":\"pages\",\"values\":[\"p1\",\"extra\"]}",
		`{"type":"header","format_version":2,"scope":"content","source_dialect":"sqlite","collections":[],"unknown":true}`,
	}
	for i, raw := range badStreams {
		if _, err := ImportNDJSON(ctx, nil, strings.NewReader(raw), DefaultMaxDocumentBytes, true); !errors.Is(err, ErrInvalidDocument) {
			t.Fatalf("case %d: err = %v, want ErrInvalidDocument", i, err)
		}
	}
	if _, err := ImportNDJSON(ctx, nil, strings.NewReader("1234567890"), 5, true); !errors.Is(err, ErrDocumentTooLarge) {
		t.Fatalf("size limit err = %v, want ErrDocumentTooLarge", err)
	}
}

type memoryRecordSink struct {
	dialect  string
	prepared bool
	records  map[string][][]any
}

func (m *memoryRecordSink) Dialect() string { return m.dialect }
func (m *memoryRecordSink) Prepare(_ context.Context, cols []CollectionSchema) error {
	m.prepared = true
	m.records = make(map[string][][]any, len(cols))
	return nil
}
func (m *memoryRecordSink) WriteRecord(_ context.Context, col CollectionSchema, values []any) error {
	m.records[col.Name] = append(m.records[col.Name], values)
	return nil
}
