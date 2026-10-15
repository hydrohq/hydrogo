package transport

import (
	"context"
	"net/http"
	"strings"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

const maxTermIDs = 64

func (h Handler) taxonomyRepository(w http.ResponseWriter, r *http.Request) (taxonomyAdminRepository, bool) {
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news taxonomy is unavailable", nil)
		return nil, false
	}
	return repository, true
}

// GET /api/v1/admin/news/taxonomy lists every term.
func (h Handler) adminTaxonomyCatalog(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	repository, ok := h.taxonomyRepository(w, r)
	if !ok {
		return
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news taxonomy is unavailable", nil)
		return
	}
	if terms == nil {
		terms = []v1.TaxonomyTerm{}
	}
	api.WriteData(w, http.StatusOK, terms, nil)
}

// POST /api/v1/admin/news/taxonomy creates or updates a term.
func (h Handler) adminTaxonomyTerm(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	repository, ok := h.taxonomyRepository(w, r)
	if !ok {
		return
	}
	var term v1.TaxonomyTerm
	if !api.DecodeJSON(w, r, &term, 16<<10) {
		return
	}
	fields := map[string]string{}
	if strings.TrimSpace(term.ID) == "" {
		fields["id"] = "required"
	}
	if strings.TrimSpace(term.Slug) == "" {
		fields["slug"] = "required"
	}
	if strings.TrimSpace(term.Kind) == "" {
		fields["kind"] = "required"
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	if term.Labels == nil {
		term.Labels = map[string]string{}
	}
	if err := upsertTaxonomyTermAtomically(r.Context(), h.DB, repository, term); err != nil {
		writeNewsError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, term, nil)
}

// DELETE /api/v1/admin/news/taxonomy/{id}
func (h Handler) adminTaxonomyTermDelete(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	repository, ok := h.taxonomyRepository(w, r)
	if !ok {
		return
	}
	if err := deleteTaxonomyTermAtomically(r.Context(), h.DB, repository, r.PathValue("id")); err != nil {
		writeNewsError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/admin/news/{id}/taxonomy lists the terms attached to an item.
func (h Handler) adminItemTaxonomy(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	repository, ok := h.taxonomyRepository(w, r)
	if !ok {
		return
	}
	if _, err := h.Service.Repository.Get(r.Context(), h.DB, r.PathValue("id")); err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	terms, err := repository.ListTaxonomy(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news taxonomy is unavailable", nil)
		return
	}
	if terms == nil {
		terms = []v1.TaxonomyTerm{}
	}
	api.WriteData(w, http.StatusOK, terms, nil)
}

type termIDsInput struct {
	TermIDs []string `json:"term_ids"`
}

// PUT /api/v1/admin/news/{id}/taxonomy {term_ids} replaces the item's terms;
// unknown IDs are ignored.
func (h Handler) adminReplaceItemTaxonomy(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	if _, ok := h.taxonomyRepository(w, r); !ok {
		return
	}
	var value termIDsInput
	if !api.DecodeJSON(w, r, &value, 16<<10) {
		return
	}
	if _, err := h.Service.Repository.Get(r.Context(), h.DB, r.PathValue("id")); err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	if err := h.replaceTerms(r, r.PathValue("id"), value.TermIDs); err != nil {
		writeNewsError(w, r, err)
		return
	}
	h.adminItemTaxonomy(w, r)
}

func (h Handler) replaceTerms(r *http.Request, contentID string, termIDs []string) error {
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok {
		return nil
	}
	if len(termIDs) > maxTermIDs {
		termIDs = termIDs[:maxTermIDs]
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		return err
	}
	allowed := make(map[string]v1.TaxonomyTerm, len(terms))
	for _, term := range terms {
		allowed[term.ID] = term
	}
	selected := make([]v1.TaxonomyTerm, 0, len(termIDs))
	seen := make(map[string]struct{}, len(termIDs))
	for _, id := range termIDs {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		if term, exists := allowed[id]; exists {
			selected = append(selected, term)
			seen[id] = struct{}{}
		}
	}
	return replaceTaxonomyAtomically(r.Context(), h.DB, repository, contentID, selected)
}

func replaceTaxonomyAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, contentID string, terms []v1.TaxonomyTerm) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.ReplaceTaxonomy(ctx, executor, contentID, terms)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.ReplaceTaxonomy(ctx, tx, contentID, terms); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertTaxonomyTermAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, term v1.TaxonomyTerm) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.UpsertTaxonomyTerm(ctx, executor, term)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.UpsertTaxonomyTerm(ctx, tx, term); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteTaxonomyTermAtomically(ctx context.Context, executor database.Executor, repository taxonomyAdminRepository, id string) error {
	starter, ok := executor.(transactionStarter)
	if !ok {
		return repository.DeleteTaxonomyTerm(ctx, executor, id)
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := repository.DeleteTaxonomyTerm(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
