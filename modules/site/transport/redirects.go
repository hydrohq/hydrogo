package transport

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) redirectFallback(w http.ResponseWriter, r *http.Request) {
	trackMiss := func() {
		if h.Service != nil && trackableNotFoundPath(r.URL.Path) {
			_ = h.Service.RecordNotFound(r.Context(), r.URL.Path, time.Now().UTC())
		}
	}
	if h.Redirects == nil {
		trackMiss()
		h.renderPublicErrorPage(w, r, http.StatusNotFound)
		return
	}
	scope, err := tenant.FromContext(r.Context())
	if err != nil && h.Redirects.RequireTenantScope {
		trackMiss()
		h.renderPublicErrorPage(w, r, http.StatusNotFound)
		return
	}
	rule, ok := h.Redirects.Match(scope, r.URL.Path)
	if !ok {
		trackMiss()
		h.renderPublicErrorPage(w, r, http.StatusNotFound)
		return
	}
	target := rule.TargetPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, rule.Status)
}

func trackableNotFoundPath(path string) bool {
	if path == "" || len(path) > 2048 {
		return false
	}
	for _, prefix := range []string{"/admin", "/api", "/auth", "/health", "/metrics", "/ready", "/live", "/.well-known", "/password-reset", "/invite", "/verify-email"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return false
		}
	}
	return true
}

func (h Handler) notFoundPaths(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListNotFound(r.Context(), 100)
	if err != nil {
		http.Error(w, "redirect report unavailable", http.StatusServiceUnavailable)
		return
	}
	rows := make([]ui.NotFoundPathRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ui.NotFoundPathRow{Path: item.Path, Hits: strconv.FormatInt(item.Hits, 10), LastSeen: item.LastSeen.Local().Format("02 Jan 2006 15:04")})
	}
	if ui.AdminStartForRequest(w, r, "404 izleyici", "Sık ziyaret edilen eksik bağlantıları yönlendirin", "site") != nil {
		return
	}
	if err := ui.NotFoundPaths(rows, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) convertNotFoundPath(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || h.Redirects == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	source, err := canonical.Path(r.FormValue("source"))
	if err != nil || source == "" || len(source) > 2048 {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	source = "/" + source
	target, err := canonical.Path(r.FormValue("target"))
	if err != nil || target == "" || len(target) > 2048 {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	target = "/" + target
	id, err := randomID("not-found-")
	if err != nil {
		http.Error(w, "redirect could not be saved", http.StatusServiceUnavailable)
		return
	}
	rule := site.RedirectRule{ID: id, Pattern: source, Kind: "exact", TargetPath: target, Status: http.StatusMovedPermanently, CreatedAt: time.Now().UTC()}
	if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
		rule.TenantID = scope.TenantID
	}
	if err := site.ValidateRedirectRule(rule); err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	if err := h.Redirects.UpsertWith(rule, func() error {
		return h.Service.SaveRedirectRule(r.Context(), rule)
	}); err != nil {
		http.Error(w, "redirect could not be saved", siteStatus(err))
		return
	}
	if err := h.Service.DeleteNotFound(r.Context(), source); err != nil {
		http.Error(w, "redirect saved; report cleanup unavailable", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/admin/redirects", http.StatusSeeOther)
}

func (h Handler) redirectRules(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListRedirectRules(r.Context(), 500)
	if err != nil {
		http.Error(w, "redirects unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.Redirects != nil {
		tenantID := ""
		if h.Redirects.RequireTenantScope {
			if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
				tenantID = scope.TenantID
			}
		}
		hits := make(map[string]uint64)
		for _, rule := range h.Redirects.List(tenantID) {
			hits[rule.ID] = rule.Hits
		}
		for i := range items {
			if count, ok := hits[items[i].ID]; ok {
				items[i].Hits = count
			}
		}
	}
	rows := make([][]string, 0, len(items))
	rowActions := make([][]uicomponents.TableAction, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Pattern, item.Kind, item.TargetPath, strconv.Itoa(item.Status), strconv.FormatUint(item.Hits, 10)})
		rowActions = append(rowActions, []uicomponents.TableAction{{Label: "Kaldır", Action: "/admin/site/redirect-rules/" + url.PathEscape(item.ID) + "/delete", CSRF: security.CSRFTokenFromContext(r.Context()), Variant: uicomponents.Danger}})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Yönlendirmeler", "Eski bağlantıları güncel sayfalara yönetin", "site") != nil {
		return
	}
	_ = redirectReportLink().Render(r.Context(), w)
	newRuleBody := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Input(uicomponents.InputProps{ID: "redirect-pattern", Name: "pattern", Label: "Eski yol", Placeholder: "/old-page", Required: true, MaxLength: 512, HasMaxLength: true}),
			uicomponents.Select(uicomponents.SelectProps{ID: "redirect-kind", Name: "kind", Label: "Eşleştirme", Options: []string{"exact", "wildcard", "regex"}}),
			uicomponents.Input(uicomponents.InputProps{ID: "redirect-target", Name: "target", Label: "Yeni yol", Placeholder: "/new-page", Required: true, MaxLength: 2048, HasMaxLength: true}),
			uicomponents.Select(uicomponents.SelectProps{ID: "redirect-status", Name: "status", Label: "Durum", Options: []string{"301", "302"}}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "Yönlendirme ekle", Type: "submit", Variant: uicomponents.Primary}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	if err := uicomponents.Card(uicomponents.CardProps{Title: "Yeni yönlendirme", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/redirect-rules", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: newRuleBody})}).Render(r.Context(), w); err != nil {
		return
	}
	importBody := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Input(uicomponents.InputProps{ID: "redirect-csv", Name: "file", Label: "CSV dosyası", Type: "file", Required: true}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "İçe aktar", Type: "submit", Variant: uicomponents.Quiet}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	if err := uicomponents.Card(uicomponents.CardProps{Title: "CSV içe aktar", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/redirect-rules/import", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: importBody})}).Render(r.Context(), w); err != nil {
		return
	}
	_ = uicomponents.Table(uicomponents.TableProps{Caption: "Yönlendirme kuralları", Columns: []string{"Eski yol", "Eşleştirme", "Yeni yol", "HTTP", "Ziyaret"}, Rows: rows, RowActions: rowActions, EmptyMessage: "Henüz yönlendirme kuralı yok."}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveRedirectRule(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	id, err := randomID("redirect-")
	if err != nil {
		http.Error(w, "redirects unavailable", http.StatusServiceUnavailable)
		return
	}
	rule := normalizeRedirectRule(site.RedirectRule{ID: id, Pattern: r.FormValue("pattern"), Kind: r.FormValue("kind"), TargetPath: r.FormValue("target"), CreatedAt: time.Now().UTC()})
	rule.Status, err = strconv.Atoi(r.FormValue("status"))
	if err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
		rule.TenantID = scope.TenantID
	}
	persist := func() error { return h.Service.SaveRedirectRule(r.Context(), rule) }
	if h.Redirects != nil {
		err = h.Redirects.UpsertWith(rule, persist)
	} else {
		err = persist()
	}
	if err != nil {
		http.Error(w, "redirect could not be saved", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/redirect-rules", http.StatusSeeOther)
}

func (h Handler) importRedirectRules(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "invalid CSV", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "invalid CSV", http.StatusBadRequest)
		return
	}
	defer file.Close()
	reader := csv.NewReader(io.LimitReader(file, 1<<20))
	reader.FieldsPerRecord = 4
	rules := make([]site.RedirectRule, 0, 64)
	for {
		row, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil || len(rules) >= 1000 {
			http.Error(w, "invalid CSV", http.StatusBadRequest)
			return
		}
		if len(rules) == 0 && strings.EqualFold(strings.TrimSpace(row[0]), "pattern") {
			continue
		}
		id, idErr := randomID("redirect-")
		if idErr != nil {
			http.Error(w, "redirects unavailable", http.StatusServiceUnavailable)
			return
		}
		status, parseErr := strconv.Atoi(strings.TrimSpace(row[3]))
		if parseErr != nil {
			http.Error(w, "invalid CSV", http.StatusBadRequest)
			return
		}
		rule := normalizeRedirectRule(site.RedirectRule{ID: id, Pattern: row[0], Kind: row[1], TargetPath: row[2], Status: status, CreatedAt: time.Now().UTC()})
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			rule.TenantID = scope.TenantID
		}
		if err := site.ValidateRedirectRule(rule); err != nil {
			http.Error(w, "invalid CSV", http.StatusBadRequest)
			return
		}
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		http.Error(w, "invalid CSV", http.StatusBadRequest)
		return
	}
	persist := func() error { return h.Service.SaveRedirectRules(r.Context(), rules) }
	if h.Redirects != nil {
		err = h.Redirects.UpsertManyWith(rules, persist)
	} else {
		err = persist()
	}
	if err != nil {
		http.Error(w, "redirect rules could not be imported", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/redirect-rules", http.StatusSeeOther)
}

func (h Handler) deleteRedirectRule(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := r.PathValue("id")
	if h.Redirects != nil {
		tenantID := ""
		if h.Redirects.RequireTenantScope {
			if scope, err := tenant.FromContext(r.Context()); err == nil {
				tenantID = scope.TenantID
			}
		}
		if err := h.Redirects.RemoveWith(tenantID, id, func() error {
			return h.Service.DeleteRedirectRule(r.Context(), id)
		}); err != nil {
			http.Error(w, "redirect could not be removed", siteStatus(err))
			return
		}
	} else {
		if err := h.Service.DeleteRedirectRule(r.Context(), id); err != nil {
			http.Error(w, "redirect could not be removed", siteStatus(err))
			return
		}
	}
	http.Redirect(w, r, "/admin/site/redirect-rules", http.StatusSeeOther)
}

func (h Handler) saveRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Redirect
	if !decodeJSON(w, r, &value) {
		return
	}
	persist := func() error { return h.Service.SaveRedirect(r.Context(), value) }
	if h.Redirects != nil {
		source, sourceErr := canonical.Path(value.SourcePath)
		target, targetErr := canonical.Path(value.TargetPath)
		if sourceErr != nil || targetErr != nil {
			http.Error(w, "invalid redirect", http.StatusBadRequest)
			return
		}
		rule := site.RedirectRule{ID: "legacy:/" + source, Pattern: "/" + source, Kind: "exact", TargetPath: "/" + target, Status: value.Status, CreatedAt: value.CreatedAt}
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			rule.TenantID = scope.TenantID
		}
		value.SourcePath = rule.Pattern
		value.TargetPath = rule.TargetPath
		persist = func() error { return h.Service.SaveRedirect(r.Context(), value) }
		if err := h.Redirects.UpsertWith(rule, persist); err != nil {
			http.Error(w, "invalid redirect", siteStatus(err))
			return
		}
	} else if err := persist(); err != nil {
		http.Error(w, "invalid redirect", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeRedirectRule(rule site.RedirectRule) site.RedirectRule {
	rule.Pattern = strings.TrimSpace(rule.Pattern)
	rule.TargetPath = strings.TrimSpace(rule.TargetPath)
	if rule.Kind == "exact" {
		if path, err := canonical.Path(rule.Pattern); err == nil {
			rule.Pattern = "/" + path
		}
	}
	if !strings.HasPrefix(rule.TargetPath, "//") && !strings.Contains(rule.TargetPath, "*") && rule.Kind != "regex" {
		if path, err := canonical.Path(rule.TargetPath); err == nil {
			rule.TargetPath = "/" + path
		}
	}
	return rule
}
