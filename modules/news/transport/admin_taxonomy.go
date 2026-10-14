package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h Handler) adminTaxonomyCatalog(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	csrf := security.CSRFTokenFromContext(r.Context())
	table := uicomponents.TableProps{
		Caption:      "Terim kataloğu",
		Columns:      []string{"ID", "Slug", "Tür", "Üst terim", "Etiketler"},
		EmptyMessage: "Henüz terim yok.",
	}
	for _, term := range terms {
		// The table escapes cell text, so the JSON keeps "&" readable.
		var labels strings.Builder
		encoder := json.NewEncoder(&labels)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(term.Labels)
		table.Rows = append(table.Rows, []string{term.ID, term.Slug, term.Kind, term.Parent, strings.TrimSpace(labels.String())})
		table.RowActions = append(table.RowActions, []uicomponents.TableAction{{Label: "Sil", Action: "/admin/news/taxonomy/" + url.PathEscape(term.ID) + "/delete", CSRF: csrf, Variant: uicomponents.Danger}})
	}
	renderAdminScreen(w, r, "Taxonomy", "Haber sınıflandırması", newsTaxonomyScreen(csrf, table))
}

func (h Handler) adminTaxonomyTerm(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsTaxonomyFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(r.FormValue("labels_json")), &labels); err != nil {
		http.Error(w, "invalid taxonomy labels", http.StatusBadRequest)
		return
	}
	term := v1.TaxonomyTerm{ID: r.FormValue("id"), Slug: r.FormValue("slug"), Kind: r.FormValue("kind"), Parent: r.FormValue("parent"), Labels: labels}
	if err := upsertTaxonomyTermAtomically(r.Context(), h.DB, repository, term); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/taxonomy", http.StatusSeeOther)
}

func (h Handler) adminTaxonomyTermDelete(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	repository, ok := h.Service.Repository.(taxonomyAdminRepository)
	if !ok || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := deleteTaxonomyTermAtomically(r.Context(), h.DB, repository, r.PathValue("id")); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/taxonomy", http.StatusSeeOther)
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
