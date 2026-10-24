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
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type taxonomyQueryExecutor interface {
	database.Executor
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

var ErrTaxonomyTermInUse = news.ErrTaxonomyTermInUse
var ErrTaxonomyTermHasChildren = news.ErrTaxonomyTermHasChildren
var ErrTaxonomyTermIDTaken = news.ErrTaxonomyTermIDTaken

// taxonomyTenantID returns the scope's tenant id, or "" when the store is
// not tenant-scoped (single-customer deployments keep the empty-string
// discriminator every row already defaults to).
func (s Store) taxonomyTenantID(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

// UpsertTaxonomyTerm stores one validated module-owned taxonomy term. The
// id is client-supplied, so this never does a bare INSERT ... ON CONFLICT
// DO UPDATE keyed only on id -- that would let one tenant silently overwrite
// another tenant's term by guessing or reusing its id. Instead: try an
// UPDATE scoped to (id, tenant_id) first; if it matched nothing, check
// whether the id exists for any OTHER tenant before inserting a new row.
func (s Store) UpsertTaxonomyTerm(ctx context.Context, tx database.Executor, term v1.TaxonomyTerm) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.taxonomyTenantID(ctx)
	if err != nil {
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

	updateQuery := `UPDATE news_taxonomy_terms SET slug=` + s.p(1) + `,kind=` + s.p(2) + `,parent_id=` + s.p(3) + `,labels_json=` + s.p(4) + ` WHERE id=` + s.p(5) + ` AND tenant_id=` + s.p(6)
	result, err := tx.ExecContext(ctx, updateQuery, term.Slug, term.Kind, nullableTaxonomyParent(term.Parent), string(labels), term.ID, tenantID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected > 0 {
		return nil
	}

	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return fmt.Errorf("taxonomy executor does not support row queries")
	}
	var existing int
	existsQuery := `SELECT COUNT(*) FROM news_taxonomy_terms WHERE id=` + s.p(1)
	if err := queryer.QueryRowContext(ctx, existsQuery, term.ID).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return ErrTaxonomyTermIDTaken
	}

	insertQuery := `INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json,tenant_id) VALUES (` +
		s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `)`
	_, err = tx.ExecContext(ctx, insertQuery, term.ID, term.Slug, term.Kind, nullableTaxonomyParent(term.Parent), string(labels), tenantID)
	return err
}

// DeleteTaxonomyTerm removes an unused leaf term without silently changing
// content, and never reaches outside its own tenant to do so.
func (s Store) DeleteTaxonomyTerm(ctx context.Context, tx database.Executor, id string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.taxonomyTenantID(ctx)
	if err != nil {
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
	childQuery := `SELECT COUNT(*) FROM news_taxonomy_terms WHERE parent_id=` + s.p(1) + ` AND tenant_id=` + s.p(2)
	if err := queryer.QueryRowContext(ctx, childQuery, id, tenantID).Scan(&childCount); err != nil {
		return err
	}
	if childCount > 0 {
		return ErrTaxonomyTermHasChildren
	}
	var relationCount int
	relationQuery := `SELECT COUNT(*) FROM news_taxonomy_relations WHERE term_id=` + s.p(1) + ` AND tenant_id=` + s.p(2)
	if err := queryer.QueryRowContext(ctx, relationQuery, id, tenantID).Scan(&relationCount); err != nil {
		return err
	}
	if relationCount > 0 {
		return ErrTaxonomyTermInUse
	}
	deleteQuery := `DELETE FROM news_taxonomy_terms WHERE id=` + s.p(1) + ` AND tenant_id=` + s.p(2)
	result, err := tx.ExecContext(ctx, deleteQuery, id, tenantID)
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
// Every statement is also scoped to the caller's tenant, independent of
// whatever ownership check the caller already did on contentID itself --
// relation rows must never be reachable across a tenant boundary either.
func (s Store) ReplaceTaxonomy(ctx context.Context, tx database.Executor, contentID string, terms []v1.TaxonomyTerm) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.taxonomyTenantID(ctx)
	if err != nil {
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
		if err := s.UpsertTaxonomyTerm(ctx, tx, term); err != nil {
			return err
		}
	}
	deleteQuery := `DELETE FROM news_taxonomy_relations WHERE content_id=` + s.p(1) + ` AND tenant_id=` + s.p(2)
	if _, err := tx.ExecContext(ctx, deleteQuery, contentID, tenantID); err != nil {
		return err
	}
	insertQuery := `INSERT INTO news_taxonomy_relations (content_id,term_id,tenant_id) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `)`
	for _, term := range terms {
		if _, err := tx.ExecContext(ctx, insertQuery, contentID, term.ID, tenantID); err != nil {
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
	tenantID, err := s.taxonomyTenantID(ctx)
	if err != nil {
		return nil, err
	}
	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return nil, fmt.Errorf("taxonomy executor does not support row queries")
	}
	query := `SELECT t.id,t.slug,t.kind,COALESCE(t.parent_id,''),t.labels_json FROM news_taxonomy_terms t JOIN news_taxonomy_relations r ON r.term_id=t.id AND r.tenant_id=t.tenant_id WHERE r.content_id=` + s.p(1) + ` AND r.tenant_id=` + s.p(2) + ` ORDER BY t.id`
	rows, err := queryer.QueryContext(ctx, query, strings.TrimSpace(contentID), tenantID)
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

// ListTaxonomyTerms returns the module-owned term catalog in stable order,
// scoped to the caller's tenant.
func (s Store) ListTaxonomyTerms(ctx context.Context, tx database.Executor) ([]v1.TaxonomyTerm, error) {
	if err := database.RequireExecutor(tx); err != nil {
		return nil, err
	}
	tenantID, err := s.taxonomyTenantID(ctx)
	if err != nil {
		return nil, err
	}
	queryer, ok := tx.(taxonomyQueryExecutor)
	if !ok {
		return nil, fmt.Errorf("taxonomy executor does not support row queries")
	}
	query := `SELECT id,slug,kind,COALESCE(parent_id,''),labels_json FROM news_taxonomy_terms WHERE tenant_id=` + s.p(1) + ` ORDER BY id`
	rows, err := queryer.QueryContext(ctx, query, tenantID)
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
