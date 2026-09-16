package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type taxonomyQueryExecutor interface {
	database.Executor
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// ReplaceTaxonomy atomically replaces the terms attached to one news item.
// The tables and authorization policy remain owned by the News module.
func (s Store) ReplaceTaxonomy(ctx context.Context, tx database.Executor, contentID string, terms []v1.TaxonomyTerm) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	contentID = strings.TrimSpace(contentID)
	relations := []v1.TaxonomyRelation{{ContentID: contentID}}
	for _, term := range terms {
		relations[0].TermIDs = append(relations[0].TermIDs, term.ID)
	}
	if err := v1.ValidateTaxonomy(terms, relations); err != nil {
		return err
	}
	for _, term := range v1.StableTaxonomyTerms(terms) {
		labels, err := json.Marshal(term.Labels)
		if err != nil {
			return fmt.Errorf("encode taxonomy labels: %w", err)
		}
		query := `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET slug=EXCLUDED.slug,kind=EXCLUDED.kind,parent_id=EXCLUDED.parent_id,labels_json=EXCLUDED.labels_json`
		args := []any{term.ID, term.Slug, term.Kind, nullableTaxonomyParent(term.Parent), string(labels)}
		if s.Dialect == database.SQLite {
			query = `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,kind=excluded.kind,parent_id=excluded.parent_id,labels_json=excluded.labels_json`
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	deleteQuery := `DELETE FROM news_taxonomy_relations WHERE content_id=$1`
	if s.Dialect == database.SQLite {
		deleteQuery = `DELETE FROM news_taxonomy_relations WHERE content_id=?`
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, contentID); err != nil {
		return err
	}
	insertQuery := `INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES ($1,$2)`
	if s.Dialect == database.SQLite {
		insertQuery = `INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES (?,?)`
	}
	for _, term := range terms {
		if _, err := tx.ExecContext(ctx, insertQuery, contentID, term.ID); err != nil {
			return err
		}
	}
	return nil
}

// ListTaxonomy returns deterministic localized terms attached to a news item.
func (s Store) ListTaxonomy(ctx context.Context, tx database.Executor, contentID string) ([]v1.TaxonomyTerm, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return nil, err
	}
	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return nil, fmt.Errorf("taxonomy executor does not support row queries")
	}
	query := `SELECT t.id,t.slug,t.kind,COALESCE(t.parent_id,''),t.labels_json FROM news_taxonomy_terms t JOIN news_taxonomy_relations r ON r.term_id=t.id WHERE r.content_id=$1 ORDER BY t.id`
	if s.Dialect == database.SQLite {
		query = `SELECT t.id,t.slug,t.kind,COALESCE(t.parent_id,''),t.labels_json FROM news_taxonomy_terms t JOIN news_taxonomy_relations r ON r.term_id=t.id WHERE r.content_id=? ORDER BY t.id`
	}
	rows, err := queryer.QueryContext(ctx, query, strings.TrimSpace(contentID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]v1.TaxonomyTerm, 0)
	for rows.Next() {
		var term v1.TaxonomyTerm
		var labels string
		if err := rows.Scan(&term.ID, &term.Slug, &term.Kind, &term.Parent, &labels); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(labels), &term.Labels); err != nil {
			return nil, fmt.Errorf("decode taxonomy labels: %w", err)
		}
		result = append(result, term)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func nullableTaxonomyParent(parent string) any {
	if strings.TrimSpace(parent) == "" {
		return nil
	}
	return parent
}
