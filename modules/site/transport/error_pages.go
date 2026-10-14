package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) errorPages(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "Error page settings are unavailable.", http.StatusServiceUnavailable)
		return
	}
	toProps := func(status int) designsystem.PublicErrorPageProps {
		value := site.ErrorPageFromSettings(settings, status)
		links := make([]designsystem.PublicErrorLink, 0, len(value.Links))
		for _, link := range value.Links {
			links = append(links, designsystem.PublicErrorLink{Label: link.Label, URL: link.URL})
		}
		return designsystem.PublicErrorPageProps{Status: status, Headline: value.Headline, Message: value.Message, ImageURL: value.ImageURL, ActionLabel: value.ActionLabel, ActionURL: value.ActionURL, Links: links}
	}
	if ui.AdminStartForRequest(w, r, "Custom error pages", "Set helpful public pages for missing or unavailable content", "site") != nil {
		return
	}
	if err := designsystem.ErrorPageEditor(designsystem.ErrorPageEditorProps{CSRF: security.CSRFTokenFromContext(r.Context()), NotFound: toProps(http.StatusNotFound), ServerError: toProps(http.StatusInternalServerError)}).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) saveErrorPage(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	status, err := strconv.Atoi(r.PathValue("status"))
	if err != nil || status != http.StatusNotFound && status != http.StatusInternalServerError {
		http.Error(w, "Invalid error page.", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid error page.", http.StatusBadRequest)
		return
	}
	value := site.PublicErrorPage{Headline: strings.TrimSpace(r.FormValue("headline")), Message: strings.TrimSpace(r.FormValue("message")), ImageURL: strings.TrimSpace(r.FormValue("image_url")), ActionLabel: strings.TrimSpace(r.FormValue("action_label")), ActionURL: strings.TrimSpace(r.FormValue("action_url"))}
	for _, line := range strings.Split(r.FormValue("links"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		label, target, ok := strings.Cut(line, "|")
		if !ok {
			http.Error(w, "Enter links as label | /path.", http.StatusBadRequest)
			return
		}
		value.Links = append(value.Links, site.ErrorPageLink{Label: strings.TrimSpace(label), URL: strings.TrimSpace(target)})
	}
	if err := value.Validate(); err != nil {
		http.Error(w, "Check the error page text and internal links.", http.StatusBadRequest)
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "Error page could not be saved.", http.StatusInternalServerError)
		return
	}
	if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: fmt.Sprintf("site.error_page.%d", status), Value: string(encoded), Public: true}); err != nil {
		http.Error(w, "Error page could not be saved.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/error-pages?saved=1", http.StatusSeeOther)
}

func (h Handler) renderPublicErrorPage(w http.ResponseWriter, r *http.Request, status int) {
	value := site.ErrorPageFromSettings(nil, status)
	if !guards.IsNil(h.Store) {
		if settings, err := h.Store.ListPublicSettings(r.Context(), h.DB); err == nil {
			value = site.ErrorPageFromSettings(settings, status)
		}
	}
	links := make([]designsystem.PublicErrorLink, 0, len(value.Links))
	for _, link := range value.Links {
		links = append(links, designsystem.PublicErrorLink{Label: link.Label, URL: link.URL})
	}
	var body bytes.Buffer
	props := designsystem.PublicErrorPageProps{Status: status, Headline: value.Headline, Message: value.Message, ImageURL: value.ImageURL, ActionLabel: value.ActionLabel, ActionURL: value.ActionURL, Links: links}
	if err := designsystem.PublicErrorPage(props).Render(r.Context(), &body); err != nil {
		http.Error(w, "This page is unavailable.", status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
