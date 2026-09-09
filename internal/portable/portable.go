// Package portable defines the allowlisted, secret-free content export format.
package portable

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidDocument = errors.New("invalid portable document")
var ErrDocumentTooLarge = errors.New("portable document exceeds size limit")

const DefaultMaxDocumentBytes int64 = 256 << 20

type Document struct {
	FormatVersion int     `json:"format_version"`
	Tables        []Table `json:"tables"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type tableSpec struct {
	name    string
	columns []string
	orderBy string
}

var specs = []tableSpec{
	{name: "pages", columns: []string{"id", "title", "version", "locale", "slug", "body", "status", "revision", "published_revision", "publication_generation", "created_at", "updated_at", "published_locale", "published_slug", "published_title", "published_body"}, orderBy: "id"},
	{name: "page_revisions", columns: []string{"id", "page_id", "revision", "locale", "slug", "title", "body", "created_at"}, orderBy: "id"},
	{name: "media_assets", columns: []string{"id", "storage_key", "original_filename", "mime_type", "size_bytes", "width", "height", "visibility", "status", "created_at", "updated_at"}, orderBy: "id"},
	{name: "site_settings", columns: []string{"key", "value", "is_public", "updated_at"}, orderBy: "key"},
	{name: "navigation_items", columns: []string{"id", "locale", "label", "href", "position", "is_visible", "updated_at"}, orderBy: "id"},
	{name: "page_redirects", columns: []string{"source_path", "target_path", "status", "created_at"}, orderBy: "source_path"},
	{name: "form_inbox", columns: []string{"id", "form_key", "email", "payload", "source_ip_hash", "status", "created_at"}, orderBy: "id"},
	{name: "news_items", columns: []string{"id", "locale", "slug", "title", "body", "status", "version", "revision", "published_revision", "schedule_generation", "scheduled_at", "created_at", "updated_at"}, orderBy: "id"},
}

var specByName = func() map[string]tableSpec {
	result := make(map[string]tableSpec, len(specs))
	for _, spec := range specs {
		result[spec.name] = spec
	}
	return result
}()

func Export(ctx context.Context, db database.Executor) (Document, error) {
	return ExportWithLimit(ctx, db, DefaultMaxDocumentBytes)
}

// ExportWithLimit bounds the in-memory document while rows are read. The
// returned Document format is still materialized for callers, but an
// untrusted or unexpectedly large database cannot grow it without limit.
func ExportWithLimit(ctx context.Context, db database.Executor, maxBytes int64) (Document, error) {
	if maxBytes <= 0 {
		return Document{}, ErrDocumentTooLarge
	}
	if err := database.RequireExecutor(db); err != nil {
		return Document{}, err
	}
	document := Document{FormatVersion: 1, Tables: make([]Table, 0, len(specs))}
	for _, spec := range specs {
		query := "SELECT " + strings.Join(spec.columns, ", ") + " FROM " + spec.name + " ORDER BY " + spec.orderBy
		rows, err := queryRows(ctx, db, query, len(spec.columns), maxBytes)
		if err != nil {
			return Document{}, fmt.Errorf("export %s: %w", spec.name, err)
		}
		document.Tables = append(document.Tables, Table{Name: spec.name, Columns: append([]string(nil), spec.columns...), Rows: rows})
		encoded, err := json.Marshal(document)
		if err != nil {
			return Document{}, fmt.Errorf("size export %s: %w", spec.name, err)
		}
		if int64(len(encoded)) > maxBytes {
			return Document{}, ErrDocumentTooLarge
		}
	}
	return document, nil
}

func Marshal(document Document) ([]byte, error) {
	if err := validate(document); err != nil {
		return nil, err
	}
	return json.MarshalIndent(document, "", "  ")
}

func Unmarshal(data []byte) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Document{}, ErrInvalidDocument
		}
		return Document{}, err
	}
	if err := validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

// UnmarshalReader bounds bytes read from an operator-provided export before
// decoding it, preventing an untrusted path from causing an unbounded read.
func UnmarshalReader(reader io.Reader, maxBytes int64) (Document, error) {
	if reader == nil || maxBytes <= 0 {
		return Document{}, ErrInvalidDocument
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return Document{}, err
	}
	if int64(len(data)) > maxBytes {
		return Document{}, ErrDocumentTooLarge
	}
	return Unmarshal(data)
}

// Import replaces only the allowlisted content tables. Authentication,
// session, audit, outbox and job state are intentionally never imported.
func Import(ctx context.Context, tx database.Executor, document Document, dialect database.Dialect) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if err := validate(document); err != nil {
		return err
	}
	if dialect != database.SQLite && dialect != database.Postgres {
		return fmt.Errorf("unsupported portable import dialect %q", dialect)
	}
	for _, table := range []string{"page_revisions", "pages", "media_assets", "site_settings", "navigation_items", "page_redirects", "form_inbox", "news_items"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	tablesByName := make(map[string]Table, len(document.Tables))
	for _, table := range document.Tables {
		tablesByName[table.Name] = table
	}
	// Parent rows must be inserted before dependent revisions.
	for _, tableName := range []string{"pages", "page_revisions", "media_assets", "site_settings", "navigation_items", "page_redirects", "form_inbox", "news_items"} {
		table, ok := tablesByName[tableName]
		if !ok {
			continue
		}
		if len(table.Rows) == 0 {
			continue
		}
		placeholders := make([]string, len(table.Columns))
		for i := range placeholders {
			if dialect == database.Postgres {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			} else {
				placeholders[i] = "?"
			}
		}
		query := "INSERT INTO " + table.Name + " (" + strings.Join(table.Columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
		for _, row := range table.Rows {
			values := make([]any, len(row))
			for i, value := range row {
				values[i] = normalizeValue(value)
			}
			if _, err := tx.ExecContext(ctx, query, values...); err != nil {
				return fmt.Errorf("import %s: %w", table.Name, err)
			}
		}
	}
	return nil
}

func queryRows(ctx context.Context, db database.Executor, query string, columnCount int, maxBytes int64) ([][]any, error) {
	queryer, ok := db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("export requires a query-capable executor")
	}
	rows, err := queryer.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([][]any, 0)
	var encodedRows int64
	for rows.Next() {
		values := make([]any, columnCount)
		pointers := make([]any, columnCount)
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return nil, err
		}
		encodedRows += int64(len(encoded)) + 1
		if encodedRows > maxBytes {
			return nil, ErrDocumentTooLarge
		}
		result = append(result, values)
	}
	return result, rows.Err()
}

func validate(document Document) error {
	if document.FormatVersion != 1 {
		return ErrInvalidDocument
	}
	seen := make(map[string]struct{}, len(document.Tables))
	for _, table := range document.Tables {
		spec, ok := specByName[table.Name]
		if !ok || len(table.Columns) != len(spec.columns) || !sameStrings(table.Columns, spec.columns) {
			return ErrInvalidDocument
		}
		if _, exists := seen[table.Name]; exists {
			return ErrInvalidDocument
		}
		seen[table.Name] = struct{}{}
		for _, row := range table.Rows {
			if len(row) != len(table.Columns) {
				return ErrInvalidDocument
			}
		}
	}
	if len(seen) != len(specs) {
		return ErrInvalidDocument
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func normalizeValue(value any) any {
	if number, ok := value.(json.Number); ok {
		if integer, err := number.Int64(); err == nil {
			return integer
		}
		return number.String()
	}
	return value
}

func init() {
	sort.Slice(specs, func(i, j int) bool { return specs[i].name < specs[j].name })
}
