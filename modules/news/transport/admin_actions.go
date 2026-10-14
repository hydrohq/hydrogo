package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func (h Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsItemFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = h.defaultLocale()
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expiry date", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: id, Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), CoverMediaID: r.FormValue("cover_media_id"), Excerpt: r.FormValue("excerpt"), KeyTakeaways: r.FormValue("key_takeaways"), Category: r.FormValue("category"), AuthorName: r.FormValue("author_name"), Featured: r.FormValue("featured") == "on", Tags: r.FormValue("tags"), SEOTitle: r.FormValue("seo_title"), SEODescription: r.FormValue("seo_description"), ExpiresAt: expiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(id), http.StatusSeeOther)
}

func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsItemFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	locale := r.FormValue("locale")
	if locale == "" {
		locale = h.defaultLocale()
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expiry date", http.StatusBadRequest)
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: locale, Slug: r.FormValue("slug"), Title: r.FormValue("title"), Body: r.FormValue("body"), CoverMediaID: r.FormValue("cover_media_id"), Excerpt: r.FormValue("excerpt"), KeyTakeaways: r.FormValue("key_takeaways"), Category: r.FormValue("category"), AuthorName: r.FormValue("author_name"), Featured: r.FormValue("featured") == "on", Tags: r.FormValue("tags"), SEOTitle: r.FormValue("seo_title"), SEODescription: r.FormValue("seo_description"), ExpiresAt: expiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateDraft(r.Context(), item, version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	if repository, ok := h.Service.Repository.(taxonomyAdminRepository); ok {
		terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
		if err == nil {
			allowed := make(map[string]v1.TaxonomyTerm, len(terms))
			for _, term := range terms {
				allowed[term.ID] = term
			}
			selected := make([]v1.TaxonomyTerm, 0, len(r.Form["taxonomy_term"]))
			for _, id := range r.Form["taxonomy_term"] {
				if term, exists := allowed[id]; exists {
					selected = append(selected, term)
				}
			}
			if err := replaceTaxonomyAtomically(r.Context(), h.DB, repository, item.ID, selected); err != nil {
				api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
				return
			}
		}
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(item.ID)+"?saved=1", http.StatusSeeOther)
}

func (h Handler) adminTaxonomy(w http.ResponseWriter, r *http.Request) {
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
	if !ok {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	terms, err := repository.ListTaxonomyTerms(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "news taxonomy unavailable", http.StatusServiceUnavailable)
		return
	}
	allowed := make(map[string]v1.TaxonomyTerm, len(terms))
	for _, term := range terms {
		allowed[term.ID] = term
	}
	selected := make([]v1.TaxonomyTerm, 0, len(r.Form["taxonomy_term"]))
	for _, id := range r.Form["taxonomy_term"] {
		if term, exists := allowed[id]; exists {
			selected = append(selected, term)
		}
	}
	if err := replaceTaxonomyAtomically(r.Context(), h.DB, repository, r.PathValue("id"), selected); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) adminSchedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsWorkflowFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	at, err := time.Parse("2006-01-02T15:04", r.FormValue("scheduled_at"))
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), version, at); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}

func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "publish")
}

func (h Handler) adminRetract(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "retract")
}

func (h Handler) adminPreview(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	publicPath := "/public/news/" + url.PathEscape(item.Locale) + "/" + url.PathEscape(item.Slug) + "/html"
	// Every previewable status gets a revision-bound token: the admin embeds
	// this URL in a same-origin iframe, and only token-carrying responses may
	// be framed (see publicGetHTML). Published items without a signer fall
	// back to the plain public URL.
	previewable := item.Status == news.StatusPublished || item.Status == news.StatusDraft || item.Status == news.StatusScheduled || item.Status == news.StatusRetracted
	if !previewable || (h.PreviewSigner == nil && item.Status != news.StatusPublished) {
		http.Error(w, "news preview unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.PreviewSigner != nil {
		claims := preview.Claims{ResourceType: "news", ResourceID: item.ID, Revision: item.Revision}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			claims.TenantID = scope.TenantID
		} else if h.RequireTenantScope {
			http.Error(w, "news preview unavailable", http.StatusForbidden)
			return
		}
		token, err := h.PreviewSigner.Issue(claims, time.Now().UTC(), 15*time.Minute)
		if err != nil {
			http.Error(w, "news preview unavailable", http.StatusServiceUnavailable)
			return
		}
		publicPath += "?preview_token=" + url.QueryEscape(token)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"url":%q}`, publicPath)
}

func (h Handler) adminTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxNewsWorkflowFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	version, err := strconv.Atoi(r.FormValue("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if action == "publish" {
		err = h.Service.Publish(r.Context(), r.PathValue("id"), version)
	} else {
		err = h.Service.Retract(r.Context(), r.PathValue("id"), version)
	}
	if err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	http.Redirect(w, r, "/admin/news/"+url.PathEscape(r.PathValue("id")), http.StatusSeeOther)
}
