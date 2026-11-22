package portable

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/hydrohq/hydrogo/internal/database"
)

// Scope selects which collection set is included in a streaming export.
type Scope string

const (
	ScopeContent Scope = "content"
	ScopeAll     Scope = "all"
)

const StreamFormatVersion = 2

// CollectionSchema describes one exported table or document collection.
type CollectionSchema struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// StreamHeader is the first NDJSON line of a v2 portable stream.
type StreamHeader struct {
	Type          string             `json:"type"`
	FormatVersion int                `json:"format_version"`
	Scope         Scope              `json:"scope"`
	SourceDialect string             `json:"source_dialect"`
	Collections   []CollectionSchema `json:"collections"`
}

// StreamRecord is a single data row/document line in a v2 portable stream.
type StreamRecord struct {
	Type       string `json:"type"`
	Collection string `json:"collection"`
	Values     []any  `json:"values"`
}

// StreamSummary reports counts processed by ExportNDJSON or ImportNDJSON.
type StreamSummary struct {
	Collections int `json:"collections"`
	Records     int `json:"records"`
}

// RecordSource abstracts reading collections and rows from any storage engine
// (SQLite, PostgreSQL, MySQL, or future NoSQL backends such as MongoDB).
type RecordSource interface {
	Dialect() string
	Collections(ctx context.Context, scope Scope) ([]CollectionSchema, error)
	StreamRecords(ctx context.Context, collection CollectionSchema, fn func(values []any) error) error
}

// RecordSink abstracts preparing collections and writing rows to any storage engine.
type RecordSink interface {
	Dialect() string
	Prepare(ctx context.Context, collections []CollectionSchema) error
	WriteRecord(ctx context.Context, collection CollectionSchema, values []any) error
}

var identPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// denylistedTables are never exported or imported, even in ScopeAll, so auth,
// session, audit, job, and internal control-plane state never leak across DBs.
var denylistedTables = map[string]bool{
	"users":             true,
	"sessions":          true,
	"recovery_tokens":   true,
	"auth_invites":      true,
	"mfa_challenges":    true,
	"audit_logs":        true,
	"audit_chain":       true,
	"jobs":              true,
	"outbox_events":     true,
	"schema_migrations": true,
	"tenant_snapshots":  true,
	"tenants":           true,
	"sqlite_sequence":   true,
}

var denylistedColumns = map[string]bool{
	"password_hash": true,
	"totp_secret":   true,
	"token_hash":    true,
	"secret_hash":   true,
	"api_key_hash":  true,
}

// ExportNDJSON writes a bounded, streaming v2 NDJSON export from src to w.
func ExportNDJSON(ctx context.Context, src RecordSource, w io.Writer, scope Scope, maxBytes int64) (StreamSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if src == nil || w == nil {
		return StreamSummary{}, ErrInvalidDocument
	}
	if maxBytes <= 0 {
		return StreamSummary{}, ErrDocumentTooLarge
	}
	if scope != ScopeContent && scope != ScopeAll {
		return StreamSummary{}, fmt.Errorf("%w: unsupported scope %q", ErrInvalidDocument, scope)
	}

	cols, err := src.Collections(ctx, scope)
	if err != nil {
		return StreamSummary{}, fmt.Errorf("discover collections: %w", err)
	}
	for _, c := range cols {
		if !isSafeCollection(c.Name) {
			return StreamSummary{}, fmt.Errorf("%w: forbidden collection %q", ErrInvalidDocument, c.Name)
		}
	}

	header := StreamHeader{
		Type:          "header",
		FormatVersion: StreamFormatVersion,
		Scope:         scope,
		SourceDialect: src.Dialect(),
		Collections:   cols,
	}
	var written int64
	writeLine := func(v any) error {
		line, err := json.Marshal(v)
		if err != nil {
			return err
		}
		line = append(line, '\n')
		written += int64(len(line))
		if written > maxBytes {
			return ErrDocumentTooLarge
		}
		_, err = w.Write(line)
		return err
	}

	if err := writeLine(header); err != nil {
		return StreamSummary{}, err
	}

	var totalRecords int
	for _, col := range cols {
		err := src.StreamRecords(ctx, col, func(values []any) error {
			if len(values) != len(col.Columns) {
				return fmt.Errorf("%w: column count mismatch for %s", ErrInvalidDocument, col.Name)
			}
			totalRecords++
			return writeLine(StreamRecord{
				Type:       "record",
				Collection: col.Name,
				Values:     values,
			})
		})
		if err != nil {
			return StreamSummary{}, err
		}
	}
	return StreamSummary{Collections: len(cols), Records: totalRecords}, nil
}

// ImportNDJSON validates and streams a v2 NDJSON export into sink.
// When dryRun is true, sink is not touched and may be nil.
func ImportNDJSON(ctx context.Context, sink RecordSink, r io.Reader, maxBytes int64, dryRun bool) (StreamSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || maxBytes <= 0 || maxBytes == 1<<63-1 {
		return StreamSummary{}, ErrInvalidDocument
	}
	if !dryRun && sink == nil {
		return StreamSummary{}, ErrInvalidDocument
	}

	limited := &countingReader{r: io.LimitReader(r, maxBytes+1), max: maxBytes}
	scanner := bufio.NewScanner(limited)
	// Allow up to 16 MiB per individual JSONL record.
	scanner.Buffer(make([]byte, 64*1024), 16<<20)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return StreamSummary{}, err
		}
		if limited.exceeded {
			return StreamSummary{}, ErrDocumentTooLarge
		}
		return StreamSummary{}, ErrInvalidDocument
	}
	if limited.exceeded {
		return StreamSummary{}, ErrDocumentTooLarge
	}

	var header StreamHeader
	if err := decodeStrictLine(scanner.Bytes(), &header); err != nil {
		return StreamSummary{}, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
	}
	if header.Type != "header" || header.FormatVersion != StreamFormatVersion {
		return StreamSummary{}, ErrInvalidDocument
	}
	if header.Scope != ScopeContent && header.Scope != ScopeAll {
		return StreamSummary{}, ErrInvalidDocument
	}

	colByName := make(map[string]CollectionSchema, len(header.Collections))
	for _, col := range header.Collections {
		if !isSafeCollection(col.Name) || len(col.Columns) == 0 {
			return StreamSummary{}, ErrInvalidDocument
		}
		for _, c := range col.Columns {
			if !identPattern.MatchString(c) || denylistedColumns[strings.ToLower(c)] {
				return StreamSummary{}, ErrInvalidDocument
			}
		}
		if _, dup := colByName[col.Name]; dup {
			return StreamSummary{}, ErrInvalidDocument
		}
		colByName[col.Name] = col
	}

	if !dryRun {
		if err := sink.Prepare(ctx, header.Collections); err != nil {
			return StreamSummary{}, fmt.Errorf("prepare sink: %w", err)
		}
	}

	var records int
	for scanner.Scan() {
		if limited.exceeded {
			return StreamSummary{}, ErrDocumentTooLarge
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec StreamRecord
		if err := decodeStrictLine(line, &rec); err != nil {
			return StreamSummary{}, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
		}
		if rec.Type != "record" {
			return StreamSummary{}, ErrInvalidDocument
		}
		col, ok := colByName[rec.Collection]
		if !ok || len(rec.Values) != len(col.Columns) {
			return StreamSummary{}, ErrInvalidDocument
		}
		records++
		if !dryRun {
			normalized := make([]any, len(rec.Values))
			for i, v := range rec.Values {
				normalized[i] = normalizeValue(v)
			}
			if err := sink.WriteRecord(ctx, col, normalized); err != nil {
				return StreamSummary{}, fmt.Errorf("write %s: %w", col.Name, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return StreamSummary{}, err
	}
	if limited.exceeded {
		return StreamSummary{}, ErrDocumentTooLarge
	}
	return StreamSummary{Collections: len(header.Collections), Records: records}, nil
}

type countingReader struct {
	r        io.Reader
	read     int64
	max      int64
	exceeded bool
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += int64(n)
	if c.read > c.max {
		c.exceeded = true
	}
	return n, err
}

func decodeStrictLine(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return ErrInvalidDocument
	}
	return nil
}

func isSafeCollection(name string) bool {
	if !identPattern.MatchString(name) {
		return false
	}
	lower := strings.ToLower(name)
	if lower == "hydro_search_fts" || strings.HasPrefix(lower, "hydro_search_fts_") {
		return false
	}
	return !denylistedTables[lower]
}

// sqlRecordSource implements RecordSource for SQLite, PostgreSQL, and MySQL.
type sqlRecordSource struct {
	db      database.Executor
	dialect database.Dialect
}

// NewSQLRecordSource creates a RecordSource backed by SQLite, PostgreSQL, or MySQL.
func NewSQLRecordSource(db database.Executor, dialect database.Dialect) RecordSource {
	return &sqlRecordSource{db: db, dialect: dialect}
}

func (s *sqlRecordSource) Dialect() string { return string(s.dialect) }

func (s *sqlRecordSource) Collections(ctx context.Context, scope Scope) ([]CollectionSchema, error) {
	if err := database.RequireExecutor(s.db); err != nil {
		return nil, err
	}
	if s.dialect != database.SQLite && s.dialect != database.Postgres && s.dialect != database.MySQL {
		return nil, fmt.Errorf("unsupported portable export dialect %q", s.dialect)
	}

	// Start with canonical content specs in parent-first order.
	orderedContent := []string{
		"pages", "page_revisions", "media_assets", "site_settings",
		"navigation_items", "page_redirects", "form_inbox", "news_items",
	}
	cols := make([]CollectionSchema, 0, len(orderedContent))
	seen := make(map[string]bool, len(orderedContent))
	for _, name := range orderedContent {
		spec := specByName[name]
		cols = append(cols, CollectionSchema{
			Name:    spec.name,
			Columns: append([]string(nil), spec.columns...),
		})
		seen[spec.name] = true
	}
	if scope == ScopeContent {
		return cols, nil
	}

	extraTables, err := s.discoverExtraTables(ctx, seen)
	if err != nil {
		return nil, err
	}
	for _, tbl := range extraTables {
		tblCols, err := s.discoverColumns(ctx, tbl)
		if err != nil {
			return nil, err
		}
		if len(tblCols) > 0 {
			cols = append(cols, CollectionSchema{Name: tbl, Columns: tblCols})
		}
	}
	return cols, nil
}

func (s *sqlRecordSource) StreamRecords(ctx context.Context, col CollectionSchema, fn func(values []any) error) error {
	if !isSafeCollection(col.Name) || len(col.Columns) == 0 {
		return ErrInvalidDocument
	}
	for _, c := range col.Columns {
		if !identPattern.MatchString(c) || denylistedColumns[strings.ToLower(c)] {
			return ErrInvalidDocument
		}
	}
	queryer, ok := s.db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return errors.New("export requires a query-capable executor")
	}

	query := "SELECT " + strings.Join(col.Columns, ", ") + " FROM " + col.Name + " ORDER BY " + col.Columns[0]
	rows, err := queryer.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("export %s: %w", col.Name, err)
	}
	defer rows.Close()

	for rows.Next() {
		values := make([]any, len(col.Columns))
		ptrs := make([]any, len(col.Columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}
		if err := fn(values); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *sqlRecordSource) discoverExtraTables(ctx context.Context, seen map[string]bool) ([]string, error) {
	queryer, ok := s.db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("discover tables requires query-capable executor")
	}
	var query string
	switch s.dialect {
	case database.SQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
	case database.Postgres:
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY table_name"
	case database.MySQL:
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name"
	}
	rows, err := queryer.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if seen[name] || !isSafeCollection(name) {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result, rows.Err()
}

func (s *sqlRecordSource) discoverColumns(ctx context.Context, table string) ([]string, error) {
	queryer, ok := s.db.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("discover columns requires query-capable executor")
	}
	var cols []string
	switch s.dialect {
	case database.SQLite:
		rows, err := queryer.QueryContext(ctx, "PRAGMA table_info("+table+")")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				return nil, err
			}
			if identPattern.MatchString(name) && !denylistedColumns[strings.ToLower(name)] {
				cols = append(cols, name)
			}
		}
		return cols, rows.Err()
	case database.Postgres, database.MySQL:
		var query string
		if s.dialect == database.Postgres {
			query = "SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position"
		} else {
			query = "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position"
		}
		rows, err := queryer.QueryContext(ctx, query, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			if identPattern.MatchString(name) && !denylistedColumns[strings.ToLower(name)] {
				cols = append(cols, name)
			}
		}
		return cols, rows.Err()
	}
	return nil, nil
}

// sqlRecordSink implements RecordSink for SQLite, PostgreSQL, and MySQL.
type sqlRecordSink struct {
	tx      database.Executor
	dialect database.Dialect
}

// NewSQLRecordSink creates a RecordSink backed by SQLite, PostgreSQL, or MySQL.
func NewSQLRecordSink(tx database.Executor, dialect database.Dialect) RecordSink {
	return &sqlRecordSink{tx: tx, dialect: dialect}
}

func (s *sqlRecordSink) Dialect() string { return string(s.dialect) }

func (s *sqlRecordSink) Prepare(ctx context.Context, collections []CollectionSchema) error {
	if err := database.RequireExecutor(s.tx); err != nil {
		return err
	}
	if s.dialect != database.SQLite && s.dialect != database.Postgres && s.dialect != database.MySQL {
		return fmt.Errorf("unsupported portable import dialect %q", s.dialect)
	}
	// Clear in reverse order so dependent child rows are deleted before parents.
	for i := len(collections) - 1; i >= 0; i-- {
		name := collections[i].Name
		if !isSafeCollection(name) {
			return ErrInvalidDocument
		}
		if _, err := s.tx.ExecContext(ctx, "DELETE FROM "+name); err != nil {
			return fmt.Errorf("clear %s: %w", name, err)
		}
	}
	return nil
}

func (s *sqlRecordSink) WriteRecord(ctx context.Context, col CollectionSchema, values []any) error {
	placeholders := make([]string, len(col.Columns))
	for i := range placeholders {
		if s.dialect == database.Postgres {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		} else {
			placeholders[i] = "?"
		}
	}
	query := "INSERT INTO " + col.Name + " (" + strings.Join(col.Columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	_, err := s.tx.ExecContext(ctx, query, values...)
	return err
}
