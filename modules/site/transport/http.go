package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/internal/xlsx"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

type Handler struct {
	DB                database.Executor
	Store             site.Store
	Service           *site.Service
	Media             *media.Service
	SubmissionLimiter security.RateLimiter
	AllowedForms      map[string]struct{}
	FormsEnabled      bool
	Turnstile         interface {
		Verify(context.Context, string, string) (bool, error)
	}
	MailSpool interface {
		List(context.Context, string, int) ([]mailer.SpoolEntry, error)
		RetryNow(context.Context, string) error
	}
	Redirects             *site.RedirectRegistry
	SitemapOrigin         string
	SitemapProvider       func(context.Context) ([]site.SitemapEntry, error)
	SitemapStream         site.SitemapStream
	SitemapCategories     []string
	InvalidateDiscovery   func()
	FontsDir              string
	FontsEnabled          bool
	NavigationLocales     []string
	NavigationSuggestions func(context.Context, string) ([]site.NavigationSuggestion, error)
	Realtime              interface {
		Publish(context.Context, realtime.Event) error
	}
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "GET /sitemap-index.xml", Public: true, Handler: http.HandlerFunc(h.sitemapIndex)},
		{Pattern: "GET /sitemap-pages.xml", Public: true, Handler: http.HandlerFunc(h.sitemapPart)},
		{Pattern: "GET /sitemap-articles.xml", Public: true, Handler: http.HandlerFunc(h.sitemapArticlePart)},
		{Pattern: "POST /admin/site/settings", Handler: http.HandlerFunc(h.saveSetting)},
		{Pattern: "GET /admin/translations", Handler: http.HandlerFunc(h.translations)},
		{Pattern: "POST /admin/translations", Handler: http.HandlerFunc(h.saveTranslation)},
		{Pattern: "GET /admin/site/appearance", Handler: http.HandlerFunc(h.appearance)},
		{Pattern: "POST /admin/site/appearance", Handler: http.HandlerFunc(h.saveAppearance)},
		{Pattern: "GET /admin/site/seo", Handler: http.HandlerFunc(h.seoSettings)},
		{Pattern: "POST /admin/site/seo", Handler: http.HandlerFunc(h.saveSEOSettings)},
		{Pattern: "POST /admin/site/redirects", Handler: http.HandlerFunc(h.saveRedirect)},
		{Pattern: "GET /admin/site/redirect-rules", Handler: http.HandlerFunc(h.redirectRules)},
		{Pattern: "POST /admin/site/redirect-rules", Handler: http.HandlerFunc(h.saveRedirectRule)},
		{Pattern: "POST /admin/site/redirect-rules/import", Handler: http.HandlerFunc(h.importRedirectRules)},
		{Pattern: "POST /admin/site/redirect-rules/{id}/delete", Handler: http.HandlerFunc(h.deleteRedirectRule)},
		{Pattern: "POST /admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.saveNavigation)},
		{Pattern: "GET /admin/site/navigation/editor", Handler: http.HandlerFunc(h.navigationEditor)},
		{Pattern: "POST /admin/site/navigation/editor", Handler: http.HandlerFunc(h.saveNavigationEditor)},
		{Pattern: "GET /{path...}", Public: true, Handler: http.HandlerFunc(h.redirectFallback)},
	}
	if h.FontsEnabled {
		routes = append(routes,
			Route{Pattern: "GET /admin/site/fonts", Handler: http.HandlerFunc(h.fontPicker)},
			Route{Pattern: "POST /admin/site/fonts", Handler: http.HandlerFunc(h.localizeFont)},
		)
	}
	if h.FormsEnabled {
		routes = append(routes,
			Route{Pattern: "GET /admin/site/inbox/{form}", Handler: http.HandlerFunc(h.inbox)},
			Route{Pattern: "GET /admin/site/forms", Handler: http.HandlerFunc(h.forms)},
			Route{Pattern: "POST /admin/site/forms", Handler: http.HandlerFunc(h.saveForm)},
			Route{Pattern: "POST /admin/site/forms/{id}/working-hours", Handler: http.HandlerFunc(h.saveFormWorkingHours)},
			Route{Pattern: "POST /admin/site/forms/{id}/delete", Handler: http.HandlerFunc(h.deleteForm)},
			Route{Pattern: "POST /admin/site/inbox/{id}/status", Handler: http.HandlerFunc(h.inboxStatus)},
			Route{Pattern: "POST /admin/site/inbox/{id}/lead", Handler: http.HandlerFunc(h.leadUpdate)},
			Route{Pattern: "POST /admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.leadNoteCreate)},
			Route{Pattern: "GET /admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.leadNotes)},
			Route{Pattern: "GET /admin/site/inbox/{form}/export.csv", Handler: http.HandlerFunc(h.exportInboxCSV)},
			Route{Pattern: "GET /admin/site/inbox/{form}/export.xlsx", Handler: http.HandlerFunc(h.exportInboxXLSX)},
			Route{Pattern: "GET /admin/site/mail", Handler: http.HandlerFunc(h.mailSpool)},
			Route{Pattern: "POST /admin/site/mail/{id}/retry", Handler: http.HandlerFunc(h.mailSpoolRetry)},
			Route{Pattern: "POST /forms/{form}/submit", Public: true, Handler: http.HandlerFunc(h.submitForm)},
		)
	}
	return routes
}

func (h Handler) translations(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	localizedRows := make(map[string]map[string]i18n.TranslationRow)
	for _, cellLocale := range i18n.SupportedCatalogLocales() {
		localizedRows[cellLocale] = make(map[string]i18n.TranslationRow)
		for _, row := range i18n.CatalogRows(scope.TenantID, cellLocale) {
			localizedRows[cellLocale][row.Key] = row
		}
	}
	if ui.AdminStartForRequest(w, r, "Çeviri düzenleyici", "Sabit arayüz metinlerini düzenleyin", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
	_, _ = io.WriteString(w, `<main class="studio-card"><label for="translation-search">Çevirilerde ara</label><input id="translation-search" type="search" autocomplete="off" aria-controls="translation-matrix"><p>Eksik çeviriler sarı renkle işaretlenir. Boş bırakılan değerler Türkçe metne geri döner.</p><div class="studio-table-wrap"><table id="translation-matrix"><thead><tr><th scope="col">Anahtar</th><th scope="col">Türkçe (TR)</th><th scope="col">English (EN)</th></tr></thead><tbody>`)
	for _, row := range i18n.CatalogRows(scope.TenantID, "tr") {
		searchText := row.Key + " " + row.Default
		for _, cellLocale := range i18n.SupportedCatalogLocales() {
			searchText += " " + localizedRows[cellLocale][row.Key].Translation
		}
		_, _ = io.WriteString(w, `<tr data-translation-row data-search="`+template.HTMLEscapeString(strings.ToLower(searchText))+`"><th scope="row"><code>`+template.HTMLEscapeString(row.Key)+`</code><small>`+template.HTMLEscapeString(row.Default)+`</small></th>`)
		for _, cellLocale := range i18n.SupportedCatalogLocales() {
			localized := localizedRows[cellLocale][row.Key]
			if err := renderTranslationCell(r.Context(), w, row.Key, cellLocale, localized.Translation, localized.Missing, security.CSRFTokenFromContext(r.Context())); err != nil {
				return
			}
		}
		_, _ = io.WriteString(w, `</tr>`)
	}
	_, _ = io.WriteString(w, `</tbody></table></div></main><style nonce="`+template.HTMLEscapeString(security.NonceFromContext(r.Context()))+`">.translation-missing{background:#fef3c7}.translation-missing input{border-color:#d97706}#translation-matrix small{display:block;color:var(--muted);font-weight:400}#translation-matrix form{display:flex;gap:6px;align-items:center}#translation-matrix input:not([type=hidden]){min-width:180px;width:100%}</style><script nonce="`+template.HTMLEscapeString(security.NonceFromContext(r.Context()))+`">(()=>{const input=document.getElementById('translation-search');input?.addEventListener('input',()=>{const query=input.value.trim().toLocaleLowerCase();document.querySelectorAll('[data-translation-row]').forEach(row=>{row.hidden=!row.dataset.search.includes(query)})})})()</script>`)
	_ = ui.AdminEnd(w)
}

func renderTranslationCell(ctx context.Context, output io.Writer, key, locale, value string, missing bool, csrf string) error {
	if _, err := io.WriteString(output, `<td`); err != nil {
		return err
	}
	if missing {
		if _, err := io.WriteString(output, ` class="translation-missing"`); err != nil {
			return err
		}
	}
	id := "translation-" + locale + "-" + strings.ReplaceAll(key, ".", "-")
	if _, err := io.WriteString(output, `><form method="post" action="/admin/translations"><input type="hidden" name="_csrf" value="`+template.HTMLEscapeString(csrf)+`"><input type="hidden" name="key" value="`+template.HTMLEscapeString(key)+`"><input type="hidden" name="locale" value="`+template.HTMLEscapeString(locale)+`">`); err != nil {
		return err
	}
	if err := uicomponents.Input(uicomponents.InputProps{ID: id, Name: "value", Type: "text", Label: locale + " " + key, Value: value, HasMaxLength: true, MaxLength: 1000}).Render(ctx, output); err != nil {
		return err
	}
	if err := uicomponents.Button(uicomponents.ButtonProps{Label: "Kaydet", Type: "submit", Variant: uicomponents.Quiet}).Render(ctx, output); err != nil {
		return err
	}
	_, err := io.WriteString(output, `</form></td>`)
	return err
}

func (h Handler) saveTranslation(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	requestedLocale := strings.TrimSpace(r.FormValue("locale"))
	if !i18n.IsSupportedCatalogLocale(requestedLocale) {
		http.Error(w, "invalid translation locale", http.StatusBadRequest)
		return
	}
	locale := i18n.ResolveCatalogLocale(requestedLocale)
	key, value := strings.TrimSpace(r.FormValue("key")), strings.TrimSpace(r.FormValue("value"))
	if !i18n.IsCatalogKey(key) || len(value) > 1000 || strings.ContainsAny(value, "\x00\r\n") {
		http.Error(w, "invalid translation", http.StatusBadRequest)
		return
	}
	settingKey := i18n.OverrideSettingKey(locale, key)
	if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: settingKey, Value: value, Public: true}); err != nil {
		http.Error(w, "translation could not be saved", siteStatus(err))
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	if !i18n.SetOverride(scope.TenantID, locale, key, value) {
		http.Error(w, "translation could not be saved", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/translations?locale="+url.QueryEscape(locale)+"&saved=1", http.StatusSeeOther)
}

func (h Handler) redirectFallback(w http.ResponseWriter, r *http.Request) {
	if h.Redirects == nil {
		http.NotFound(w, r)
		return
	}
	scope, err := tenant.FromContext(r.Context())
	if err != nil && h.Redirects.RequireTenantScope {
		http.NotFound(w, r)
		return
	}
	rule, ok := h.Redirects.Match(scope, r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	target := rule.TargetPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, rule.Status)
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
	if ui.AdminStartForRequest(w, r, "Yönlendirmeler", "Eski bağlantıları güncel sayfalara yönetin", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
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
	rule := site.RedirectRule{ID: id, Pattern: r.FormValue("pattern"), Kind: r.FormValue("kind"), TargetPath: r.FormValue("target"), CreatedAt: time.Now().UTC()}
	rule.Status, err = strconv.Atoi(r.FormValue("status"))
	if err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}
	if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
		rule.TenantID = scope.TenantID
	}
	if err := h.Service.SaveRedirectRule(r.Context(), rule); err != nil {
		http.Error(w, "redirect could not be saved", siteStatus(err))
		return
	}
	if h.Redirects != nil {
		_ = h.Redirects.Upsert(rule)
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
		rule := site.RedirectRule{ID: id, Pattern: strings.TrimSpace(row[0]), Kind: strings.TrimSpace(row[1]), TargetPath: strings.TrimSpace(row[2]), Status: status, CreatedAt: time.Now().UTC()}
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
	if err := h.Service.SaveRedirectRules(r.Context(), rules); err != nil {
		http.Error(w, "redirect rules could not be imported", siteStatus(err))
		return
	}
	if h.Redirects != nil {
		for _, rule := range rules {
			_ = h.Redirects.Upsert(rule)
		}
	}
	http.Redirect(w, r, "/admin/site/redirect-rules", http.StatusSeeOther)
}

func (h Handler) deleteRedirectRule(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := r.PathValue("id")
	if err := h.Service.DeleteRedirectRule(r.Context(), id); err != nil {
		http.Error(w, "redirect could not be removed", siteStatus(err))
		return
	}
	if h.Redirects != nil {
		tenantID := ""
		if h.Redirects.RequireTenantScope {
			if scope, err := tenant.FromContext(r.Context()); err == nil {
				tenantID = scope.TenantID
			}
		}
		h.Redirects.Remove(tenantID, id)
	}
	http.Redirect(w, r, "/admin/site/redirect-rules", http.StatusSeeOther)
}

func (h Handler) forms(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListForms(r.Context())
	if err != nil {
		http.Error(w, "forms unavailable", http.StatusServiceUnavailable)
		return
	}
	selected := forms.Form{ID: "contact", Title: "İletişim", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "E-posta", Required: true}, {ID: "message", Type: forms.Textarea, Label: "Mesaj", Required: true}}}
	for _, item := range items {
		if item.ID == r.URL.Query().Get("id") || (r.URL.Query().Get("id") == "" && selected.ID == "contact") {
			selected = item
			if r.URL.Query().Get("id") != "" {
				break
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Formlar", "Schema yönetimi", security.NonceFromContext(r.Context()), "site"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><p class="muted">Kayıtlı formlar</p><div class="studio-actions">`)
	for _, item := range items {
		_, _ = fmt.Fprintf(w, `<a class="button button-quiet" href="/admin/site/forms?id=%s">%s</a>`, template.URLQueryEscaper(item.ID), template.HTMLEscapeString(item.Title))
	}
	_, _ = fmt.Fprint(w, `</div></section>`)
	_ = adminstudio.RenderFormBuilder(w, adminstudio.FormBuilderData{Form: selected, Action: "/admin/site/forms", CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context())})
	workingHours := forms.WorkingHours{Timezone: "UTC"}
	if selected.WorkingHours != nil {
		workingHours = *selected.WorkingHours
	}
	_ = ui.WorkingHoursEditor(selected.ID, workingHours.Timezone, workingHours.Notice, security.CSRFTokenFromContext(r.Context()), ui.WorkingHoursDays(workingHours)).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveFormWorkingHours(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || h.Service.Forms == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid working-hours settings", http.StatusBadRequest)
		return
	}
	form, err := h.Service.Forms.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "form unavailable", http.StatusNotFound)
		return
	}
	workingHours := forms.WorkingHours{Timezone: strings.TrimSpace(r.FormValue("timezone")), Notice: strings.TrimSpace(r.FormValue("notice"))}
	for day := range workingHours.Days {
		workingHours.Days[day] = forms.DailyHours{Open: r.FormValue(fmt.Sprintf("open_%d", day)), Close: r.FormValue(fmt.Sprintf("close_%d", day))}
	}
	form.WorkingHours = &workingHours
	if err := h.Service.SaveForm(r.Context(), form); err != nil {
		http.Error(w, "working-hours settings are invalid", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms?id="+url.QueryEscape(form.ID), http.StatusSeeOther)
}

func (h Handler) saveForm(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	var form forms.Form
	if err := json.Unmarshal([]byte(r.FormValue("schema")), &form); err != nil {
		http.Error(w, "invalid form schema", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveForm(r.Context(), form); err != nil {
		http.Error(w, "invalid form schema", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms?id="+url.QueryEscape(form.ID), http.StatusSeeOther)
}

func (h Handler) deleteForm(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Service.DeleteForm(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, "form deletion failed", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/forms", http.StatusSeeOther)
}

func (h Handler) inbox(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 100)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusInternalServerError)
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(items)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		if query != "" && !strings.Contains(strings.ToLower(item.Email+" "+item.Payload+" "+item.ID), query) {
			continue
		}
		rows = append(rows, []string{item.Email, item.Status, item.LeadStage, item.AssignedTo, item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Gelen talepler", "Form gelen kutusu", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<form method="get" action="/admin/site/inbox/%s" class="studio-form">`, template.URLQueryEscaper(r.PathValue("form")))
	_ = uicomponents.Input(uicomponents.InputProps{ID: "inbox-search", Name: "q", Label: "Ara", Value: query, Type: "search", MaxLength: 128, HasMaxLength: true}).Render(r.Context(), w)
	formPath := template.URLQueryEscaper(r.PathValue("form"))
	_, _ = fmt.Fprint(w, `<button class="button button-primary" type="submit">Ara</button></form><p>`)
	_ = designsystem.ExportLinks([]designsystem.ExportLink{{Label: "CSV indir", URL: "/admin/site/inbox/" + formPath + "/export.csv"}, {Label: "Excel indir", URL: "/admin/site/inbox/" + formPath + "/export.xlsx"}}).Render(r.Context(), w)
	_, _ = fmt.Fprint(w, `</p>`)
	_ = uicomponents.Table(uicomponents.TableProps{Caption: "Form talepleri", Columns: []string{"E-posta", "Durum", "Lead aşaması", "Atanan kişi", "Tarih"}, Rows: rows, EmptyMessage: "Eşleşen talep yok."}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) inboxStatus(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil || site.ValidateInboxStatus(r.FormValue("status")) != nil {
		http.Error(w, "invalid inbox status", http.StatusBadRequest)
		return
	}
	if err := h.Service.SetSubmissionStatus(r.Context(), r.PathValue("id"), r.FormValue("status")); err != nil {
		http.Error(w, "inbox update failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) leadUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid lead update", http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateLead(r.Context(), r.PathValue("id"), r.FormValue("lead_status"), r.FormValue("assigned_to")); err != nil {
		http.Error(w, "lead update failed", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) leadNoteCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.admin(r) || h.Service == nil || user == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 9<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid note", http.StatusBadRequest)
		return
	}
	id, err := randomID("note-")
	if err != nil {
		http.Error(w, "notes unavailable", http.StatusServiceUnavailable)
		return
	}
	note := site.LeadNote{ID: id, SubmissionID: r.PathValue("id"), AuthorID: user.ID, Note: r.FormValue("note"), CreatedAt: time.Now().UTC()}
	if err := h.Service.AddLeadNote(r.Context(), note); err != nil {
		http.Error(w, "note could not be saved", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h Handler) leadNotes(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Service.ListLeadNotes(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		http.Error(w, "notes unavailable", siteStatus(err))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}

func (h Handler) exportInboxCSV(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || guards.IsNil(h.Store) || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 1000)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"ID", "Form", "Email", "Created at", "Inbox status", "Lead stage", "Assigned to", "Values"})
	for _, item := range items {
		created := item.CreatedAt.UTC().Format(time.RFC3339)
		_ = writer.Write([]string{csvSafe(item.ID), csvSafe(item.FormKey), csvSafe(item.Email), created, csvSafe(item.Status), csvSafe(item.LeadStage), csvSafe(item.AssignedTo), csvSafe(item.Payload)})
	}
	writer.Flush()
	if err := writer.Error(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return
	}
}

func (h Handler) exportInboxXLSX(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || guards.IsNil(h.Store) || database.RequireExecutor(h.DB) != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.Store.ListSubmissions(r.Context(), h.DB, r.PathValue("form"), 1000)
	if err != nil {
		http.Error(w, "inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	rows := [][]string{{"ID", "Form", "Email", "Created at", "Inbox status", "Lead stage", "Assigned to", "Values"}}
	for _, item := range items {
		rows = append(rows, []string{item.ID, item.FormKey, item.Email, item.CreatedAt.UTC().Format(time.RFC3339), item.Status, item.LeadStage, item.AssignedTo, item.Payload})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="form-inbox.xlsx"`)
	if err := xlsx.Write(w, rows); err != nil {
		return
	}
}

func (h Handler) mailSpool(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.MailSpool == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	items, err := h.MailSpool.List(r.Context(), r.URL.Query().Get("status"), 100)
	if err != nil {
		http.Error(w, "mail queue unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html><html><head><meta charset="utf-8"><title>Mail queue</title></head><body><main><h1>Mail queue</h1><p>Pending messages are retried automatically. Delivery can happen more than once after an unclear server response.</p><table><thead><tr><th>Status</th><th>Recipient</th><th>Subject</th><th>Attempts</th><th>Next attempt</th><th>Action</th></tr></thead><tbody>`)
	csrf := template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))
	for _, item := range items {
		fmt.Fprintf(w, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>`, template.HTMLEscapeString(item.Status), template.HTMLEscapeString(item.Message.To), template.HTMLEscapeString(item.Message.Subject), item.Attempts, template.HTMLEscapeString(item.NextAttemptAt.UTC().Format(time.RFC3339)))
		if item.Status == mailer.SpoolFailedPermanent {
			fmt.Fprintf(w, `<form method="post" action="/admin/site/mail/%s/retry"><input type="hidden" name="_csrf" value="%s"><button type="submit">Retry now</button></form>`, url.PathEscape(item.ID), csrf)
		}
		_, _ = io.WriteString(w, `</td></tr>`)
	}
	_, _ = io.WriteString(w, `</tbody></table></main></body></html>`)
}

func (h Handler) mailSpoolRetry(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.MailSpool == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.MailSpool.RetryNow(r.Context(), r.PathValue("id")); err != nil {
		http.Error(w, "message could not be retried", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/mail", http.StatusSeeOther)
}

func csvSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}

func (h Handler) submitForm(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	formKey := strings.TrimSpace(r.PathValue("form"))
	if _, ok := h.AllowedForms[formKey]; !ok {
		http.Error(w, "form not found", http.StatusNotFound)
		return
	}
	if h.SubmissionLimiter != nil && !h.SubmissionLimiter.Allow(formKey+":"+requestIP(r)) {
		security.WriteRateLimitedResponse(w, "too many form submissions", time.Minute)
		return
	}
	form, err := h.publicForm(r.Context(), formKey)
	if err != nil {
		http.Error(w, "form not found", http.StatusNotFound)
		return
	}
	maxBody := int64(64 << 10)
	hasDocuments := false
	for _, field := range form.Fields {
		if field.Type == forms.Document {
			hasDocuments = true
			maxBody = 21 << 20
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	var parseErr error
	if hasDocuments {
		parseErr = r.ParseMultipartForm(1 << 20)
	} else {
		parseErr = r.ParseForm()
	}
	if parseErr != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if honeypotFilled(r) {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if form.TurnstileEnabled {
		if h.Turnstile == nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		verified, verifyErr := h.Turnstile.Verify(r.Context(), r.PostForm.Get("cf-turnstile-response"), requestIP(r))
		if verifyErr != nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if !verified {
			http.Error(w, "please check your form and try again", http.StatusBadRequest)
			return
		}
	}
	values := make(map[string]string, len(form.Fields))
	for _, field := range form.Fields {
		values[field.ID] = r.PostForm.Get(field.ID)
		if field.Type == forms.Document {
			var files []*multipart.FileHeader
			if r.MultipartForm != nil {
				files = r.MultipartForm.File[field.ID]
			}
			if len(files) > 1 {
				http.Error(w, "invalid form submission", http.StatusBadRequest)
				return
			}
			if len(files) == 1 && strings.TrimSpace(files[0].Filename) != "" {
				values[field.ID] = "pending-document"
			}
		}
	}
	clean, validation, err := form.NormalizeSubmission(values)
	if err != nil || len(validation) != 0 {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	for _, field := range form.Fields {
		if field.Type != forms.Document || values[field.ID] == "" {
			continue
		}
		if h.Media == nil {
			http.Error(w, "form temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.MultipartForm == nil || len(r.MultipartForm.File[field.ID]) != 1 {
			continue
		}
		header := r.MultipartForm.File[field.ID][0]
		file, openErr := header.Open()
		if openErr != nil {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
		asset, uploadErr := h.Media.UploadPrivateDocument(r.Context(), file, header.Filename)
		_ = file.Close()
		if uploadErr != nil {
			status := http.StatusBadRequest
			if errors.Is(uploadErr, media.ErrUploaderRequired) || errors.Is(uploadErr, media.ErrAssetRepositoryRequired) {
				status = http.StatusServiceUnavailable
			}
			http.Error(w, "document upload failed", status)
			return
		}
		values[field.ID] = asset.ID
	}
	if len(form.Fields) > 0 {
		clean, validation, err = form.NormalizeSubmission(values)
		if err != nil || len(validation) != 0 {
			http.Error(w, "invalid form submission", http.StatusBadRequest)
			return
		}
	}
	if source := formSourcePath(r.Referer()); source != "" {
		clean["_source_url"] = source
	}
	email := ""
	for _, field := range form.Fields {
		if field.Type == forms.Email {
			email = clean[field.ID]
			break
		}
	}
	if email == "" {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	payload, err := json.Marshal(clean)
	if err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	digest := sha256.Sum256([]byte(requestIP(r)))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		http.Error(w, "form inbox unavailable", http.StatusServiceUnavailable)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(r.PostForm.Get("_submission_token"))
	}
	if site.ValidateIdempotencyKey(idempotencyKey) != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	requestBytes, _ := json.Marshal(struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}{Form: formKey, Values: clean})
	requestHash := sha256.Sum256(requestBytes)
	submission := site.Submission{ID: "form-" + hex.EncodeToString(idBytes), FormKey: formKey, Email: email, Payload: string(payload), SourceIPHash: hex.EncodeToString(digest[:]), IdempotencyKey: idempotencyKey, RequestHash: hex.EncodeToString(requestHash[:])}
	submission.Notifications = formNotifications(form, clean, submission.ID)
	replayed, err := h.Service.SubmitIdempotent(r.Context(), submission)
	if err != nil {
		http.Error(w, "form submission failed", siteStatus(err))
		return
	}
	if !replayed && h.Realtime != nil {
		data, _ := json.Marshal(map[string]string{"form_key": formKey})
		_ = h.Realtime.Publish(r.Context(), realtime.Event{ID: submission.ID, Topic: "forms:submission_created", Time: time.Now().UTC(), Data: data})
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h Handler) publicForm(ctx context.Context, id string) (forms.Form, error) {
	if h.Service != nil && !guards.IsNil(h.Service.Forms) {
		form, err := h.Service.Forms.Get(ctx, id)
		if err == nil {
			return form, nil
		}
		if id != "contact" {
			return forms.Form{}, err
		}
	}
	if _, ok := h.AllowedForms[id]; !ok || id != "contact" {
		return forms.Form{}, site.ErrInvalidSubmission
	}
	return forms.Form{ID: "contact", Title: "İletişim", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "E-posta", Required: true}, {ID: "name", Type: forms.Text, Label: "Adınız"}, {ID: "message", Type: forms.Textarea, Label: "Mesaj", Required: true}}}, nil
}

func honeypotFilled(r *http.Request) bool {
	for _, field := range []string{"website_url", "hp_token", "_gotcha"} {
		if strings.TrimSpace(r.PostForm.Get(field)) != "" {
			return true
		}
	}
	return false
}

func formNotifications(form forms.Form, values map[string]string, submissionID string) []site.EmailNotification {
	return formNotificationsAt(form, values, submissionID, time.Now().UTC())
}

func formNotificationsAt(form forms.Form, values map[string]string, submissionID string, now time.Time) []site.EmailNotification {
	if len(form.NotificationRecipients) == 0 && !form.Autoresponder {
		return nil
	}
	variables := make(map[string]string, len(values)+1)
	for key, value := range values {
		variables[key] = value
	}
	variables["ticket_code"] = strings.ToUpper(strings.TrimPrefix(submissionID, "form-"))
	var summary strings.Builder
	fmt.Fprintf(&summary, "New submission: %s\n\n", form.Title)
	for _, field := range form.Fields {
		fmt.Fprintf(&summary, "%s: %s\n", field.Label, values[field.ID])
	}
	result := make([]site.EmailNotification, 0, len(form.NotificationRecipients)+1)
	for index, recipient := range form.NotificationRecipients {
		result = append(result, site.EmailNotification{EventKey: fmt.Sprintf("%s:notification:%d", submissionID, index), To: recipient, Subject: "New form submission: " + form.Title, Body: summary.String()})
	}
	if form.Autoresponder {
		email := ""
		for _, field := range form.Fields {
			if field.Type == forms.Email {
				email = values[field.ID]
				break
			}
		}
		subject := form.AutoresponderSubject
		if subject == "" {
			subject = "We received your message"
		}
		body, err := form.AutoresponderBodyFor(now)
		if err != nil {
			return result
		}
		if body == "" {
			body = "Thank you for contacting us. We have received your message. Your reference is {{ticket_code}}."
		}
		subject = expandEmailTemplate(subject, variables)
		body = expandEmailTemplate(body, variables)
		htmlBody, err := mailer.RenderAutoresponderHTML(subject, body)
		if err == nil {
			result = append(result, site.EmailNotification{EventKey: submissionID + ":autoresponder", To: email, Subject: subject, Body: body, HTMLBody: htmlBody})
		}
	}
	return result
}

func expandEmailTemplate(value string, variables map[string]string) string {
	for key, replacement := range variables {
		value = strings.ReplaceAll(value, "{{"+key+"}}", replacement)
	}
	return value
}

func requestIP(r *http.Request) string { return requestmeta.ClientIP(r) }

func formSourcePath(value string) string {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || u.User != nil || u.Path == "" || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(u.Path, "\\\r\n\x00") {
		return ""
	}
	return u.EscapedPath()
}

func (h Handler) saveSetting(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Setting
	if !decodeJSON(w, r, &value) {
		return
	}
	if err := h.Service.SaveSetting(r.Context(), value); err != nil {
		http.Error(w, "invalid site setting", siteStatus(err))
		return
	}
	if h.InvalidateDiscovery != nil {
		h.InvalidateDiscovery()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) appearance(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site appearance unavailable", http.StatusServiceUnavailable)
		return
	}
	value := site.AppearanceFromSettings(settings)
	if ui.AdminStartForRequest(w, r, "Marka ve görünüm", "Renkleri ve yazı tipini sitenizin görünümüne uygulayın", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, field := range []uicomponents.InputProps{
			{ID: "appearance-primary", Name: "primary_color", Type: "color", Label: "Ana renk (açık tema)", Value: value.PrimaryColor, Required: true},
			{ID: "appearance-accent", Name: "accent_color", Type: "color", Label: "Vurgu rengi (açık tema)", Value: value.AccentColor, Required: true},
			{ID: "appearance-dark-primary", Name: "dark_primary_color", Type: "color", Label: "Ana renk (koyu tema)", Value: value.DarkPrimaryColor, Required: true},
			{ID: "appearance-dark-accent", Name: "dark_accent_color", Type: "color", Label: "Vurgu rengi (koyu tema)", Value: value.DarkAccentColor, Required: true},
			{ID: "appearance-logo", Name: "logo_url", Type: "text", Label: "Logo medya yolu (ör. /media/logo.svg)", Value: value.LogoURL},
			{ID: "appearance-dark-logo", Name: "dark_logo_url", Type: "text", Label: "Koyu tema logo medya yolu", Value: value.DarkLogoURL},
			{ID: "appearance-favicon", Name: "favicon_url", Type: "text", Label: "Favicon medya yolu", Value: value.FaviconURL},
			{ID: "appearance-social-image", Name: "social_image_url", Type: "text", Label: "Sosyal paylaşım görseli medya yolu", Value: value.SocialImageURL},
		} {
			if err := uicomponents.Input(field).Render(ctx, output); err != nil {
				return err
			}
			if strings.HasSuffix(field.Name, "_url") {
				if _, err := fmt.Fprintf(output, `<button type="button" class="button button-quiet media-picker-open" data-media-target="%s">Medyadan seç</button>`, template.HTMLEscapeString(field.ID)); err != nil {
					return err
				}
			}
		}
		if err := uicomponents.Select(uicomponents.SelectProps{ID: "appearance-font", Name: "font_family", Label: "Yazı tipi", Value: value.FontFamily, Options: []string{"system", "inter", "lato", "roboto"}}).Render(ctx, output); err != nil {
			return err
		}
		if _, err := io.WriteString(output, `<p id="appearance-contrast" role="status" aria-live="polite">Renk kontrastı denetleniyor…</p>`); err != nil {
			return err
		}
		if _, err := io.WriteString(output, `<section id="appearance-preview" aria-label="Canlı site görünümü önizlemesi" style="padding:24px;border-radius:12px;background:var(--appearance-primary,#334155);color:white;font-family:var(--appearance-font,system-ui)"><p>Site görünümü önizlemesi</p><img id="appearance-preview-logo" alt="Logo önizlemesi" hidden style="max-width:180px;max-height:72px;object-fit:contain"><h2>İşinizi büyütmeye hazır mısınız?</h2><button type="button" style="background:var(--appearance-accent,#0f766e);color:white;border:0;border-radius:6px;padding:10px 16px">İletişime geçin</button><p id="appearance-preview-font">Örnek metin • Aa 123</p></section>`); err != nil {
			return err
		}
		if err := uicomponents.Button(uicomponents.ButtonProps{Label: "Görünümü kaydet", Type: "submit", Variant: uicomponents.Primary}).Render(ctx, output); err != nil {
			return err
		}
		_, err := io.WriteString(output, `<script nonce="`+template.HTMLEscapeString(security.NonceFromContext(ctx))+`">(()=>{const ids=['appearance-primary','appearance-accent','appearance-dark-primary','appearance-dark-accent'];const out=document.getElementById('appearance-contrast');const preview=document.getElementById('appearance-preview');const lum=h=>{const c=h.slice(1).match(/../g).map(x=>parseInt(x,16)/255).map(x=>x<=.04045?x/12.92:Math.pow((x+.055)/1.055,2.4));return .2126*c[0]+.7152*c[1]+.0722*c[2]};const update=()=>{const ratios=ids.map(id=>{const x=document.getElementById(id);return 1.05/(lum(x.value)+.05)});const min=Math.min(...ratios);const low=min<4.5;out.textContent=low?'Kontrast yetersiz: beyaz metin için en az 4,5:1 gerekli.':min>=7?'WCAG AAA uyumlu: en düşük oran '+min.toFixed(2)+':1.':'WCAG AA uyumlu: en düşük oran '+min.toFixed(2)+':1.';out.dataset.state=low?'low':min>=7?'aaa':'aa';preview.style.setProperty('--appearance-primary',document.getElementById(ids[0]).value);preview.style.setProperty('--appearance-accent',document.getElementById(ids[1]).value);preview.style.fontFamily=document.getElementById('appearance-font').value+', system-ui'};ids.forEach(id=>document.getElementById(id)?.addEventListener('input',update));document.getElementById('appearance-font')?.addEventListener('change',update);['appearance-logo','appearance-dark-logo'].forEach(id=>document.getElementById(id)?.addEventListener('input',()=>{const img=document.getElementById('appearance-preview-logo');const path=document.getElementById(id).value;img.src=path;img.hidden=!path}));update()})()</script>`)
		return err
	})
	form := uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/appearance", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: body})
	if err := uicomponents.Card(uicomponents.CardProps{Title: "Renkler ve tipografi", Body: form}).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) saveAppearance(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Görünüm ayarları geçersiz.", http.StatusBadRequest)
		return
	}
	value := site.Appearance{PrimaryColor: r.FormValue("primary_color"), AccentColor: r.FormValue("accent_color"), DarkPrimaryColor: r.FormValue("dark_primary_color"), DarkAccentColor: r.FormValue("dark_accent_color"), FontFamily: r.FormValue("font_family"), LogoURL: r.FormValue("logo_url"), DarkLogoURL: r.FormValue("dark_logo_url"), FaviconURL: r.FormValue("favicon_url"), SocialImageURL: r.FormValue("social_image_url")}
	if err := value.Normalize(); err != nil {
		http.Error(w, "Renkler beyaz metinle en az 4,5:1 kontrast sağlamalı ve desteklenen yazı tiplerinden seçilmelidir.", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveAppearance(r.Context(), value); err != nil {
		http.Error(w, "Görünüm kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/appearance", http.StatusSeeOther)
}

func (h Handler) seoSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	var excluded string
	for _, item := range settings {
		if item.Key == "seo.llms_excluded_paths" {
			excluded = item.Value
			break
		}
	}
	if ui.AdminStartForRequest(w, r, "SEO ayarları", "Yayınlanmış içeriklerin arama ve keşif görünümü", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Textarea(uicomponents.TextareaProps{ID: "llms-excluded-paths", Name: "paths", Label: "LLM metin dosyalarından çıkarılacak yollar (satır başına bir yol)", Value: excluded, Rows: 8, MaxLength: 4096, HasMaxLength: true}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "Ayarları kaydet", Type: "submit", Variant: uicomponents.Primary}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	_ = uicomponents.Card(uicomponents.CardProps{Title: "Yayınlanan içerik", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/seo", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: body})}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveSEOSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid SEO settings", http.StatusBadRequest)
		return
	}
	lines := strings.Split(r.FormValue("paths"), "\n")
	paths := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(path)
		if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(path, "\\\r\n\x00") || len(path) > 512 {
			http.Error(w, "invalid SEO settings", http.StatusBadRequest)
			return
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if len(paths) > 100 {
		http.Error(w, "invalid SEO settings", http.StatusBadRequest)
		return
	}
	value := site.Setting{Key: "seo.llms_excluded_paths", Value: strings.Join(paths, "\n"), Public: true, UpdatedAt: time.Now().UTC()}
	if err := h.Service.SaveSetting(r.Context(), value); err != nil {
		http.Error(w, "SEO settings could not be saved", siteStatus(err))
		return
	}
	if h.InvalidateDiscovery != nil {
		h.InvalidateDiscovery()
	}
	http.Redirect(w, r, "/admin/site/seo", http.StatusSeeOther)
}

var localFontFamilies = []string{"Inter", "Lato", "Montserrat", "Open Sans", "Roboto", "Source Sans 3"}

func (h Handler) fontPicker(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	current := ""
	for _, item := range settings {
		if item.Key == "design.font_css" {
			current = item.Value
			break
		}
	}
	if ui.AdminStartForRequest(w, r, "Yerel yazı tipi", "Kendi siteniz için yerel bir yazı tipi seçin", security.NonceFromContext(r.Context()), "site") != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Select(uicomponents.SelectProps{ID: "font-family", Name: "family", Label: "Yazı tipi", Options: localFontFamilies, Required: true}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "Yazı tipini indir ve uygula", Type: "submit", Variant: uicomponents.Primary}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	if current != "" {
		_, _ = fmt.Fprintf(w, `<p>Etkin yerel yazı tipi: <code>%s</code></p>`, template.HTMLEscapeString(filepath.Base(current)))
	}
	if r.URL.Query().Get("saved") == "1" {
		_, _ = io.WriteString(w, `<p role="status">Yazı tipi indirildi ve siteye uygulandı.</p>`)
	}
	_ = uicomponents.Card(uicomponents.CardProps{Title: "Yazı tipi seç", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/fonts", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: body})}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) localizeFont(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.FontsDir == "" {
		http.Error(w, "Bu özellik için yerel yazı tipi klasörü yapılandırılmalı.", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Yazı tipi seçimi geçersiz.", http.StatusBadRequest)
		return
	}
	family := r.FormValue("family")
	allowed := false
	for _, option := range localFontFamilies {
		if option == family {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "Yazı tipi seçimi geçersiz.", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	cssPath, err := designsystem.LocalizeGoogleFont(ctx, family, h.FontsDir, nil)
	if err != nil {
		http.Error(w, "Yazı tipi indirilemedi. Lütfen daha sonra tekrar deneyin.", http.StatusBadGateway)
		return
	}
	publicPath := "/fonts/" + filepath.Base(cssPath)
	if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: "design.font_css", Value: publicPath, Public: true, UpdatedAt: time.Now().UTC()}); err != nil {
		http.Error(w, "Yazı tipi ayarı kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/fonts?saved=1", http.StatusSeeOther)
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
	if err := h.Service.SaveRedirect(r.Context(), value); err != nil {
		http.Error(w, "invalid redirect", siteStatus(err))
		return
	}
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
		if err := h.Redirects.Upsert(rule); err != nil {
			http.Error(w, "invalid redirect", siteStatus(err))
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveNavigation(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var items []site.NavigationItem
	if !decodeJSON(w, r, &items) {
		return
	}
	if err := h.Service.SaveNavigation(r.Context(), r.PathValue("locale"), items); err != nil {
		http.Error(w, "invalid navigation", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) admin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionSiteManage)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 128<<10); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func siteStatus(err error) int {
	if errors.Is(err, site.ErrIdempotencyConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, site.ErrIdempotencyStoreRequired) || errors.Is(err, site.ErrLeadStoreRequired) || errors.Is(err, site.ErrMailQueueRequired) || errors.Is(err, site.ErrRedirectStoreRequired) || errors.Is(err, mailer.ErrSpoolLeaseInvalid) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, site.ErrStoreRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func (h Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "", false)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	if part := strings.TrimSpace(r.URL.Query().Get("part")); part != "" {
		index, parseErr := strconv.Atoi(part)
		if parseErr != nil || index < 1 || index > len(parts) {
			http.NotFound(w, r)
			return
		}
		writeSitemap(w, parts[index-1])
		return
	}
	if len(parts) > 1 {
		data, err := site.RenderSitemapIndex(h.SitemapOrigin, len(parts))
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		writeSitemapData(w, data)
		return
	}
	writeSitemap(w, entries)
}

func (h Handler) sitemapIndex(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "", true)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	data, err := site.RenderSitemapIndex(h.SitemapOrigin, len(parts))
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	writeSitemapData(w, data)
}

func (h Handler) sitemapPart(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream != nil {
		h.streamSitemap(w, r, "pages", false)
		return
	}
	part, err := strconv.Atoi(r.URL.Query().Get("part"))
	if err != nil || part < 1 {
		http.NotFound(w, r)
		return
	}
	entries, ok := h.sitemapEntries(w, r)
	if !ok {
		return
	}
	parts, err := site.SplitSitemap(entries, site.MaxSitemapURLs, site.MaxSitemapBytes)
	if err != nil || part > len(parts) {
		http.NotFound(w, r)
		return
	}
	writeSitemap(w, parts[part-1])
}

func (h Handler) sitemapArticlePart(w http.ResponseWriter, r *http.Request) {
	if h.SitemapStream == nil {
		http.NotFound(w, r)
		return
	}
	h.streamSitemap(w, r, "articles", false)
}

func (h Handler) streamSitemap(w http.ResponseWriter, r *http.Request, category string, forceIndex bool) {
	if h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return
	}
	if category == "" && strings.TrimSpace(r.URL.Query().Get("part")) != "" {
		category = "pages"
	}
	categories := h.SitemapCategories
	if len(categories) == 0 {
		categories = []string{"pages"}
	}
	if category != "" {
		part := 1
		if raw := strings.TrimSpace(r.URL.Query().Get("part")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				http.NotFound(w, r)
				return
			}
			part = parsed
		}
		var output bytes.Buffer
		count, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, category, part, site.MaxSitemapURLs, site.MaxSitemapBytes, &output)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		if part > count || count == 0 {
			http.NotFound(w, r)
			return
		}
		writeSitemapData(w, output.Bytes())
		return
	}
	counts := make(map[string]int, len(categories))
	totalParts := 0
	for _, item := range categories {
		count, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, item, 0, site.MaxSitemapURLs, site.MaxSitemapBytes, nil)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			counts[item] = count
			totalParts += count
		}
	}
	if totalParts == 0 {
		writeSitemap(w, nil)
		return
	}
	if forceIndex || totalParts > 1 {
		data, err := site.RenderCategorizedSitemapIndex(h.SitemapOrigin, counts)
		if err != nil {
			http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
			return
		}
		writeSitemapData(w, data)
		return
	}
	for item, count := range counts {
		if count == 1 {
			var output bytes.Buffer
			_, err := site.StreamSitemapPart(r.Context(), h.SitemapStream, item, 1, site.MaxSitemapURLs, site.MaxSitemapBytes, &output)
			if err != nil {
				http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
				return
			}
			writeSitemapData(w, output.Bytes())
			return
		}
	}
}

func (h Handler) sitemapEntries(w http.ResponseWriter, r *http.Request) ([]site.SitemapEntry, bool) {
	if h.SitemapProvider == nil || h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return nil, false
	}
	entries, err := h.SitemapProvider(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return nil, false
	}
	return entries, true
}

func writeSitemap(w http.ResponseWriter, entries []site.SitemapEntry) {
	data, err := site.RenderSitemap(entries)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	writeSitemapData(w, data)
}

func writeSitemapData(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=300")
	_, _ = w.Write(data)
}

func (h Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "site settings unavailable", http.StatusServiceUnavailable)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(result)
}

func (h Handler) navigation(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "navigation unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListNavigation(r.Context(), h.DB, r.PathValue("locale"))
	if err != nil {
		http.Error(w, "navigation unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}
