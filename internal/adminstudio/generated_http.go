package adminstudio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/a-h/templ"
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
