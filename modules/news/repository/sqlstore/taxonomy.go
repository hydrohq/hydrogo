package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type taxonomyQueryExecutor interface {
	database.Executor
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

var ErrTaxonomyTermInUse = errors.New("taxonomy term is in use")
var ErrTaxonomyTermHasChildren = errors.New("taxonomy term has children")

// UpsertTaxonomyTerm stores one validated module-owned taxonomy term.
func (s Store) UpsertTaxonomyTerm(ctx context.Context, tx database.Executor, term v1.TaxonomyTerm) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	terms, err := s.ListTaxonomyTerms(ctx, tx)
	if err != nil {
		return err
	}
	found := false
	for index := range terms {
		if terms[index].ID == term.ID {
			terms[index] = term
			found = true
			break
		}
	}
	if !found {
		terms = append(terms, term)
	}
	if err := v1.ValidateTaxonomy(terms, nil); err != nil {
		return err
	}
	labels, err := json.Marshal(term.Labels)
	if err != nil {
		return fmt.Errorf("encode taxonomy labels: %w", err)
	}
	query := `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET slug=EXCLUDED.slug,kind=EXCLUDED.kind,parent_id=EXCLUDED.parent_id,labels_json=EXCLUDED.labels_json`
	switch s.Dialect {
	case database.SQLite:
		query = `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,kind=excluded.kind,parent_id=excluded.parent_id,labels_json=excluded.labels_json`
	case database.MySQL:
		query = `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE slug=VALUES(slug),kind=VALUES(kind),parent_id=VALUES(parent_id),labels_json=VALUES(labels_json)`
	}
	_, err = tx.ExecContext(ctx, query, term.ID, term.Slug, term.Kind, nullableTaxonomyParent(term.Parent), string(labels))
	return err
}

// DeleteTaxonomyTerm removes an unused leaf term without silently changing content.
func (s Store) DeleteTaxonomyTerm(ctx context.Context, tx database.Executor, id string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("taxonomy term id is required")
	}
	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return fmt.Errorf("taxonomy executor does not support row queries")
	}
	var childCount int
	childQuery := `SELECT COUNT(*) FROM news_taxonomy_terms WHERE parent_id=$1`
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
		childQuery = `SELECT COUNT(*) FROM news_taxonomy_terms WHERE parent_id=?`
	}
	if err := queryer.QueryRowContext(ctx, childQuery, id).Scan(&childCount); err != nil {
		return err
	}
	if childCount > 0 {
		return ErrTaxonomyTermHasChildren
	}
	var relationCount int
	relationQuery := `SELECT COUNT(*) FROM news_taxonomy_relations WHERE term_id=$1`
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
		relationQuery = `SELECT COUNT(*) FROM news_taxonomy_relations WHERE term_id=?`
	}
	if err := queryer.QueryRowContext(ctx, relationQuery, id).Scan(&relationCount); err != nil {
		return err
	}
	if relationCount > 0 {
		return ErrTaxonomyTermInUse
	}
	deleteQuery := `DELETE FROM news_taxonomy_terms WHERE id=$1`
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
		deleteQuery = `DELETE FROM news_taxonomy_terms WHERE id=?`
	}
	result, err := tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	// Validate relations against the complete catalog, not only selected terms;
	// a child can be attached while its parent is not selected for this article.
	catalog, err := s.ListTaxonomyTerms(ctx, tx)
	if err != nil {
		return err
	}
	byID := make(map[string]v1.TaxonomyTerm, len(catalog)+len(terms))
	for _, term := range catalog {
		byID[term.ID] = term
	}
	for _, term := range terms {
		byID[term.ID] = term
	}
	allTerms := make([]v1.TaxonomyTerm, 0, len(byID))
	for _, term := range byID {
		allTerms = append(allTerms, term)
	}
	if err := v1.ValidateTaxonomy(allTerms, relations); err != nil {
		return err
	}
	for _, term := range v1.StableTaxonomyTerms(terms) {
		labels, err := json.Marshal(term.Labels)
		if err != nil {
			return fmt.Errorf("encode taxonomy labels: %w", err)
		}
		query := `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET slug=EXCLUDED.slug,kind=EXCLUDED.kind,parent_id=EXCLUDED.parent_id,labels_json=EXCLUDED.labels_json`
		args := []any{term.ID, term.Slug, term.Kind, nullableTaxonomyParent(term.Parent), string(labels)}
		switch s.Dialect {
		case database.SQLite:
			query = `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,kind=excluded.kind,parent_id=excluded.parent_id,labels_json=excluded.labels_json`
		case database.MySQL:
			query = `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE slug=VALUES(slug),kind=VALUES(kind),parent_id=VALUES(parent_id),labels_json=VALUES(labels_json)`
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	deleteQuery := `DELETE FROM news_taxonomy_relations WHERE content_id=$1`
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
		deleteQuery = `DELETE FROM news_taxonomy_relations WHERE content_id=?`
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, contentID); err != nil {
		return err
	}
	insertQuery := `INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES ($1,$2)`
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
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
	if s.Dialect == database.SQLite || s.Dialect == database.MySQL {
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

// ListTaxonomyTerms returns the module-owned term catalog in stable order.
func (s Store) ListTaxonomyTerms(ctx context.Context, tx database.Executor) ([]v1.TaxonomyTerm, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return nil, err
	}
	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return nil, fmt.Errorf("taxonomy executor does not support row queries")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id,slug,kind,COALESCE(parent_id,''),labels_json FROM news_taxonomy_terms ORDER BY id`)
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
