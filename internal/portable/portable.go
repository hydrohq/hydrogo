// Package portable defines the allowlisted, secret-free content export format.
package portable

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidDocument = errors.New("invalid portable document")

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
	document := Document{FormatVersion: 1, Tables: make([]Table, 0, len(specs))}
	for _, spec := range specs {
		query := "SELECT " + strings.Join(spec.columns, ", ") + " FROM " + spec.name + " ORDER BY " + spec.orderBy
		rows, err := queryRows(ctx, db, query, len(spec.columns))
		if err != nil {
			return Document{}, fmt.Errorf("export %s: %w", spec.name, err)
		}
		document.Tables = append(document.Tables, Table{Name: spec.name, Columns: append([]string(nil), spec.columns...), Rows: rows})
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
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, err
	}
	if err := validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

// Import replaces only the allowlisted content tables. Authentication,
// session, audit, outbox and job state are intentionally never imported.
func Import(ctx context.Context, tx database.Executor, document Document, dialect database.Dialect) error {
	if err := validate(document); err != nil {
		return err
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

func queryRows(ctx context.Context, db database.Executor, query string, columnCount int) ([][]any, error) {
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
