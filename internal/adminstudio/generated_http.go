package adminstudio

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

const generatedAdminBodyLimit = 1 << 20

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8})*$`)
var draftIDPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._:-]{0,127}$`)

// GeneratedAdminOptions contains the Edition-owned security and page-shell
// boundaries. Nil callbacks make the generated Admin surface unavailable.
type GeneratedAdminOptions struct {
	BlockEditor     BlockEditorConfig
	API             http.Handler
	Authorize       func(*http.Request) bool
	CanReadResource func(*http.Request, Collection) bool
	// ResourceAccess optionally controls the generated Admin's basic CRUD
	// affordances and routes. The resource API remains the authorization source.
	ResourceAccess  func(*http.Request, Collection, string) ResourceAccess
	CanTransition   func(*http.Request, Collection, map[string]any, PublicationState) bool
	Transition      func(*http.Request, Collection, map[string]any, PublicationState, int64) error
	ListRevisions   func(*http.Request, Collection, string, int) ([]RevisionSummary, error)
	GetRevision     func(*http.Request, Collection, string, int64) (RevisionSnapshot, error)
	CanRestore      func(*http.Request, Collection, string, int64) bool
	RestoreRevision func(*http.Request, Collection, string, int64, int64) error
	// Locales returns only languages the current Edition user may access.
	Locales func(*http.Request, Collection) (LocaleOptions, error)
	// Locale storage callbacks must apply tenant scope and expected-version CAS.
	GetLocalized  func(*http.Request, Collection, string, string) (map[string]any, int64, error)
	SaveLocalized func(*http.Request, Collection, string, string, map[string]any, int64) error
	ListLocalized func(*http.Request, Collection, []string, string) (map[string]map[string]any, error)
	// CopyLocale must revalidate the source and target and audit the copy.
	CanCopyLocale func(*http.Request, Collection, string, string) bool
	CopyLocale    func(*http.Request, Collection, string, string, string, int64) error
	ListDrafts    func(*http.Request, Collection, string, int) ([]DraftDocument, error)
	// ListDraftPage returns a stable, tenant-scoped draft page for the supplied
	// offset and bounded limit. It enables pagination while ListDrafts remains
	// the source-compatible first-page fallback.
	ListDraftPage      func(*http.Request, Collection, string, int, int) ([]DraftDocument, error)
	GetDraft           func(*http.Request, Collection, string, string) (DraftDocument, error)
	CanSaveDraft       func(*http.Request, Collection, string, string) bool
	SaveDraft          func(*http.Request, Collection, string, string, map[string]any, int64, int64) error
	CanTransitionDraft func(*http.Request, Collection, DraftDocument, PublicationState) bool
	TransitionDraft    func(*http.Request, Collection, string, string, PublicationState, int64, int64) error
	CanDiscardDraft    func(*http.Request, Collection, DraftDocument) bool
	DiscardDraft       func(*http.Request, Collection, string, string, int64) error
	CSRFToken          func(*http.Request) string
	ValidateCSRF       func(*http.Request) bool
	RenderPage         func(http.ResponseWriter, *http.Request, string, templ.Component) error
}

// ResourceAccess describes basic generated CRUD capabilities for a resource
// or record. An empty record ID asks for collection-level create access.
type ResourceAccess struct {
	Create bool
	Update bool
	Delete bool
}

// BlockEditorConfig binds the Edition-owned Editor.js asset. PreviewURL is
// optional; when set, the endpoint must enforce CSRF and render only validated,
// sanitized block content.
type BlockEditorConfig struct {
	EditorScript string
	PreviewURL   string
}

type LocaleOptions struct {
	Default string
	Allowed []string
}

type localeContextKey struct{}

type selectedLocale struct {
	Options LocaleOptions
	Current string
}

type RevisionSummary struct {
	Version   int64
	State     string
	Actor     string
	CreatedAt time.Time
}

type RevisionSnapshot struct {
	RevisionSummary
	Fields map[string]string
}

// DraftDocument is a working copy. PublishedVersion is checked independently
// from the draft version so a concurrent publish cannot be silently replaced.
type DraftDocument struct {
	ID               string
	Version          int64
	PublishedVersion int64
	State            PublicationState
	UpdatedAt        time.Time
	Fields           map[string]any
	Exists           bool
}

// GeneratedAdminHandler renders the manifest's generated resources and sends
// every read/write through the Edition's already-composed API handlers.
type GeneratedAdminHandler struct {
	Registry Registry
	Options  GeneratedAdminOptions
}

func (h GeneratedAdminHandler) Register(mux *http.ServeMux) error {
	if mux == nil || h.Options.API == nil || h.Options.Authorize == nil || h.Options.CanReadResource == nil || h.Options.CSRFToken == nil || h.Options.ValidateCSRF == nil || h.Options.RenderPage == nil || len(h.Registry.Collections()) == 0 {
		return errors.New("generated Admin registration requires registry, API, authorization, CSRF and page renderer")
	}
	for _, collection := range h.Registry.Collections() {
		for _, field := range collection.Fields {
			if field.Kind != FieldBlocks && field.Widget != "block-editor" {
				continue
			}
			config := h.Options.BlockEditor
			if !validEditionAssetPath(config.EditorScript) {
				return errors.New("generated Admin block editor requires an Edition asset")
			}
			if config.PreviewURL != "" && !validEditionAssetPath(config.PreviewURL) {
				return errors.New("generated Admin block preview must use a same-origin path")
			}
		}
		if collection.Workflow != nil && !collection.Drafts && (h.Options.CanTransition == nil || h.Options.Transition == nil) {
			return errors.New("generated Admin workflow requires Edition-owned transition policy and service")
		}
		if collection.Revisions && (h.Options.ListRevisions == nil || h.Options.GetRevision == nil || h.Options.CanRestore == nil || h.Options.RestoreRevision == nil) {
			return errors.New("generated Admin revisions require Edition-owned history and restore services")
		}
		if collection.Localization && (h.Options.Locales == nil || h.Options.GetLocalized == nil || h.Options.SaveLocalized == nil || h.Options.ListLocalized == nil || h.Options.CanCopyLocale == nil || h.Options.CopyLocale == nil) {
			return errors.New("generated Admin localization requires Edition-owned locale access and storage services")
		}
		if collection.Drafts && (h.Options.ListDrafts == nil && h.Options.ListDraftPage == nil || h.Options.GetDraft == nil || h.Options.CanSaveDraft == nil || h.Options.SaveDraft == nil || h.Options.CanTransitionDraft == nil || h.Options.TransitionDraft == nil || h.Options.CanDiscardDraft == nil || h.Options.DiscardDraft == nil) {
			return errors.New("generated Admin drafts require Edition-owned draft, publish, and discard services")
		}
	}
	mux.Handle("/admin/resources", h)
	mux.Handle("/admin/resources/", h)
	return nil
}

func validEditionAssetPath(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.IsAbs() == false && parsed.Host == "" && strings.HasPrefix(parsed.Path, "/") && !strings.HasPrefix(parsed.Path, "//")
}

func (h GeneratedAdminHandler) resourceAccess(r *http.Request, collection Collection, id string) ResourceAccess {
	if h.Options.ResourceAccess == nil {
		return ResourceAccess{Create: true, Update: true, Delete: true}
	}
	return h.Options.ResourceAccess(r, collection, id)
}

func (h GeneratedAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Options.Authorize == nil || !h.Options.Authorize(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/resources"), "/")
	if path == "" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.index(w, r)
		return
	}
	parts := strings.Split(path, "/")
	collection, ok := h.Registry.Collection(parts[0])
	if !ok || collection.Hidden || !h.Options.CanReadResource(r, collection) || len(parts) > 5 {
		http.NotFound(w, r)
		return
	}
	if collection.Localization {
		locale, err := h.localeSelection(r, collection)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), localeContextKey{}, locale))
	} else if len(r.URL.Query()["locale"]) != 0 {
		http.Error(w, "invalid locale selection", http.StatusBadRequest)
		return
	}
	if len(parts) == 4 && parts[2] == "revisions" && r.Method == http.MethodGet {
		h.revision(w, r, collection, parts[1], parts[3])
		return
	}
	if len(parts) == 5 && parts[2] == "revisions" && parts[4] == "restore" && r.Method == http.MethodPost {
		h.restoreRevision(w, r, collection, parts[1], parts[3])
		return
	}
	if len(parts) == 2 && parts[1] == "drafts" && r.Method == http.MethodGet {
		h.draftList(w, r, collection)
		return
	}
	if len(parts) == 3 && parts[2] == "locale-copy" && r.Method == http.MethodPost {
		h.copyLocale(w, r, collection, parts[1])
		return
	}
	if len(parts) == 3 && parts[2] == "locale" && r.Method == http.MethodPost {
		if !collection.Drafts && !h.resourceAccess(r, collection, parts[1]).Update {
			http.Error(w, "Bu kaydı düzenleme yetkiniz yok.", http.StatusForbidden)
			return
		}
		h.mutateLocale(w, r, collection, parts[1])
		return
	}
	if len(parts) == 3 && parts[2] == "draft" && r.Method == http.MethodPost {
		h.mutateDraft(w, r, collection, parts[1], false)
		return
	}
	if len(parts) == 4 && parts[2] == "draft" && parts[3] == "discard" && r.Method == http.MethodPost {
		h.discardDraft(w, r, collection, parts[1])
		return
	}
	if len(parts) == 2 && parts[1] == "new" {
		if !collection.Drafts && !h.resourceAccess(r, collection, "").Create {
			http.Error(w, "Bu kaynakta kayıt oluşturma yetkiniz yok.", http.StatusForbidden)
			return
		}
		if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.renderForm(w, r, collection, "", nil, false)
		case http.MethodPost:
			if collection.Drafts {
				h.mutateDraft(w, r, collection, "", true)
			} else {
				h.mutate(w, r, collection, "", false)
			}
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 3 && parts[1] == "new" && parts[2] == "draft" && r.Method == http.MethodPost {
		h.mutateDraft(w, r, collection, "", true)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.list(w, r, collection)
		return
	}
	if parts[1] == "" || (len(parts) == 3 && parts[2] != "delete" && parts[2] != "workflow" && parts[2] != "locale-copy" && parts[2] != "locale" && parts[2] != "draft") || (len(parts) > 3 && parts[2] != "revisions") {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet && len(parts) == 2 {
		h.edit(w, r, collection, parts[1])
		return
	}
	if r.Method == http.MethodPost {
		if len(parts) == 3 && parts[2] == "workflow" {
			h.transition(w, r, collection, parts[1])
			return
		}
		deleting := len(parts) == 3
		access := h.resourceAccess(r, collection, parts[1])
		if deleting && !access.Delete || !deleting && !access.Update {
			http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
			return
		}
		h.mutate(w, r, collection, parts[1], deleting)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h GeneratedAdminHandler) localeSelection(r *http.Request, collection Collection) (selectedLocale, error) {
	options, err := h.Options.Locales(r, collection)
	if err != nil || len(options.Allowed) == 0 || len(options.Allowed) > 32 || !localePattern.MatchString(options.Default) {
		return selectedLocale{}, errors.New("locale policy unavailable")
	}
	allowed := make(map[string]struct{}, len(options.Allowed))
	defaultFound := false
	for _, locale := range options.Allowed {
		if !localePattern.MatchString(locale) {
			return selectedLocale{}, errors.New("invalid Edition locale")
		}
		if _, exists := allowed[locale]; exists {
			return selectedLocale{}, errors.New("duplicate Edition locale")
		}
		allowed[locale] = struct{}{}
		defaultFound = defaultFound || locale == options.Default
	}
	if !defaultFound || len(r.URL.Query()["locale"]) > 1 {
		return selectedLocale{}, errors.New("invalid locale selection")
	}
	current := r.URL.Query().Get("locale")
	if current == "" {
		current = options.Default
	}
	if _, ok := allowed[current]; !ok {
		return selectedLocale{}, errors.New("locale is not available")
	}
	options.Allowed = append([]string(nil), options.Allowed...)
	return selectedLocale{Options: options, Current: current}, nil
}

func localeState(r *http.Request) selectedLocale {
	state, _ := r.Context().Value(localeContextKey{}).(selectedLocale)
	return state
}

func localizedTarget(target, locale string) string {
	if locale == "" {
		return target
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return target
	}
	query := parsed.Query()
	query.Set("locale", locale)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func localePath(path, locale string) string {
	return localizedTarget(path, locale)
}

func applyLocalizedValues(collection Collection, record, values map[string]any) error {
	encoded, err := json.Marshal(values)
	if err != nil || len(encoded) > generatedAdminBodyLimit {
		return errors.New("localized payload exceeds the resource limit")
	}
	allowed := make(map[string]struct{})
	for _, field := range collection.Fields {
		if field.Localized {
			allowed[field.ID] = struct{}{}
		}
	}
	if len(values) > len(allowed) {
		return errors.New("localized field count exceeds the resource contract")
	}
	for field, value := range values {
		if _, ok := allowed[field]; !ok {
			return errors.New("localized value is outside the resource contract")
		}
		record[field] = value
	}
	return nil
}

func (h GeneratedAdminHandler) currentResource(r *http.Request, collection Collection, id string) (map[string]any, error) {
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	if err != nil || status != http.StatusOK {
		return nil, errors.New("resource is unavailable")
	}
	var record map[string]any
	if json.Unmarshal(response, &record) != nil {
		return nil, errors.New("resource response is invalid")
	}
	state := localeState(r)
	if collection.Localization && state.Current != state.Options.Default {
		values, version, err := h.Options.GetLocalized(r, collection, id, state.Current)
		if err != nil || version < 0 || applyLocalizedValues(collection, record, values) != nil {
			return nil, errors.New("localized resource is unavailable")
		}
		record["locale_version"] = version
	}
	if collection.Drafts {
		draft, err := h.Options.GetDraft(r, collection, id, state.Current)
		if err != nil || draft.ID != "" && draft.ID != id || draft.Version < 0 || draft.PublishedVersion < 0 || draft.PublishedVersion > recordVersion(record["version"]) || draft.Exists && draft.Version < 1 || !draft.Exists && (draft.Version != 0 || len(draft.Fields) != 0) {
			return nil, errors.New("draft is unavailable")
		}
		if draft.Exists && applyDraftFields(collection, record, draft.Fields, state.Current, state.Options.Default) != nil {
			return nil, errors.New("draft fields are invalid")
		}
		if draft.Exists {
			if !slices.Contains(collection.Workflow.States, draft.State) {
				return nil, errors.New("draft state is invalid")
			}
			record[collection.Workflow.StateField] = string(draft.State)
			record["draft_version"] = draft.Version
		}
		record["published_version"] = draft.PublishedVersion
		record["_draft"] = draft
	}
	return record, nil
}

func (h GeneratedAdminHandler) generatedLocaleNav(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string) error {
	if !collection.Localization {
		return nil
	}
	state := localeState(r)
	if _, err := fmt.Fprintf(out, `<nav class="locale-tabs" aria-label="%s">`, template.HTMLEscapeString(i18n.T(generatedAdminLocale(r), "admin.generated.locale.navigation"))); err != nil {
		return err
	}
	for _, locale := range state.Options.Allowed {
		label := strings.ToUpper(locale)
		if locale == state.Current {
			if _, err := io.WriteString(out, `<span aria-current="page">`+template.HTMLEscapeString(label)+`</span> `); err != nil {
				return err
			}
			continue
		}
		body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: locale}).Render(ctx, form); err != nil {
				return err
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Quiet}).Render(ctx, form)
		})
		if err := ui.Form(ui.FormProps{Action: r.URL.Path, Method: http.MethodGet, Body: body}).Render(ctx, out); err != nil {
			return err
		}
		if _, err := io.WriteString(out, " "); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</nav>`)
	return err
}

func (h GeneratedAdminHandler) generatedCopyLocaleActions(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any) error {
	locale := generatedAdminLocale(r)
	if _, err := fmt.Fprintf(out, `<section class="studio-card" aria-labelledby="locale-copy-heading"><h2 id="locale-copy-heading">%s</h2>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.locale.copy_heading"))); err != nil {
		return err
	}
	expected := generatedRecordVersion(record)
	if collection.Localization {
		expected = recordVersion(record["locale_version"])
	}
	for _, from := range localeState(r).Options.Allowed {
		if from == localeState(r).Current || !h.Options.CanCopyLocale(r, collection, id, from) {
			continue
		}
		body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "from_locale", Value: from}).Render(ctx, form); err != nil {
				return err
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(expected, 10)}).Render(ctx, form); err != nil {
				return err
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: fmt.Sprintf(i18n.T(locale, "admin.generated.locale.copy_action"), strings.ToUpper(from)), Variant: ui.Quiet}).Render(ctx, form)
		})
		action := localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id)+"/locale-copy", localeState(r).Current)
		if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: body}).Render(ctx, out); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</section>`)
	return err
}

func (h GeneratedAdminHandler) index(w http.ResponseWriter, r *http.Request) {
	collections := h.visibleCollections(r)
	locale := generatedAdminLocale(r)
	title, heading := "Kaynaklar", "Koleksiyonlar"
	if locale == "en" {
		title, heading = "Generated resources", "Collections"
	}
	content := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<section class="studio-card"><h2>`+template.HTMLEscapeString(heading)+`</h2><div class="collection-grid">`); err != nil {
			return err
		}
		for _, collection := range collections {
			if err := ui.Link(ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: collection.Label, Class: "collection-tile"}).Render(ctx, out); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</div></section>`)
		return err
	})
	h.renderPage(w, r, title, content)
}

func (h GeneratedAdminHandler) visibleCollections(r *http.Request) []Collection {
	all := h.Registry.Collections()
	visible := make([]Collection, 0, len(all))
	for _, collection := range all {
		if !collection.Hidden && h.Options.CanReadResource(r, collection) {
			visible = append(visible, collection)
		}
	}
	return visible
}

func (h GeneratedAdminHandler) list(w http.ResponseWriter, r *http.Request, collection Collection) {
	if collection.Type == "global" {
		if !h.resourceAccess(r, collection, "global").Update {
			http.Error(w, "Bu ayarı düzenleme yetkiniz yok.", http.StatusForbidden)
			return
		}
		h.edit(w, r, collection, "global")
		return
	}
	query, err := parseGeneratedListQuery(r.URL.Query())
	if err != nil {
		http.Error(w, "invalid list query", http.StatusBadRequest)
		return
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	if page > 100 {
		http.Error(w, "page is out of range", http.StatusBadRequest)
		return
	}
	queryValues, err := generatedResourceQuery(collection, query, page)
	if err != nil {
		http.Error(w, "invalid list query", http.StatusBadRequest)
		return
	}
	queryValues.Set("limit", strconv.Itoa(DefaultPageSize+1))
	queryValues.Set("offset", strconv.Itoa((page-1)*DefaultPageSize))
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"?"+queryValues.Encode(), nil)
	if err != nil {
		h.apiError(w, r)
		return
	}
	if status == http.StatusForbidden || status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if status != http.StatusOK {
		h.apiError(w, r)
		return
	}
	var records []map[string]any
	if json.Unmarshal(response, &records) != nil {
		h.apiError(w, r)
		return
	}
	hasNext := len(records) > DefaultPageSize
	if hasNext {
		records = records[:DefaultPageSize]
	}
	localizedRecords := map[string]map[string]any(nil)
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		ids := make([]string, 0, len(records))
		for _, record := range records {
			if id, _ := record["id"].(string); id != "" {
				ids = append(ids, id)
			}
		}
		var err error
		localizedRecords, err = h.Options.ListLocalized(r, collection, ids, localeState(r).Current)
		if err != nil || len(localizedRecords) > len(ids) {
			h.apiError(w, r)
			return
		}
		known := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			known[id] = struct{}{}
		}
		for id := range localizedRecords {
			if _, ok := known[id]; !ok {
				h.apiError(w, r)
				return
			}
		}
	}
	rows := make([]Row, 0, len(records))
	uiLocale := generatedAdminLocale(r)
	for _, record := range records {
		id, _ := record["id"].(string)
		if translated, ok := localizedRecords[id]; ok && applyLocalizedValues(collection, record, translated) != nil {
			h.apiError(w, r)
			return
		}
		values := make(map[string]string, len(collection.Fields))
		for _, field := range collection.Fields {
			values[field.ID] = displayValue(record[field.ID])
		}
		editURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
		if collection.Localization {
			editURL = localePath(editURL, localeState(r).Current)
		}
		row := Row{ID: id, Values: values, Status: StatusInfo{Value: values["status"], Label: generatedStatusLabel(uiLocale, values["status"])}}
		row.DetailURL = editURL
		canUpdate := collection.Drafts || h.resourceAccess(r, collection, id).Update
		if canUpdate {
			row.Actions = []Action{{Label: i18n.T(uiLocale, "admin.generated.list.edit"), Href: editURL}}
		}
		rows = append(rows, row)
	}
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		newURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/new"
		if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
			newURL = ""
		}
		if _, err := io.WriteString(out, `<section class="resource-page"><header class="dashboard-card-head"><div><p class="muted">`+template.HTMLEscapeString(collection.Description)+`</p></div>`); err != nil {
			return err
		}
		canCreate := collection.Drafts || h.resourceAccess(r, collection, "").Create
		if newURL != "" && canCreate {
			if err := ui.Link(ui.LinkProps{Href: newURL, Label: i18n.T(uiLocale, "admin.generated.list.new_record"), Class: "button"}).Render(ctx, out); err != nil {
				return err
			}
		}
		if collection.Drafts {
			draftsPath := "/admin/resources/" + url.PathEscape(collection.ID) + "/drafts"
			if collection.Localization {
				draftsBody := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
					if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: localeState(r).Current}).Render(ctx, form); err != nil {
						return err
					}
					return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(uiLocale, "admin.generated.drafts.title"), Variant: ui.Quiet}).Render(ctx, form)
				})
				if err := ui.Form(ui.FormProps{Action: draftsPath, Method: http.MethodGet, Body: draftsBody}).Render(ctx, out); err != nil {
					return err
				}
			} else if err := ui.Link(ui.LinkProps{Href: draftsPath, Label: i18n.T(uiLocale, "admin.generated.drafts.title"), Class: "button button-quiet"}).Render(ctx, out); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</header>`); err != nil {
			return err
		}
		if err := h.generatedLocaleNav(context.Background(), out, r, collection, ""); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `<section class="studio-card">`); err != nil {
			return err
		}
		if err := generatedListControls(out, collection, query, localeState(r).Current, uiLocale); err != nil {
			return err
		}
		if err := RenderList(out, ListData{
			Collection: collection, Rows: rows, BaseURL: "/admin/resources/" + url.PathEscape(collection.ID),
			Locale: localeState(r).Current, UILocale: uiLocale, Query: query,
			ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
			ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
		}); err != nil {
			return err
		}
		if page > 1 {
			previous := query
			previous.Page = page - 1
			if err := generatedListPageForm(context.Background(), out, collection.ID, previous, localeState(r).Current, i18n.T(uiLocale, "admin.generated.list.previous")); err != nil {
				return err
			}
		}
		if hasNext {
			next := query
			next.Page = page + 1
			if err := generatedListPageForm(context.Background(), out, collection.ID, next, localeState(r).Current, i18n.T(uiLocale, "admin.generated.list.next")); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</section>`); err != nil {
			return err
		}
		_, err := io.WriteString(out, `</section>`)
		return err
	})
	h.renderPage(w, r, collection.Label, body)
}

func generatedResourceQuery(collection Collection, query ListQuery, page int) (url.Values, error) {
	values := url.Values{}
	searchFields := make([]Field, 0)
	filterFields := make(map[string]Field)
	sortFields := make(map[string]struct{})
	for _, field := range collection.Fields {
		if field.Searchable && !field.Localized && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
			searchFields = append(searchFields, field)
		}
		if field.Filterable && !field.Localized {
			filterFields[field.ID] = field
		}
		if field.VisibleInList && !field.Localized {
			sortFields[field.ID] = struct{}{}
		}
	}
	if len(query.Search) > 256 {
		return nil, errors.New("search query too long")
	}
	if query.Search != "" {
		if len(searchFields) == 0 {
			return nil, errors.New("search is not enabled")
		}
		if len(searchFields) > 32 {
			searchFields = searchFields[:32]
		}
		for index, field := range searchFields {
			values.Set(fmt.Sprintf("where[or][%d][%s][contains]", index, field.ID), query.Search)
		}
	}
	for id, value := range query.Filters {
		field, ok := filterFields[id]
		if !ok || len(value) > 256 {
			return nil, errors.New("filter is not allowed")
		}
		if len(field.Options) > 0 {
			allowed := false
			for _, option := range field.Options {
				if option == value {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, errors.New("filter value is not allowed")
			}
		}
		operator := "equals"
		if len(field.Options) == 0 && (field.Kind == FieldText || field.Kind == FieldRichText) {
			operator = "contains"
		}
		values.Set(fmt.Sprintf("where[%s][%s]", field.ID, operator), value)
	}
	if query.Sort != "" {
		field := strings.TrimPrefix(query.Sort, "-")
		if _, ok := sortFields[field]; !ok {
			return nil, errors.New("sort field is not allowed")
		}
		values.Set("sort", query.Sort)
	}
	return values, nil
}

func parseGeneratedListQuery(values url.Values) (ListQuery, error) {
	if len(values) > 34 {
		return ListQuery{}, errors.New("too many list query parameters")
	}
	queryBytes := 0
	for key, raw := range values {
		queryBytes += len(key)
		if len(raw) != 1 {
			return ListQuery{}, errors.New("duplicate list query parameter")
		}
		queryBytes += len(raw[0])
		if queryBytes > 8192 {
			return ListQuery{}, errors.New("list query exceeds limit")
		}
		filterKey := strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") && len(key) > len("filter[]")
		if key != "search" && key != "sort" && key != "page" && key != "locale" && !filterKey {
			return ListQuery{}, errors.New("unsupported list query parameter")
		}
	}
	query := ParseListQuery(values)
	if len(query.Search) > 256 {
		return ListQuery{}, errors.New("search query too long")
	}
	if rawPage := values.Get("page"); rawPage != "" {
		page, err := strconv.Atoi(rawPage)
		if err != nil || page < 1 || page > 100 {
			return ListQuery{}, errors.New("page is out of range")
		}
		query.Page = page
	}
	return query, nil
}

func generatedListPageForm(ctx context.Context, out io.Writer, id string, query ListQuery, locale, label string) error {
	return generatedListPageFormAt(ctx, out, "/admin/resources/"+url.PathEscape(id), query, locale, label)
}

func generatedListPageFormAt(ctx context.Context, out io.Writer, action string, query ListQuery, locale, label string) error {
	body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
		values := url.Values{}
		if query.Search != "" {
			values.Set("search", query.Search)
		}
		if query.Sort != "" {
			values.Set("sort", query.Sort)
		}
		if query.Page > 0 {
			values.Set("page", strconv.Itoa(query.Page))
		}
		if locale != "" {
			values.Set("locale", locale)
		}
		for field, value := range query.Filters {
			values.Set("filter["+field+"]", value)
		}
		keys := make([]string, 0, len(values))
		for name := range values {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			entries := values[name]
			if len(entries) != 1 {
				return errors.New("invalid list pagination query")
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: name, Value: entries[0]}).Render(ctx, form); err != nil {
				return err
			}
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Quiet}).Render(ctx, form)
	})
	return ui.Form(ui.FormProps{Action: action, Method: http.MethodGet, Body: body}).Render(ctx, out)
}

func generatedListControls(w io.Writer, collection Collection, query ListQuery, locale, uiLocale string) error {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<form class="admin-list-toolbar studio-form" method="get" action="/admin/resources/`+url.PathEscape(collection.ID)+`">`); err != nil {
			return err
		}
		if query.Sort != "" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "sort", Value: query.Sort}).Render(ctx, out); err != nil {
				return err
			}
		}
		if collection.Localization && locale != "" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: locale}).Render(ctx, out); err != nil {
				return err
			}
		}
		searchFields := 0
		for _, field := range collection.Fields {
			if field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
				searchFields++
			}
		}
		if searchFields > 0 {
			if err := ui.Input(ui.InputProps{ID: "resource-search", Name: "search", Label: i18n.T(uiLocale, "admin.generated.list.search"), Type: "search", Value: query.Search}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if !field.Filterable {
				continue
			}
			options := append([]string(nil), field.Options...)
			if len(options) > 0 {
				allLabel := i18n.T(uiLocale, "admin.generated.list.all")
				items := make([]ui.SelectOption, 0, len(options)+1)
				items = append(items, ui.SelectOption{Value: "", Label: allLabel})
				for _, option := range options {
					items = append(items, ui.SelectOption{Value: option, Label: option})
				}
				if err := ui.Select(ui.SelectProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID], Items: items}).Render(ctx, out); err != nil {
					return err
				}
			} else if err := ui.Input(ui.InputProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `<div class="studio-actions">`); err != nil {
			return err
		}
		if err := ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(uiLocale, "admin.generated.list.apply"), Variant: ui.Primary}).Render(ctx, out); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `</div>`); err != nil {
			return err
		}
		_, err := io.WriteString(out, `</form>`)
		return err
	}).Render(context.Background(), w)
}

func (h GeneratedAdminHandler) edit(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if id == "global" && collection.Type != "global" {
		http.NotFound(w, r)
		return
	}
	apiPath := "/api/v1/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	response, status, err := h.callAPI(r, http.MethodGet, apiPath, nil)
	if err != nil {
		h.apiError(w, r)
		return
	}
	if status == http.StatusForbidden || !collection.Drafts && status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if status != http.StatusOK && !(collection.Drafts && status == http.StatusNotFound) {
		h.apiError(w, r)
		return
	}
	record := map[string]any{"id": id}
	publishedVersion := int64(0)
	if status == http.StatusOK {
		if json.Unmarshal(response, &record) != nil {
			h.apiError(w, r)
			return
		}
		publishedVersion = generatedRecordVersion(record)
	}
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		values, version, err := h.Options.GetLocalized(r, collection, id, localeState(r).Current)
		if err != nil || version < 0 || applyLocalizedValues(collection, record, values) != nil {
			h.apiError(w, r)
			return
		}
		record["locale_version"] = version
	}
	if !collection.Drafts && !h.resourceAccess(r, collection, id).Update {
		h.renderReadOnlyRecord(w, r, collection, record)
		return
	}
	if collection.Drafts {
		draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
		if err != nil || draft.ID != "" && draft.ID != id || draft.Version < 0 || draft.PublishedVersion < 0 || draft.PublishedVersion > publishedVersion || draft.Exists && draft.Version < 1 || !draft.Exists && (draft.Version != 0 || len(draft.Fields) != 0) {
			h.apiError(w, r)
			return
		}
		if draft.Exists && applyDraftFields(collection, record, draft.Fields, localeState(r).Current, localeState(r).Options.Default) != nil {
			h.apiError(w, r)
			return
		}
		state := draft.State
		if state == "" {
			state = collection.Workflow.States[0]
		}
		if !slices.Contains(collection.Workflow.States, state) {
			h.apiError(w, r)
			return
		}
		record[collection.Workflow.StateField] = string(state)
		record["draft_state"] = string(state)
		record["draft_version"] = draft.Version
		record["published_version"] = draft.PublishedVersion
		record["draft_exists"] = draft.Exists
		record["draft_updated_at"] = draft.UpdatedAt
		record["published_conflict"] = draft.Exists && draft.PublishedVersion != publishedVersion
		record["_draft"] = draft
	}
	h.renderForm(w, r, collection, id, record, true)
}

func (h GeneratedAdminHandler) renderReadOnlyRecord(w http.ResponseWriter, r *http.Request, collection Collection, record map[string]any) {
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<dl class="studio-record-details">`); err != nil {
			return err
		}
		for _, field := range collection.Fields {
			if !field.VisibleInList {
				continue
			}
			if _, err := fmt.Fprintf(out, `<div><dt>%s</dt><dd>%s</dd></div>`, template.HTMLEscapeString(field.Label), template.HTMLEscapeString(displayFormValue(field, record[field.ID]))); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</dl>`); err != nil {
			return err
		}
		return ui.Link(ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: i18n.T(generatedAdminLocale(r), "admin.generated.form.back_to_list"), Class: "button button-quiet"}).Render(ctx, out)
	})
	card := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		return ui.Card(ui.CardProps{Title: collection.Label + " · " + displayValue(record[collection.TitleField]), Class: "studio-card", Body: body}).Render(ctx, out)
	})
	if err := h.Options.RenderPage(w, r, collection.Label, card); err != nil {
		h.apiError(w, r)
	}
}

func applyDraftFields(collection Collection, record, values map[string]any, locale, defaultLocale string) error {
	encoded, err := json.Marshal(values)
	if err != nil || len(encoded) > generatedAdminBodyLimit {
		return errors.New("draft payload exceeds the resource limit")
	}
	allowed := make(map[string]struct{}, len(collection.Fields))
	for _, field := range collection.Fields {
		if !field.ReadOnly && (!collection.Localization || locale == defaultLocale || field.Localized) {
			allowed[field.ID] = struct{}{}
		}
	}
	if len(values) > len(allowed) {
		return errors.New("draft field count exceeds the resource contract")
	}
	for field, value := range values {
		if _, ok := allowed[field]; !ok {
			return errors.New("draft field is outside the resource contract")
		}
		record[field] = value
	}
	return nil
}

func (h GeneratedAdminHandler) renderForm(w http.ResponseWriter, r *http.Request, collection Collection, id string, record map[string]any, editing bool) {
	h.renderFormState(w, r, collection, id, record, editing, nil, nil, 0, "")
}

func collectionField(collection Collection, id string) (Field, bool) {
	for _, field := range collection.Fields {
		if field.ID == id {
			return field, true
		}
	}
	return Field{}, false
}

func (h GeneratedAdminHandler) renderFormState(w http.ResponseWriter, r *http.Request, collection Collection, id string, record map[string]any, editing bool, submitted map[string]string, fieldErrors []ui.FieldValidationError, status int, failureKind string) {
	if h.Options.CSRFToken(r) == "" {
		http.Error(w, "admin page unavailable", http.StatusServiceUnavailable)
		return
	}
	values := make(map[string]string, len(collection.Fields))
	for _, field := range collection.Fields {
		values[field.ID] = displayFormValue(field, record[field.ID])
	}
	for fieldID, value := range submitted {
		if _, ok := collectionField(collection, fieldID); ok || fieldID == "version" || fieldID == "locale_version" || fieldID == "draft_version" || fieldID == "published_version" {
			values[fieldID] = value
		}
	}
	if collection.Versioned {
		if _, ok := submitted["version"]; !ok {
			values["version"] = displayValue(record["version"])
		}
	}
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		if _, ok := submitted["locale_version"]; !ok {
			values["locale_version"] = displayValue(record["locale_version"])
		}
	}
	if collection.Drafts {
		if _, ok := submitted["draft_version"]; !ok {
			values["draft_version"] = displayValue(record["draft_version"])
		}
		if _, ok := submitted["published_version"]; !ok {
			values["published_version"] = displayValue(record["published_version"])
		}
	}
	var revisions []RevisionSummary
	if editing && collection.Revisions {
		var err error
		revisions, err = h.Options.ListRevisions(r, collection, id, 50)
		if err != nil || len(revisions) > 50 {
			h.apiError(w, r)
			return
		}
		seen := make(map[int64]struct{}, len(revisions))
		for _, revision := range revisions {
			if revision.Version < 1 {
				h.apiError(w, r)
				return
			}
			if _, exists := seen[revision.Version]; exists {
				h.apiError(w, r)
				return
			}
			seen[revision.Version] = struct{}{}
		}
	}
	relationOptions := make(map[string][]ui.SelectOption)
	for _, field := range collection.Fields {
		if field.Kind != FieldRelation || field.Relation == nil || field.ReadOnly {
			continue
		}
		options, err := h.loadRelationOptions(r, field, values[field.ID])
		if err != nil {
			h.apiError(w, r)
			return
		}
		relationOptions[field.ID] = options
	}
	action := "/admin/resources/" + url.PathEscape(collection.ID)
	if editing {
		action += "/" + url.PathEscape(id)
		if collection.Drafts {
			action += "/draft"
		} else if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
			action += "/locale"
		}
	} else {
		action += "/new"
		if collection.Type == "global" {
			id = "global"
		}
		if id == "" {
			id = randomAdminID()
			if id == "" {
				http.Error(w, "admin page unavailable", http.StatusInternalServerError)
				return
			}
		}
	}
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if err := h.generatedLocaleNav(ctx, out, r, collection, id); err != nil {
			return err
		}
		localeInfo := localeState(r)
		localizedEdit := collection.Localization && localeInfo.Current != localeInfo.Options.Default
		draftEdit := collection.Drafts
		saveAllowed := !draftEdit || h.Options.CanSaveDraft(r, collection, id, localeInfo.Current) && record["published_conflict"] != true
		if draftEdit && record["published_conflict"] == true {
			if _, err := fmt.Fprintf(out, `<p role="alert">%s</p>`, template.HTMLEscapeString(i18n.T(generatedAdminLocale(r), "admin.generated.error.published_conflict"))); err != nil {
				return err
			}
		}
		fields := generatedFormFields(collection, values, relationOptions, id, editing, localeInfo.Current, localeInfo.Options.Default, localizedEdit, draftEdit, saveAllowed, generatedAdminLocale(r), h.Options.BlockEditor)
		formBody := templ.Component(fields)
		if status == http.StatusUnprocessableEntity || len(fieldErrors) > 0 {
			formBody = templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
				if err := ui.ValidationSummary(ui.ValidationSummaryProps{
					Title:   i18n.T(generatedAdminLocale(r), "admin.generated.validation.title"),
					Message: i18n.T(generatedAdminLocale(r), "admin.generated.validation.message"),
					Fields:  fieldErrors,
				}).Render(ctx, out); err != nil {
					return err
				}
				return fields.Render(ctx, out)
			})
		} else if status == http.StatusBadGateway {
			formBody = templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
				titleKey, messageKey := "admin.generated.error.title", "admin.generated.error.operation"
				if failureKind == "delete" {
					titleKey, messageKey = "admin.generated.error.delete_title", "admin.generated.error.delete_operation"
				}
				if err := ui.ValidationSummary(ui.ValidationSummaryProps{
					Title:   i18n.T(generatedAdminLocale(r), titleKey),
					Message: i18n.T(generatedAdminLocale(r), messageKey),
				}).Render(ctx, out); err != nil {
					return err
				}
				return fields.Render(ctx, out)
			})
		}
		props := ui.FormProps{Action: action, Method: "post", Class: "studio-form", CSRF: h.Options.CSRFToken(r), Body: formBody}
		if err := ui.Form(props).Render(ctx, out); err != nil {
			return err
		}
		if editing && !collection.Drafts && h.resourceAccess(r, collection, id).Delete && collection.Type != "global" && (!collection.Localization || localeState(r).Current == localeState(r).Options.Default) {
			deleteAction := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/delete"
			if collection.Localization {
				deleteAction = localePath(deleteAction, localeState(r).Current)
			}
			locale := generatedAdminLocale(r)
			if err := ui.Form(ui.FormProps{Action: deleteAction, Method: "post", CSRF: h.Options.CSRFToken(r), ConfirmAction: i18n.T(locale, "admin.generated.form.delete_confirm"), Body: ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.form.delete"), Variant: ui.Danger})}).Render(ctx, out); err != nil {
				return err
			}
		}
		if editing && collection.Workflow != nil && (!collection.Localization || localeState(r).Current == localeState(r).Options.Default || collection.Drafts) {
			if err := h.generatedWorkflowActions(ctx, out, r, collection, id, record); err != nil {
				return err
			}
		}
		if editing && collection.Revisions {
			if err := h.generatedRevisionHistory(ctx, out, r, collection, id, record, revisions); err != nil {
				return err
			}
		}
		if editing && collection.Localization {
			if err := h.generatedCopyLocaleActions(ctx, out, r, collection, id, record); err != nil {
				return err
			}
		}
		if editing && collection.Drafts && record["draft_exists"] == true {
			return h.generatedDiscardDraft(ctx, out, r, collection, id, record)
		}
		return nil
	})
	title := i18n.T(generatedAdminLocale(r), "admin.generated.form.create_title") + " · " + collection.Label
	if editing {
		title = i18n.T(generatedAdminLocale(r), "admin.generated.form.edit_title") + " · " + collection.Label
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	h.renderPage(w, r, title, body)
}

func (h GeneratedAdminHandler) loadRelationOptions(r *http.Request, field Field, selectedValue any) ([]ui.SelectOption, error) {
	selected := relationIDs(selectedValue)
	target, targetRegistered := h.Registry.Collection(field.Relation.Collection)
	titleField := ""
	if targetRegistered {
		titleField = target.TitleField
	}
	query := url.Values{"limit": {"50"}, "offset": {"0"}}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(field.Relation.Collection)+"?"+query.Encode(), nil)
	if err != nil || status != http.StatusOK {
		return nil, errors.New("related resource is unavailable")
	}
	var records []map[string]any
	if json.Unmarshal(response, &records) != nil {
		return nil, errors.New("related resource response is invalid")
	}
	options := make([]ui.SelectOption, 0, min(len(records), 50)+len(selected))
	known := make(map[string]struct{}, cap(options))
	add := func(record map[string]any) {
		id, _ := record["id"].(string)
		if id == "" {
			return
		}
		if _, exists := known[id]; exists {
			return
		}
		label := displayValue(record[titleField])
		if label == "" {
			label = id
		}
		known[id] = struct{}{}
		options = append(options, ui.SelectOption{Value: id, Label: label})
	}
	for _, record := range records[:min(len(records), 50)] {
		add(record)
	}
	for _, id := range selected {
		if _, exists := known[id]; exists {
			continue
		}
		response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(field.Relation.Collection)+"/"+url.PathEscape(id), nil)
		if err != nil || status != http.StatusOK {
			return nil, errors.New("selected relation is unavailable")
		}
		var record map[string]any
		if json.Unmarshal(response, &record) != nil {
			return nil, errors.New("selected relation response is invalid")
		}
		add(record)
	}
	return options, nil
}

func relationIDs(value any) []string {
	switch value := value.(type) {
	case string:
		if value == "" {
			return nil
		}
		return []string{value}
	case []string:
		return append([]string(nil), value...)
	case []any:
		ids := make([]string, 0, len(value))
		for _, item := range value {
			if id, ok := item.(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	default:
		return nil
	}
}

func (h GeneratedAdminHandler) generatedWorkflowActions(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any) error {
	current := PublicationState(displayValue(record[collection.Workflow.StateField]))
	expectedVersion := generatedRecordVersion(record)
	draft := draftForRecord(collection, id, record)
	if collection.Drafts {
		current = draft.State
		expectedVersion = draft.Version
		if !draft.Exists {
			return nil
		}
	}
	if !slices.Contains(collection.Workflow.States, current) || current == "" || expectedVersion < 1 {
		return nil
	}
	allowedTargets := make([]PublicationState, 0, len(collection.Workflow.Transitions[current]))
	for _, target := range collection.Workflow.Transitions[current] {
		allowed := false
		if collection.Drafts {
			allowed = h.Options.CanTransitionDraft(r, collection, draft, target)
		} else {
			allowed = h.Options.CanTransition(r, collection, record, target)
		}
		if allowed {
			allowedTargets = append(allowedTargets, target)
		}
	}
	if len(allowedTargets) == 0 {
		return nil
	}
	if _, err := io.WriteString(out, `<section class="studio-card" aria-labelledby="generated-workflow-heading"><h2 id="generated-workflow-heading">Yayın akışı</h2><div class="studio-actions">`); err != nil {
		return err
	}
	for _, target := range allowedTargets {
		body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "target", Value: string(target)}).Render(ctx, form); err != nil {
				return err
			}
			versionName := "expected_version"
			if collection.Drafts {
				versionName = "expected_draft_version"
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: versionName, Value: strconv.FormatInt(expectedVersion, 10)}).Render(ctx, form); err != nil {
				return err
			}
			if collection.Drafts {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_published_version", Value: strconv.FormatInt(draft.PublishedVersion, 10)}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: workflowActionLabel(generatedAdminLocale(r), target), Variant: ui.Primary}).Render(ctx, form)
		})
		action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/workflow"
		if collection.Localization {
			action = localePath(action, localeState(r).Current)
		}
		if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: body}).Render(ctx, out); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</div></section>`)
	return err
}

func draftForRecord(collection Collection, id string, record map[string]any) DraftDocument {
	if draft, ok := record["_draft"].(DraftDocument); ok {
		return draft
	}
	return DraftDocument{ID: id, Version: recordVersion(record["draft_version"]), PublishedVersion: recordVersion(record["published_version"]), State: PublicationState(displayValue(record[collection.Workflow.StateField])), Exists: record["draft_exists"] == true}
}

func (h GeneratedAdminHandler) generatedDiscardDraft(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any) error {
	draft := draftForRecord(collection, id, record)
	if !draft.Exists || !h.Options.CanDiscardDraft(r, collection, draft) {
		return nil
	}
	body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
		if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(draft.Version, 10)}).Render(ctx, form); err != nil {
			return err
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(generatedAdminLocale(r), "admin.generated.form.discard_draft"), Variant: ui.Danger}).Render(ctx, form)
	})
	action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/draft/discard"
	if collection.Localization {
		action = localePath(action, localeState(r).Current)
	}
	return ui.Form(ui.FormProps{Action: action, Method: http.MethodPost, CSRF: h.Options.CSRFToken(r), ConfirmAction: i18n.T(generatedAdminLocale(r), "admin.generated.form.discard_draft_confirm"), Body: body}).Render(ctx, out)
}

func (h GeneratedAdminHandler) generatedRevisionHistory(ctx context.Context, out io.Writer, r *http.Request, collection Collection, id string, record map[string]any, revisions []RevisionSummary) error {
	locale := generatedAdminLocale(r)
	if _, err := fmt.Fprintf(out, `<section class="studio-card" aria-labelledby="generated-revisions-heading"><h2 id="generated-revisions-heading">%s</h2>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.title"))); err != nil {
		return err
	}
	if len(revisions) == 0 {
		if _, err := fmt.Fprintf(out, `<p role="status">%s</p></section>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.empty"))); err != nil {
			return err
		}
		return nil
	}
	revisions = append([]RevisionSummary(nil), revisions...)
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].Version > revisions[j].Version })
	if _, err := io.WriteString(out, `<ol class="revision-list">`); err != nil {
		return err
	}
	currentVersion := generatedRecordVersion(record)
	for _, revision := range revisions {
		href := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(revision.Version, 10)
		if _, err := io.WriteString(out, `<li><div class="revision-list-item">`); err != nil {
			return err
		}
		label := fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.version"), revision.Version)
		visit := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if collection.Localization {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: localeState(r).Current}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Quiet}).Render(ctx, form)
		})
		if err := ui.Form(ui.FormProps{Action: href, Method: http.MethodGet, Body: visit}).Render(ctx, out); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `<p>`+template.HTMLEscapeString(revisionSummaryText(locale, revision))+`</p></div>`); err != nil {
			return err
		}
		if revision.Version != currentVersion && h.Options.CanRestore(r, collection, id, revision.Version) {
			body := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(currentVersion, 10)}).Render(ctx, form); err != nil {
					return err
				}
				return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.revisions.restore"), Variant: ui.Quiet}).Render(ctx, form)
			})
			action := href + "/restore"
			if collection.Localization {
				action = localePath(action, localeState(r).Current)
			}
			if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: body}).Render(ctx, out); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</li>`); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</ol></section>`)
	return err
}

func revisionSummaryText(locale string, revision RevisionSummary) string {
	parts := make([]string, 0, 3)
	if !revision.CreatedAt.IsZero() {
		parts = append(parts, revision.CreatedAt.Local().Format("02.01.2006 15:04"))
	}
	if strings.TrimSpace(revision.Actor) != "" {
		parts = append(parts, strings.TrimSpace(revision.Actor))
	}
	if strings.TrimSpace(revision.State) != "" {
		parts = append(parts, generatedStatusLabel(locale, revision.State))
	}
	return strings.Join(parts, " · ")
}

func (h GeneratedAdminHandler) revision(w http.ResponseWriter, r *http.Request, collection Collection, id, rawVersion string) {
	if !collection.Revisions || h.Options.GetRevision == nil {
		http.NotFound(w, r)
		return
	}
	version, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil || version < 1 {
		http.NotFound(w, r)
		return
	}
	revision, err := h.Options.GetRevision(r, collection, id, version)
	if err != nil || revision.Version != version || len(revision.Fields) > len(collection.Fields) {
		http.NotFound(w, r)
		return
	}
	snapshotJSON, err := json.Marshal(revision.Fields)
	if err != nil || len(snapshotJSON) > generatedAdminBodyLimit {
		http.Error(w, "revision unavailable", http.StatusRequestEntityTooLarge)
		return
	}
	current, err := h.currentResource(r, collection, id)
	if err != nil {
		h.apiError(w, r)
		return
	}
	before, after := make(map[string]string), make(map[string]string)
	labels := make(map[string]string)
	for _, field := range collection.Fields {
		if field.ReadOnly {
			continue
		}
		labels[field.ID] = field.Label
		if value, ok := revision.Fields[field.ID]; ok {
			before[field.ID] = value
		}
		after[field.ID] = displayValue(current[field.ID])
	}
	for fieldID := range revision.Fields {
		if _, ok := labels[fieldID]; !ok {
			http.NotFound(w, r)
			return
		}
	}
	diffs := CompareFields(before, after)
	locale := generatedAdminLocale(r)
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		content := templ.ComponentFunc(func(_ context.Context, body io.Writer) error {
			if len(diffs) == 0 {
				_, err := fmt.Fprintf(body, `<p role="status">%s</p>`, template.HTMLEscapeString(i18n.T(locale, "admin.generated.revisions.no_changes")))
				return err
			}
			if _, err := io.WriteString(body, `<dl class="revision-diff-list">`); err != nil {
				return err
			}
			for _, diff := range diffs {
				label := labels[diff.Field]
				if label == "" {
					continue
				}
				if _, err := fmt.Fprintf(body, `<div><dt>%s</dt><dd><del>%s</del><br><ins>%s</ins></dd></div>`, template.HTMLEscapeString(label), template.HTMLEscapeString(diff.Before), template.HTMLEscapeString(diff.After)); err != nil {
					return err
				}
			}
			_, err := io.WriteString(body, `</dl>`)
			return err
		})
		if err := ui.Card(ui.CardProps{Title: fmt.Sprintf(i18n.T(locale, "admin.generated.revisions.detail_title"), revision.Version), Class: "studio-card", Body: content}).Render(ctx, out); err != nil {
			return err
		}
		if h.Options.CanRestore(r, collection, id, version) {
			expectedVersion := generatedRecordVersion(current)
			if expectedVersion > 0 {
				formBody := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
					if err := ui.HiddenInput(ui.HiddenInputProps{Name: "expected_version", Value: strconv.FormatInt(expectedVersion, 10)}).Render(ctx, form); err != nil {
						return err
					}
					return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.revisions.restore"), Variant: ui.Primary}).Render(ctx, form)
				})
				action := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id) + "/revisions/" + strconv.FormatInt(version, 10) + "/restore"
				if collection.Localization {
					action = localePath(action, localeState(r).Current)
				}
				if err := ui.Form(ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: formBody}).Render(ctx, out); err != nil {
					return err
				}
			}
		}
		back := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
		backBody := templ.ComponentFunc(func(ctx context.Context, form io.Writer) error {
			if collection.Localization {
				if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: localeState(r).Current}).Render(ctx, form); err != nil {
					return err
				}
			}
			return ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.form.back_to_record"), Variant: ui.Quiet}).Render(ctx, form)
		})
		return ui.Form(ui.FormProps{Action: back, Method: http.MethodGet, Body: backBody}).Render(ctx, out)
	})
	h.renderPage(w, r, i18n.T(locale, "admin.generated.revisions.title")+" · "+collection.Label, body)
}

func (h GeneratedAdminHandler) restoreRevision(w http.ResponseWriter, r *http.Request, collection Collection, id, rawVersion string) {
	if !collection.Revisions || h.Options.RestoreRevision == nil || h.Options.CanRestore == nil {
		http.NotFound(w, r)
		return
	}
	revisionVersion, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil || revisionVersion < 1 {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != 2 || len(r.PostForm["expected_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	expectedVersion, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expectedVersion < 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	current, err := h.currentResource(r, collection, id)
	if err != nil {
		h.apiError(w, r)
		return
	}
	if generatedRecordVersion(current) != expectedVersion {
		http.Error(w, "document changed; reload before restoring", http.StatusConflict)
		return
	}
	if !h.Options.CanRestore(r, collection, id, revisionVersion) {
		http.Error(w, "restore unavailable", http.StatusForbidden)
		return
	}
	if err := h.Options.RestoreRevision(r, collection, id, revisionVersion, expectedVersion); err != nil {
		http.Error(w, "restore unavailable", http.StatusConflict)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h GeneratedAdminHandler) copyLocale(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Localization || h.Options.CopyLocale == nil || h.Options.CanCopyLocale == nil {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != 3 || len(r.PostForm["from_locale"]) != 1 || len(r.PostForm["expected_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	from := r.PostFormValue("from_locale")
	currentLocale := localeState(r).Current
	allowed := false
	for _, locale := range localeState(r).Options.Allowed {
		allowed = allowed || locale == from
	}
	expected, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expected < 0 || from == currentLocale || !allowed || !h.Options.CanCopyLocale(r, collection, id, from) {
		http.Error(w, "locale copy unavailable", http.StatusForbidden)
		return
	}
	_, currentVersion, err := h.Options.GetLocalized(r, collection, id, currentLocale)
	if err != nil || currentVersion < 0 {
		h.apiError(w, r)
		return
	}
	if currentVersion != expected {
		http.Error(w, "document changed; reload before copying", http.StatusConflict)
		return
	}
	if err := h.Options.CopyLocale(r, collection, id, from, currentLocale, expected); err != nil {
		http.Error(w, "locale copy unavailable", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), currentLocale), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) mutateLocale(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Localization || localeState(r).Current == localeState(r).Options.Default {
		http.NotFound(w, r)
		return
	}
	if !collection.Drafts && !h.resourceAccess(r, collection, id).Update {
		http.Error(w, "Bu kaydı düzenleme yetkiniz yok.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	versions := r.PostForm["locale_version"]
	if len(versions) != 1 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expected, err := strconv.ParseInt(versions[0], 10, 64)
	if err != nil || expected < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	fields := make(map[string]any, len(collection.Fields))
	allowed := map[string]struct{}{"_csrf": {}, "locale_version": {}}
	for _, field := range collection.Fields {
		if field.ReadOnly {
			continue
		}
		allowed[field.ID] = struct{}{}
		values, exists := r.PostForm[field.ID]
		if !exists || !field.Localized {
			continue
		}
		value, err := parseAdminField(field, values)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		fields[field.ID] = value
	}
	for key, values := range r.PostForm {
		if _, ok := allowed[key]; !ok || len(values) != 1 {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
	}
	_, currentVersion, err := h.Options.GetLocalized(r, collection, id, localeState(r).Current)
	if err != nil || currentVersion < 0 {
		h.apiError(w, r)
		return
	}
	if currentVersion != expected {
		http.Error(w, "document changed; reload before saving", http.StatusConflict)
		return
	}
	if err := h.Options.SaveLocalized(r, collection, id, localeState(r).Current, fields, expected); err != nil {
		http.Error(w, "translation could not be saved", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), localeState(r).Current), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) mutateDraft(w http.ResponseWriter, r *http.Request, collection Collection, id string, creating bool) {
	if !collection.Drafts {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if creating {
		id = strings.TrimSpace(r.PostFormValue("id"))
	}
	if !draftIDPattern.MatchString(id) {
		http.Error(w, "invalid document id", http.StatusBadRequest)
		return
	}
	draftVersions, publishedVersions := r.PostForm["draft_version"], r.PostForm["published_version"]
	if len(draftVersions) != 1 || len(publishedVersions) != 1 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expectedDraft, err := strconv.ParseInt(draftVersions[0], 10, 64)
	if err != nil || expectedDraft < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	expectedPublished, err := strconv.ParseInt(publishedVersions[0], 10, 64)
	if err != nil || expectedPublished < 0 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	fields := make(map[string]any, len(collection.Fields))
	allowed := map[string]struct{}{"_csrf": {}, "draft_version": {}, "published_version": {}}
	if creating {
		allowed["id"] = struct{}{}
	}
	for _, field := range collection.Fields {
		if collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			allowed[field.ID] = struct{}{}
			continue
		}
		if field.ReadOnly {
			continue
		}
		if collection.Localization && localeState(r).Current != localeState(r).Options.Default && !field.Localized {
			continue
		}
		allowed[field.ID] = struct{}{}
		values, exists := r.PostForm[field.ID]
		if !exists {
			continue
		}
		value, err := parseAdminField(field, values)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		fields[field.ID] = value
	}
	if creating {
		fields[collection.Workflow.StateField] = string(collection.Workflow.States[0])
	}
	for key, values := range r.PostForm {
		if _, ok := allowed[key]; !ok || len(values) != 1 {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
	}
	if !h.Options.CanSaveDraft(r, collection, id, localeState(r).Current) {
		http.Error(w, "draft editing unavailable", http.StatusForbidden)
		return
	}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	publishedVersion := int64(0)
	if err != nil || status == http.StatusOK && func() bool {
		var published map[string]any
		if json.Unmarshal(response, &published) != nil {
			return true
		}
		publishedVersion = generatedRecordVersion(published)
		return false
	}() || status != http.StatusOK && status != http.StatusNotFound {
		h.apiError(w, r)
		return
	}
	if publishedVersion != expectedPublished || creating && publishedVersion != 0 {
		http.Error(w, "published document changed; reload before saving", http.StatusConflict)
		return
	}
	if err := h.Options.SaveDraft(r, collection, id, localeState(r).Current, fields, expectedDraft, expectedPublished); err != nil {
		http.Error(w, "draft could not be saved", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), localeState(r).Current), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) discardDraft(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if !collection.Drafts {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != 2 || len(r.PostForm["expected_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	expected, err := strconv.ParseInt(r.PostFormValue("expected_version"), 10, 64)
	if err != nil || expected < 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
	if err != nil || !draft.Exists || draft.ID != id || draft.Version != expected {
		http.Error(w, "draft changed; reload before discarding", http.StatusConflict)
		return
	}
	if !h.Options.CanDiscardDraft(r, collection, draft) {
		http.Error(w, "draft discard unavailable", http.StatusForbidden)
		return
	}
	if err := h.Options.DiscardDraft(r, collection, id, localeState(r).Current, expected); err != nil {
		http.Error(w, "draft discard unavailable", http.StatusConflict)
		return
	}
	http.Redirect(w, r, localePath("/admin/resources/"+url.PathEscape(collection.ID)+"/drafts", localeState(r).Current), http.StatusSeeOther)
}

func (h GeneratedAdminHandler) draftList(w http.ResponseWriter, r *http.Request, collection Collection) {
	if !collection.Drafts {
		http.NotFound(w, r)
		return
	}
	query, err := parseGeneratedListQuery(r.URL.Query())
	if err != nil || query.Search != "" || query.Sort != "" || len(query.Filters) != 0 {
		http.Error(w, "invalid draft list query", http.StatusBadRequest)
		return
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	if page > 100 || page > 1 && h.Options.ListDraftPage == nil {
		http.Error(w, "draft list page is out of range", http.StatusBadRequest)
		return
	}
	const draftPageSize = DefaultPageSize
	var drafts []DraftDocument
	hasNext := false
	if h.Options.ListDraftPage != nil {
		drafts, err = h.Options.ListDraftPage(r, collection, localeState(r).Current, (page-1)*draftPageSize, draftPageSize+1)
		if len(drafts) > draftPageSize+1 {
			err = errors.New("draft list callback exceeded its bounded limit")
		}
		hasNext = err == nil && len(drafts) > draftPageSize && page < 100
		if len(drafts) > draftPageSize {
			drafts = drafts[:draftPageSize]
		}
	} else {
		drafts, err = h.Options.ListDrafts(r, collection, localeState(r).Current, 50)
		if len(drafts) > 50 {
			err = errors.New("legacy draft list callback exceeded its bound")
		}
	}
	if err != nil {
		h.apiError(w, r)
		return
	}
	seen := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		if !draftIDPattern.MatchString(draft.ID) || draft.Version < 1 || draft.PublishedVersion < 0 || !draft.Exists || !slices.Contains(collection.Workflow.States, draft.State) {
			h.apiError(w, r)
			return
		}
		if _, exists := seen[draft.ID]; exists {
			h.apiError(w, r)
			return
		}
		seen[draft.ID] = struct{}{}
		state := localeState(r)
		if applyDraftFields(collection, map[string]any{}, draft.Fields, state.Current, state.Options.Default) != nil {
			h.apiError(w, r)
			return
		}
	}
	locale := generatedAdminLocale(r)
	listCollection := Collection{
		ID: collection.ID, Label: collection.Label,
		Fields: []Field{
			{ID: "title", Label: i18n.T(locale, "admin.generated.drafts.title_column"), Kind: FieldText, VisibleInList: true},
			{ID: "status", Label: i18n.T(locale, "admin.generated.drafts.state_column"), Kind: FieldStatus, VisibleInList: true},
			{ID: "version", Label: i18n.T(locale, "admin.generated.drafts.version_column"), Kind: FieldNumber, VisibleInList: true},
			{ID: "updated_at", Label: i18n.T(locale, "admin.generated.drafts.updated_column"), Kind: FieldText, VisibleInList: true},
		},
		Actions: []Action{{Label: i18n.T(locale, "admin.generated.drafts.open"), Href: "/admin/resources/" + url.PathEscape(collection.ID)}},
	}
	rows := make([]Row, 0, len(drafts))
	for _, draft := range drafts {
		title := displayValue(draft.Fields[collection.TitleField])
		if title == "" {
			title = draft.ID
		}
		stateLabel := generatedStatusLabel(locale, string(draft.State))
		updated := "—"
		if !draft.UpdatedAt.IsZero() {
			if locale == "en" {
				updated = draft.UpdatedAt.Local().Format("Jan 02, 2006 15:04")
			} else {
				updated = draft.UpdatedAt.Local().Format("02.01.2006 15:04")
			}
		}
		detailURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(draft.ID)
		if collection.Localization {
			detailURL = localePath(detailURL, localeState(r).Current)
		}
		rows = append(rows, Row{
			ID: draft.ID, Values: map[string]string{
				"title": title, "status": stateLabel, "version": strconv.FormatInt(draft.Version, 10), "updated_at": updated,
			},
			Status:  StatusInfo{Value: string(draft.State), Label: stateLabel},
			Actions: []Action{{Label: i18n.T(locale, "admin.generated.drafts.open"), Href: detailURL}},
		})
	}
	baseURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/drafts"
	query.Page = page
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if err := h.generatedLocaleNav(ctx, out, r, collection, ""); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `<section class="studio-card"><h1>`+template.HTMLEscapeString(i18n.T(locale, "admin.generated.drafts.title"))+` · `+template.HTMLEscapeString(collection.Label)+`</h1>`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `<div class="table-wrap" id="generated-drafts-list" data-view-mode="table">`); err != nil {
			return err
		}
		if err := RenderList(out, ListData{
			Collection: listCollection, Rows: rows, Locale: localeState(r).Current, UILocale: locale,
			EmptyMessage:      i18n.T(locale, "admin.generated.drafts.empty"),
			ListCaption:       fmt.Sprintf(i18n.T(locale, "admin.generated.drafts.caption"), collection.Label),
			ActionColumnLabel: i18n.T(locale, "admin.generated.drafts.actions"),
		}); err != nil {
			return err
		}
		if h.Options.ListDraftPage != nil && page > 1 {
			previous := query
			previous.Page = page - 1
			if err := generatedListPageFormAt(ctx, out, baseURL, previous, localeState(r).Current, i18n.T(locale, "admin.generated.list.previous")); err != nil {
				return err
			}
		}
		if h.Options.ListDraftPage != nil && hasNext {
			next := query
			next.Page = page + 1
			if err := generatedListPageFormAt(ctx, out, baseURL, next, localeState(r).Current, i18n.T(locale, "admin.generated.list.next")); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</div></section>`)
		return err
	})
	h.renderPage(w, r, i18n.T(locale, "admin.generated.drafts.title")+" · "+collection.Label, body)
}

func workflowActionLabel(locale string, state PublicationState) string {
	key := "admin.generated.workflow.change_state"
	switch state {
	case StateDraft:
		key = "admin.generated.workflow.draft"
	case StateScheduled:
		key = "admin.generated.workflow.schedule"
	case StatePublished:
		key = "admin.generated.workflow.publish"
	case StateRetracted:
		key = "admin.generated.workflow.retract"
	}
	label := i18n.T(locale, key)
	if state != StateDraft && state != StateScheduled && state != StatePublished && state != StateRetracted {
		return fmt.Sprintf(label, state)
	}
	return label
}

func generatedStatusLabel(locale, status string) string {
	status = strings.TrimSpace(status)
	key := "admin.generated.drafts.state." + strings.ToLower(status)
	label := i18n.T(locale, key)
	if label == key {
		return status
	}
	return label
}

func (h GeneratedAdminHandler) transition(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if collection.Workflow == nil || collection.Drafts && (h.Options.TransitionDraft == nil || h.Options.CanTransitionDraft == nil) || !collection.Drafts && (h.Options.Transition == nil || h.Options.CanTransition == nil) || collection.Localization && !collection.Drafts && localeState(r).Current != localeState(r).Options.Default {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	versionKey, expectedFields := "expected_version", 3
	if collection.Drafts {
		versionKey, expectedFields = "expected_draft_version", 4
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm) != expectedFields || len(r.PostForm["target"]) != 1 || len(r.PostForm[versionKey]) != 1 || collection.Drafts && len(r.PostForm["expected_published_version"]) != 1 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	target := PublicationState(r.PostFormValue("target"))
	if !slices.Contains(collection.Workflow.States, target) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	if err != nil || status != http.StatusOK {
		h.apiError(w, r)
		return
	}
	var record map[string]any
	if json.Unmarshal(response, &record) != nil {
		h.apiError(w, r)
		return
	}
	expectedVersion, err := strconv.ParseInt(r.PostFormValue(versionKey), 10, 64)
	if err != nil || expectedVersion < 1 {
		http.Error(w, "document changed; reload before transitioning", http.StatusConflict)
		return
	}
	current := PublicationState(displayValue(record[collection.Workflow.StateField]))
	if collection.Drafts {
		draft, err := h.Options.GetDraft(r, collection, id, localeState(r).Current)
		publishedVersion := generatedRecordVersion(record)
		expectedPublished, parseErr := strconv.ParseInt(r.PostFormValue("expected_published_version"), 10, 64)
		if err != nil || parseErr != nil || !draft.Exists || draft.ID != id || draft.Version != expectedVersion || draft.PublishedVersion != expectedPublished || publishedVersion != expectedPublished {
			http.Error(w, "draft or published document changed; reload before transitioning", http.StatusConflict)
			return
		}
		current = draft.State
		if current == target || !collection.Workflow.CanTransition(current, target) || !h.Options.CanTransitionDraft(r, collection, draft, target) {
			http.Error(w, "transition unavailable", http.StatusForbidden)
			return
		}
		if err := h.Options.TransitionDraft(r, collection, id, localeState(r).Current, target, expectedVersion, expectedPublished); err != nil {
			http.Error(w, "transition unavailable", http.StatusConflict)
			return
		}
	} else if generatedRecordVersion(record) != expectedVersion {
		http.Error(w, "document changed; reload before transitioning", http.StatusConflict)
		return
	} else if current == target || !collection.Workflow.CanTransition(current, target) || !h.Options.CanTransition(r, collection, record, target) {
		http.Error(w, "transition unavailable", http.StatusForbidden)
		return
	} else if err := h.Options.Transition(r, collection, record, target, expectedVersion); err != nil {
		http.Error(w, "transition unavailable", http.StatusConflict)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func generatedRecordVersion(record map[string]any) int64 {
	if value, ok := record["draft_version"]; ok {
		return recordVersion(value)
	}
	if value, ok := record["locale_version"]; ok {
		return recordVersion(value)
	}
	return recordVersion(record["version"])
}

func recordVersion(value any) int64 {
	switch value := value.(type) {
	case float64:
		if value > 0 && value == float64(int64(value)) {
			return int64(value)
		}
	case int64:
		return value
	case int:
		return int64(value)
	case json.Number:
		version, _ := value.Int64()
		return version
	}
	return 0
}

func generatedFormFields(collection Collection, values map[string]string, relationOptions map[string][]ui.SelectOption, id string, editing bool, locale, defaultLocale string, localizedEdit, draftEdit, saveAllowed bool, interfaceLocale string, blockEditor BlockEditorConfig) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if editing && collection.Versioned && !localizedEdit && !draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "version", Value: values["version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if editing && localizedEdit && !draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale_version", Value: values["locale_version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "draft_version", Value: values["draft_version"]}).Render(ctx, out); err != nil {
				return err
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "published_version", Value: values["published_version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if !editing && collection.Type != "global" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "id", Value: id}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if collection.Workflow != nil && field.ID == collection.Workflow.StateField {
				value := values[field.ID]
				if value == "" && !editing {
					value = string(collection.Workflow.States[0])
				}
				if err := ui.Input(ui.InputProps{ID: field.ID, Name: field.ID, Label: field.Label, Type: "text", Value: value, ReadOnly: true}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.ReadOnly {
				continue
			}
			label := field.Label
			localizedOnly := collection.Localization && locale != defaultLocale && !field.Localized
			if field.Localized {
				label += " · " + locale
			}
			props := ui.InputProps{ID: field.ID, Name: field.ID, Label: label, Type: "text", Value: values[field.ID], Required: field.Required && (!collection.Localization || locale == defaultLocale), ReadOnly: localizedOnly}
			if localizedOnly {
				if err := ui.Input(props).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldArray {
				if err := ui.ItemListEditor(ui.ItemListEditorProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					AddLabel:               i18n.T(interfaceLocale, "admin.generated.widget.array.add"),
					AddAriaLabel:           fmt.Sprintf(i18n.T(interfaceLocale, "admin.generated.widget.array.add_aria"), field.Label),
					ItemPlaceholder:        i18n.T(interfaceLocale, "admin.generated.widget.array.item_placeholder"),
					ItemAriaLabelPattern:   i18n.T(interfaceLocale, "admin.generated.widget.array.item_aria"),
					RemoveAriaLabelPattern: i18n.T(interfaceLocale, "admin.generated.widget.array.remove_aria"),
				}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldBlocks || field.Widget == "block-editor" {
				if err := ui.BlockEditor(ui.BlockEditorProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], EditorScript: blockEditor.EditorScript, PreviewURL: blockEditor.PreviewURL, PreviewPending: i18n.T(interfaceLocale, "admin.generated.widget.blocks.preview_pending")}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldMedia || field.Widget == "media" {
				if err := ui.MediaPicker(ui.MediaPickerProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					SelectedImageAlt:  i18n.T(interfaceLocale, "admin.generated.widget.media.selected_image_alt"),
					EmptySelection:    i18n.T(interfaceLocale, "admin.generated.widget.media.empty"),
					ChooseFromLibrary: i18n.T(interfaceLocale, "admin.generated.widget.media.choose"),
					ClearSelection:    i18n.T(interfaceLocale, "admin.generated.widget.media.clear"),
					OpenLibrary:       i18n.T(interfaceLocale, "admin.generated.widget.media.open_library"),
				}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldRelation && field.Relation != nil {
				items := append([]ui.SelectOption(nil), relationOptions[field.ID]...)
				selected := relationIDsFromValue(values[field.ID], field.Relation.Cardinality)
				if field.Relation.Cardinality == "one" {
					items = append([]ui.SelectOption{{Value: "", Label: i18n.T(interfaceLocale, "admin.generated.form.choose_relation")}}, items...)
				}
				if err := ui.Select(ui.SelectProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Items: items, Selected: selected, Multiple: field.Relation.Cardinality == "many", Required: props.Required}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			switch field.Kind {
			case FieldNumber:
				props.Type = "number"
			case FieldBoolean:
				props.Type = "checkbox"
				props.Checked = values[field.ID] == "true"
			case FieldDate:
				props.Type = "date"
			}
			switch field.Widget {
			case "date":
				props.Type = "date"
			case "datetime", "datetime-local":
				props.Type = "datetime-local"
			case "checkbox":
				if field.Kind == FieldBoolean {
					props.Type = "checkbox"
					props.Checked = values[field.ID] == "true"
				}
			case "currency":
				if field.Kind == FieldNumber {
					props.Type = "number"
				}
			}
			var component templ.Component = ui.Input(props)
			if len(field.Options) > 0 {
				component = ui.Select(ui.SelectProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Options: field.Options, Required: props.Required})
			} else if field.Widget == "richtext" || field.Kind == FieldRichText {
				component = ui.RichTextEditor(ui.RichTextEditorProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					ToolbarLabel:     i18n.T(interfaceLocale, "admin.generated.widget.richtext.toolbar"),
					BoldLabel:        i18n.T(interfaceLocale, "admin.generated.widget.richtext.bold"),
					ItalicLabel:      i18n.T(interfaceLocale, "admin.generated.widget.richtext.italic"),
					UnderlineLabel:   i18n.T(interfaceLocale, "admin.generated.widget.richtext.underline"),
					LinkLabel:        i18n.T(interfaceLocale, "admin.generated.widget.richtext.link"),
					ImageLabel:       i18n.T(interfaceLocale, "admin.generated.widget.richtext.image"),
					BlockQuoteLabel:  i18n.T(interfaceLocale, "admin.generated.widget.richtext.quote"),
					OrderedListLabel: i18n.T(interfaceLocale, "admin.generated.widget.richtext.ordered_list"),
					BulletListLabel:  i18n.T(interfaceLocale, "admin.generated.widget.richtext.bullet_list"),
				})
			} else if field.Widget == "textarea" {
				component = ui.Textarea(ui.TextareaProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Required: props.Required, Rows: 8})
			}
			if err := component.Render(ctx, out); err != nil {
				return err
			}
		}
		label := i18n.T(interfaceLocale, "admin.generated.form.save")
		if draftEdit {
			label = i18n.T(interfaceLocale, "admin.generated.form.save_draft")
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Primary, Disabled: !saveAllowed}).Render(ctx, out)
	})
}

func relationIDsFromValue(value, cardinality string) []string {
	if value == "" || cardinality != "many" {
		return nil
	}
	return strings.Split(value, ",")
}

func (h GeneratedAdminHandler) mutate(w http.ResponseWriter, r *http.Request, collection Collection, id string, deleting bool) {
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if h.Options.ValidateCSRF == nil || !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	method, path := http.MethodPost, "/api/v1/"+url.PathEscape(collection.ID)
	var body []byte
	creating := !deleting && strings.HasSuffix(r.URL.Path, "/new")
	if deleting {
		method = http.MethodDelete
		path += "/" + url.PathEscape(id)
	} else {
		data := make(map[string]any, len(collection.Fields)+1)
		if collection.Versioned && !creating {
			versions := r.Form["version"]
			if len(versions) != 1 {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			version, err := strconv.ParseInt(versions[0], 10, 64)
			if err != nil || version < 1 {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			data["version"] = version
		}
		if id == "" {
			id = r.FormValue("id")
		}
		if collection.Type == "global" && id == "" {
			id = "global"
		}
		if id != "" {
			data["id"] = id
		}
		for _, field := range collection.Fields {
			if field.ReadOnly || (collection.Workflow != nil && field.ID == collection.Workflow.StateField) {
				continue
			}
			values, exists := r.Form[field.ID]
			if !exists {
				if field.Kind == FieldBoolean {
					data[field.ID] = false
				} else if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
					data[field.ID] = []string{}
				}
				continue
			}
			value, err := parseAdminField(field, values)
			if err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			data[field.ID] = value
		}
		if collection.Workflow != nil && creating {
			data[collection.Workflow.StateField] = string(collection.Workflow.States[0])
		} else if collection.Workflow != nil {
			current, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
			var record map[string]any
			if err != nil || status != http.StatusOK || json.Unmarshal(current, &record) != nil {
				h.renderMutationFailure(w, r, collection, id, creating, deleting)
				return
			}
			data[collection.Workflow.StateField] = record[collection.Workflow.StateField]
		}
		if id != "" && !creating {
			method = http.MethodPut
			path += "/" + url.PathEscape(id)
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		body = encoded
	}
	responseBody, status, err := h.callAPI(r, method, path, body)
	if err != nil {
		h.renderMutationFailure(w, r, collection, id, creating, deleting)
		return
	}
	switch status {
	case http.StatusConflict:
		http.Error(w, i18n.T(generatedAdminLocale(r), "admin.generated.error.conflict"), http.StatusConflict)
		return
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		if !deleting && status == http.StatusUnprocessableEntity {
			fieldErrors, valid := generatedAdminValidationErrors(collection, responseBody, generatedAdminLocale(r))
			if valid {
				record := map[string]any{}
				if !creating {
					var loadErr error
					record, loadErr = h.currentResource(r, collection, id)
					if loadErr != nil {
						h.apiError(w, r)
						return
					}
				}
				h.renderFormState(w, r, collection, id, record, !creating, submittedGeneratedValues(r, collection), fieldErrors, http.StatusUnprocessableEntity, "")
				return
			}
		}
		http.Error(w, "Bilgiler geçersiz veya eksik. Alanları kontrol edip yeniden deneyin.", http.StatusBadRequest)
		return
	case http.StatusForbidden:
		http.Error(w, "Bu işlem için yetkiniz yok.", http.StatusForbidden)
		return
	case http.StatusNotFound:
		http.NotFound(w, r)
		return
	}
	if status < 200 || status >= 300 {
		h.renderMutationFailure(w, r, collection, id, creating, deleting)
		return
	}
	redirect := "/admin/resources/" + url.PathEscape(collection.ID)
	if collection.Localization {
		redirect = localePath(redirect, localeState(r).Current)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h GeneratedAdminHandler) renderMutationFailure(w http.ResponseWriter, r *http.Request, collection Collection, id string, creating, deleting bool) {
	if deleting {
		record, err := h.currentResource(r, collection, id)
		if err != nil {
			locale := generatedAdminLocale(r)
			title := i18n.T(locale, "admin.generated.error.delete_title")
			message := i18n.T(locale, "admin.generated.error.delete_operation")
			http.Error(w, title+": "+message, http.StatusBadGateway)
			return
		}
		h.renderFormState(w, r, collection, id, record, true, nil, nil, http.StatusBadGateway, "delete")
		return
	}
	h.renderFormState(w, r, collection, id, map[string]any{}, !creating, submittedGeneratedValues(r, collection), nil, http.StatusBadGateway, "save")
}

func submittedGeneratedValues(r *http.Request, collection Collection) map[string]string {
	values := make(map[string]string, len(collection.Fields)+4)
	for _, field := range collection.Fields {
		if field.ReadOnly || collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			continue
		}
		items, exists := r.Form[field.ID]
		if !exists {
			if field.Kind == FieldBoolean || field.Kind == FieldArray || field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
				values[field.ID] = ""
			}
			continue
		}
		value, err := parseAdminField(field, items)
		if err != nil {
			continue
		}
		switch typed := value.(type) {
		case []string:
			if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
				values[field.ID] = strings.Join(typed, ",")
			} else {
				encoded, _ := json.Marshal(typed)
				values[field.ID] = string(encoded)
			}
		case bool:
			values[field.ID] = strconv.FormatBool(typed)
		default:
			values[field.ID] = fmt.Sprint(typed)
		}
	}
	for _, key := range []string{"version", "locale_version", "draft_version", "published_version"} {
		if items, ok := r.Form[key]; ok && len(items) == 1 {
			values[key] = items[0]
		}
	}
	return values
}

func generatedAdminValidationErrors(collection Collection, response []byte, locale string) ([]ui.FieldValidationError, bool) {
	var payload struct {
		Code   string            `json:"code"`
		Fields map[string]string `json:"fields"`
	}
	if len(response) == 0 || json.Unmarshal(response, &payload) != nil || payload.Code != "validation_failed" {
		return nil, false
	}
	result := make([]ui.FieldValidationError, 0, len(payload.Fields))
	for _, field := range collection.Fields {
		code, exists := payload.Fields[field.ID]
		if !exists || field.ReadOnly || collection.Workflow != nil && field.ID == collection.Workflow.StateField {
			continue
		}
		target := field.ID
		switch {
		case field.Kind == FieldBlocks || field.Widget == "block-editor":
			target += "-mount"
		case field.Widget == "richtext":
			target += "-editor"
		case field.Kind == FieldMedia || field.Widget == "media":
			target += "-field"
		}
		messageKey := "admin.generated.validation.field.invalid"
		if code == "required" {
			messageKey = "admin.generated.validation.field.required"
		}
		result = append(result, ui.FieldValidationError{FieldID: target, Label: field.Label, Message: i18n.T(locale, messageKey)})
	}
	return result, true
}

func parseAdminField(field Field, values []string) (any, error) {
	if field.Kind == FieldRelation && field.Relation != nil && field.Relation.Cardinality == "many" {
		limit := field.Relation.MaxItems
		if limit == 0 {
			limit = 100
		}
		if len(values) > limit {
			return nil, errors.New("too many related records")
		}
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return nil, errors.New("empty related record")
			}
			if _, exists := seen[value]; exists {
				return nil, errors.New("duplicate related record")
			}
			seen[value] = struct{}{}
		}
		return append([]string(nil), values...), nil
	}
	if len(values) != 1 {
		return nil, errors.New("multiple field values")
	}
	value := values[0]
	switch field.Kind {
	case FieldNumber:
		return strconv.ParseFloat(value, 64)
	case FieldBoolean:
		return value == "true" || value == "on" || value == "1", nil
	case FieldArray:
		if value == "" {
			return []string{}, nil
		}
		items := strings.Split(value, "\n")
		result := make([]string, 0, len(items))
		for _, item := range items {
			if item = strings.TrimSpace(item); item != "" {
				result = append(result, item)
			}
		}
		return result, nil
	default:
		return value, nil
	}
}

func (h GeneratedAdminHandler) callAPI(r *http.Request, method, target string, body []byte) ([]byte, int, error) {
	req := r.Clone(r.Context())
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, 0, err
	}
	req.Method, req.URL = method, parsed
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	recorder := newBoundedRecorder(generatedAdminBodyLimit)
	h.Options.API.ServeHTTP(recorder, req)
	return recorder.body.Bytes(), recorder.status, recorder.err
}

type boundedResponseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
	limit  int
	err    error
}

func newBoundedRecorder(limit int) *boundedResponseRecorder {
	return &boundedResponseRecorder{header: make(http.Header), limit: limit}
}

func (r *boundedResponseRecorder) Header() http.Header { return r.header }
func (r *boundedResponseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *boundedResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if r.body.Len()+len(data) > r.limit {
		r.err = errors.New("generated Admin API response exceeds limit")
		return 0, r.err
	}
	return r.body.Write(data)
}

func (h GeneratedAdminHandler) renderPage(w http.ResponseWriter, r *http.Request, title string, body templ.Component) {
	h.renderPageStatus(w, r, title, body, 0)
}

func (h GeneratedAdminHandler) renderPageStatus(w http.ResponseWriter, r *http.Request, title string, body templ.Component, status int) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 {
		w.WriteHeader(status)
	}
	locale := generatedAdminLocale(r)
	navigation := &generatedNavigationWriter{ResponseWriter: w, items: h.visibleCollections(r), current: r.URL.Path, locale: locale}
	if err := h.Options.RenderPage(navigation, r, title, body); err != nil {
		http.Error(w, "admin page unavailable", http.StatusInternalServerError)
	}
}

func generatedAdminLocale(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie("hydro_admin_locale"); err == nil && i18n.IsSupportedCatalogLocale(cookie.Value) {
			return i18n.ResolveCatalogLocale(cookie.Value)
		}
	}
	return i18n.ResolveCatalogLocale("")
}

type generatedNavigationWriter struct {
	http.ResponseWriter
	items   []Collection
	current string
	locale  string
	wrote   bool
}

func (w *generatedNavigationWriter) Write(payload []byte) (int, error) {
	if !w.wrote && len(w.items) > 0 {
		var err error
		payload, err = injectGeneratedNavigation(payload, w.items, w.current, w.locale)
		if err != nil {
			return 0, err
		}
		w.wrote = true
	}
	return w.ResponseWriter.Write(payload)
}

func injectGeneratedNavigation(markup []byte, collections []Collection, current, locale string) ([]byte, error) {
	items := make([]ui.SidebarItem, 0, len(collections))
	for _, collection := range collections {
		if collection.Hidden {
			continue
		}
		group := strings.TrimSpace(collection.Group)
		if group == "" || strings.EqualFold(group, "content") {
			group = "İÇERİK"
			if locale == "en" {
				group = "CONTENT"
			}
		}
		route := "/admin/resources/" + url.PathEscape(collection.ID)
		icon := collection.Icon
		if icon == "" {
			icon = "collection"
		}
		items = append(items, ui.SidebarItem{Route: route, Label: collection.Label, Icon: icon, Group: group, Active: current == route || strings.HasPrefix(current, route+"/")})
	}
	if len(items) == 0 {
		return markup, nil
	}
	var sidebar bytes.Buffer
	if err := ui.Sidebar(items).Render(context.Background(), &sidebar); err != nil {
		return nil, err
	}
	navigation := strings.TrimPrefix(sidebar.String(), `<nav aria-label="Modüller">`)
	navigation = strings.TrimSuffix(navigation, `</nav>`)
	result := string(markup)
	if end := strings.Index(result, `</nav>`); end >= 0 {
		result = result[:end] + navigation + result[end:]
	}
	var novaLinks bytes.Buffer
	for _, item := range items {
		className := "nova-topnav-link"
		if item.Active {
			className += " is-active"
		}
		if err := ui.Link(ui.LinkProps{Href: item.Route, Label: item.Label, Class: className}).Render(context.Background(), &novaLinks); err != nil {
			return nil, err
		}
	}
	if start := strings.Index(result, `class="nova-topnav-links"`); start >= 0 {
		if end := strings.Index(result[start:], `</nav>`); end >= 0 {
			end += start
			result = result[:end] + novaLinks.String() + result[end:]
		}
	}
	return []byte(result), nil
}

func (h GeneratedAdminHandler) apiError(w http.ResponseWriter, r *http.Request) {
	locale := generatedAdminLocale(r)
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if err := ui.ValidationSummary(ui.ValidationSummaryProps{
			Title:   i18n.T(locale, "admin.generated.error.load_title"),
			Message: i18n.T(locale, "admin.generated.error.load_message"),
		}).Render(ctx, out); err != nil {
			return err
		}
		return ui.Form(ui.FormProps{
			Action: r.URL.Path,
			Method: http.MethodGet,
			Class:  "generated-load-retry",
			Body:   ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(locale, "admin.generated.error.retry"), Variant: ui.Primary}),
		}).Render(ctx, out)
	})
	h.renderPageStatus(w, r, i18n.T(locale, "admin.generated.error.load_title"), body, http.StatusBadGateway)
}

func displayValue(value any) string {
	if value == nil {
		return ""
	}
	switch value := value.(type) {
	case []string:
		return strings.Join(value, ",")
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			items = append(items, fmt.Sprint(item))
		}
		return strings.Join(items, ",")
	}
	return fmt.Sprint(value)
}

func displayFormValue(field Field, value any) string {
	if field.Kind == FieldArray {
		switch items := value.(type) {
		case []string, []any:
			encoded, err := json.Marshal(items)
			if err == nil {
				return string(encoded)
			}
		}
	}
	return displayValue(value)
}

func randomAdminID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(value[:])
}
