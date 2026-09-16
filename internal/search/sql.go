package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// DefaultSQLMaxDocuments bounds the durable local index. A search kit can
// choose a larger value explicitly, but an unbounded table scan is never the
// default.
const DefaultSQLMaxDocuments = 100000

var ErrSQLIndexFull = errors.New("sql search index is full")

// SQLProvider is a durable, provider-neutral search adapter for the SQLite
// and PostgreSQL profiles. It stores only the module-owned searchable
// projection; callers remain responsible for publishing and authorization.
// The same implementation works with a database transaction for atomic
// rebuilds.
type SQLProvider struct {
	Executor database.Executor
	Dialect  database.Dialect
	Max      int
}

func (p SQLProvider) maxDocuments() int {
	if p.Max <= 0 || p.Max > DefaultSQLMaxDocuments {
		return DefaultSQLMaxDocuments
	}
	return p.Max
}

func (p SQLProvider) validate() error {
	if err := database.RequireExecutor(p.Executor); err != nil {
		return err
	}
	if p.Dialect != database.SQLite && p.Dialect != database.Postgres {
		return fmt.Errorf("unsupported search dialect %q", p.Dialect)
	}
	return nil
}

func (p SQLProvider) Upsert(ctx context.Context, document Document) error {
	if err := p.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := validateDocument(document); err != nil {
		return err
	}
	document = normalizeDocument(document)
	var count int
	if err := p.Executor.QueryRowContext(ctx, "SELECT COUNT(*) FROM hydro_search_documents").Scan(&count); err != nil {
		return err
	}
	var exists int
	countQuery := "SELECT COUNT(*) FROM hydro_search_documents WHERE id = " + sqlPlaceholder(p.Dialect, 1)
	if err := p.Executor.QueryRowContext(ctx, countQuery, document.ID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 && count >= p.maxDocuments() {
		return ErrSQLIndexFull
	}
	if p.Dialect == database.Postgres {
		_, err := p.Executor.ExecContext(ctx, `INSERT INTO hydro_search_documents (id,locale,title,body) VALUES ($1,$2,$3,$4)
			ON CONFLICT (id) DO UPDATE SET locale=EXCLUDED.locale,title=EXCLUDED.title,body=EXCLUDED.body`, document.ID, document.Locale, document.Title, document.Body)
		return err
	}
	_, err := p.Executor.ExecContext(ctx, `INSERT INTO hydro_search_documents (id,locale,title,body) VALUES (?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET locale=excluded.locale,title=excluded.title,body=excluded.body`, document.ID, document.Locale, document.Title, document.Body)
	return err
}

func (p SQLProvider) Delete(ctx context.Context, id string) error {
	if err := p.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("search document id is required")
	}
	_, err := p.Executor.ExecContext(ctx, "DELETE FROM hydro_search_documents WHERE id = "+sqlPlaceholder(p.Dialect, 1), id)
	return err
}

// Rebuild atomically replaces the durable projection when called with a
// transaction. It validates the complete input before deleting old data.
func (p SQLProvider) Rebuild(ctx context.Context, documents []Document) error {
	if err := p.validate(); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if len(documents) > p.maxDocuments() {
		return ErrSQLIndexFull
	}
	next := make([]Document, 0, len(documents))
	seen := make(map[string]struct{}, len(documents))
	for _, document := range documents {
		if err := validateDocument(document); err != nil {
			return err
		}
		document = normalizeDocument(document)
		if _, ok := seen[document.ID]; ok {
			return errors.New("duplicate search document id")
		}
		seen[document.ID] = struct{}{}
		next = append(next, document)
	}
	if _, err := p.Executor.ExecContext(ctx, "DELETE FROM hydro_search_documents"); err != nil {
		return err
	}
	for _, document := range next {
		if err := p.insert(ctx, document); err != nil {
			return err
		}
	}
	return nil
}

func (p SQLProvider) insert(ctx context.Context, document Document) error {
	if p.Dialect == database.Postgres {
		_, err := p.Executor.ExecContext(ctx, `INSERT INTO hydro_search_documents (id,locale,title,body) VALUES ($1,$2,$3,$4)`, document.ID, document.Locale, document.Title, document.Body)
		return err
	}
	_, err := p.Executor.ExecContext(ctx, `INSERT INTO hydro_search_documents (id,locale,title,body) VALUES (?,?,?,?)`, document.ID, document.Locale, document.Title, document.Body)
	return err
}

func (p SQLProvider) Search(ctx context.Context, query Query) (Response, error) {
	if err := p.validate(); err != nil {
		return Response{}, err
	}
	if err := contextErr(ctx); err != nil {
		return Response{}, err
	}
	if err := query.Validate(); err != nil {
		return Response{}, err
	}
	statement := "SELECT id,locale,title,body FROM hydro_search_documents"
	args := []any{}
	if query.Locale != "" {
		statement += " WHERE locale = " + sqlPlaceholder(p.Dialect, 1)
		args = append(args, canonical.Locale(query.Locale))
	}
	statement += " ORDER BY id LIMIT " + sqlPlaceholder(p.Dialect, len(args)+1)
	args = append(args, p.maxDocuments())
	rows, err := queryRows(ctx, p.Executor, statement, args...)
	if err != nil {
		return Response{}, err
	}
	terms := tokenize(strings.ToLower(strings.TrimSpace(query.Text)))
	results := make([]Result, 0, len(rows))
	for _, document := range rows {
		title, body := strings.ToLower(document.Title), strings.ToLower(document.Body)
		score := 0.0
		for _, term := range terms {
			if strings.Contains(title, term) {
				score += 2
			}
			if strings.Contains(body, term) {
				score++
			}
		}
		if score > 0 {
			results = append(results, Result{ID: document.ID, Score: score, Title: document.Title, Snippet: snippet(document.Body, terms)})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})
	total := len(results)
	start := query.Offset
	if start > total {
		start = total
	}
	pageLimit := query.Limit
	if pageLimit <= 0 || pageLimit > MaxLimit {
		pageLimit = MaxLimit
	}
	end := start + pageLimit
	if end > total {
		end = total
	}
	return Response{Items: results[start:end], Total: total}, nil
}

type sqlDocument struct{ ID, Locale, Title, Body string }

func queryRows(ctx context.Context, executor database.Executor, statement string, args ...any) ([]sqlDocument, error) {
	queryer, ok := executor.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, errors.New("search executor does not support queries")
	}
	rows, err := queryer.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var documents []sqlDocument
	for rows.Next() {
		var document sqlDocument
		if err := rows.Scan(&document.ID, &document.Locale, &document.Title, &document.Body); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func sqlPlaceholder(dialect database.Dialect, position int) string {
	if dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
