package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

func main() {
	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "author", Label: "Authors", TitleField: "name", Fields: []adminstudio.Field{{ID: "name", Label: "Name", Kind: adminstudio.FieldText}}, Actions: []adminstudio.Action{{Label: "Authors", Href: "/admin/resources/author"}}},
		{
			ID: "article", Label: "Articles", Group: "Content", TitleField: "title", Versioned: true,
			Fields: []adminstudio.Field{
				{ID: "title", Label: "Title", Kind: adminstudio.FieldText, Required: true, VisibleInList: true},
				{ID: "author_id", Label: "Author", Kind: adminstudio.FieldRelation, Relation: &adminstudio.RelationRef{Collection: "author", Cardinality: "one"}},
				{ID: "cover_id", Label: "Cover image", Kind: adminstudio.FieldMedia, Widget: "media"},
				{ID: "body", Label: "Article body", Kind: adminstudio.FieldText, Widget: "richtext"},
			},
			Actions: []adminstudio.Action{{Label: "Articles", Href: "/admin/resources/article"}},
		},
		{
			ID: "story", Label: "Stories", Group: "Content", TitleField: "title", Versioned: true, Revisions: true, Drafts: true, Localization: true,
			Fields: []adminstudio.Field{
				{ID: "title", Label: "Title", Kind: adminstudio.FieldText, Required: true, Localized: true, VisibleInList: true},
				{ID: "status", Label: "Status", Kind: adminstudio.FieldSelect, Options: []string{"draft", "review", "published"}},
			},
			Actions:  []adminstudio.Action{{Label: "Stories", Href: "/admin/resources/story"}},
			Workflow: &adminstudio.Workflow{StateField: "status", States: []adminstudio.PublicationState{adminstudio.StateDraft, "review", adminstudio.StatePublished}, Transitions: map[adminstudio.PublicationState][]adminstudio.PublicationState{adminstudio.StateDraft: {adminstudio.StatePublished}, "review": {adminstudio.StatePublished}}},
		},
	})
	if err != nil {
		panic(err)
	}

	var mu sync.Mutex
	records := map[string]map[string]any{}
	draft := adminstudio.DraftDocument{ID: "story-1", Version: 1, PublishedVersion: 1, State: "review", Fields: map[string]any{"title": "Review copy"}, Exists: true}
	translationDraft := adminstudio.DraftDocument{ID: "story-1", Version: 1, PublishedVersion: 1, State: adminstudio.StateDraft, Fields: map[string]any{"title": "Türkçe taslak"}, Exists: true}
	discardDraft := adminstudio.DraftDocument{ID: "story-discard", Version: 1, State: adminstudio.StateDraft, Fields: map[string]any{"title": "Discard-only draft"}, Exists: true}
	publishedStory := map[string]any{"id": "story-1", "title": "Published story", "status": "published", "version": float64(1)}
	localizedStory := map[string]map[string]any{"tr": {"title": "Yayımlanmış Türkçe metin"}}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/author") {
			if strings.TrimSuffix(r.URL.Path, "/") == "/api/v1/author/author-1" {
				writeJSON(w, http.StatusOK, map[string]any{"id": "author-1", "name": "Test Author"})
				return
			}
			writeJSON(w, http.StatusOK, []map[string]any{{"id": "author-1", "name": "Test Author"}})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/story") {
			mu.Lock()
			defer mu.Unlock()
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed)
				return
			}
			if r.URL.Path == "/api/v1/story" {
				writeJSON(w, http.StatusOK, []map[string]any{clone(publishedStory)})
				return
			}
			if r.URL.Path == "/api/v1/story/story-1" {
				writeJSON(w, http.StatusOK, clone(publishedStory))
				return
			}
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/article"), "/"), "/")
		if r.URL.Path == "/api/v1/article" {
			if r.Method == http.MethodPost {
				var record map[string]any
				if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
					writeError(w, http.StatusBadRequest)
					return
				}
				id, _ := record["id"].(string)
				if id == "" {
					mu.Lock()
					id = "article-" + fmt.Sprint(len(records)+1)
					mu.Unlock()
				}
				record["id"] = id
				record["version"] = float64(1)
				mu.Lock()
				records[id] = record
				mu.Unlock()
				writeJSON(w, http.StatusCreated, clone(record))
				return
			}
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed)
				return
			}
			mu.Lock()
			result := make([]map[string]any, 0, len(records))
			for _, record := range records {
				result = append(result, clone(record))
			}
			mu.Unlock()
			writeJSON(w, http.StatusOK, result)
			return
		}
		if len(parts) != 1 || parts[0] == "" {
			writeError(w, http.StatusNotFound)
			return
		}
		id := parts[0]
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			record, ok := records[id]
			if !ok {
				writeError(w, http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, clone(record))
		case http.MethodPost, http.MethodPut:
			var record map[string]any
			if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
				writeError(w, http.StatusBadRequest)
				return
			}
			if id == "new" || id == "" || r.Method == http.MethodPost {
				if _, exists := records[id]; exists || id == "" || id == "new" {
					id = "article-" + fmt.Sprint(len(records)+1)
				}
				record["id"] = id
				records[id] = record
				writeJSON(w, http.StatusCreated, clone(record))
				return
			}
			current, exists := records[id]
			if !exists {
				writeError(w, http.StatusNotFound)
				return
			}
			expectedVersion, _ := record["version"].(float64)
			currentVersion, _ := current["version"].(float64)
			if expectedVersion != currentVersion {
				writeError(w, http.StatusConflict)
				return
			}
			record["version"] = currentVersion + 1
			record["id"] = id
			records[id] = record
			writeJSON(w, http.StatusOK, clone(record))
		case http.MethodDelete:
			delete(records, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed)
		}
	})

	mux := http.NewServeMux()
	vendorFS, err := fs.Sub(staticassets.FS, "vendor")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /vendor/{path...}", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))
	mux.HandleFunc("GET /api/media", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]any{{"id": "fixture-image", "status": "ready", "mime_type": "image/png", "original_filename": "fixture.png"}})
	})
	mux.Handle("/api/v1/story", api)
	mux.Handle("/api/v1/story/", api)
	if err := (adminstudio.GeneratedAdminHandler{Registry: registry, Options: adminstudio.GeneratedAdminOptions{
		API:             api,
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ListDrafts: func(_ *http.Request, collection adminstudio.Collection, locale string, limit int) ([]adminstudio.DraftDocument, error) {
			if collection.ID != "story" || limit != 50 {
				return nil, fmt.Errorf("unexpected draft list request: %s limit=%d", collection.ID, limit)
			}
			mu.Lock()
			defer mu.Unlock()
			if locale == "tr" {
				if translationDraft.Exists {
					return []adminstudio.DraftDocument{cloneDraft(translationDraft)}, nil
				}
				return nil, nil
			}
			result := make([]adminstudio.DraftDocument, 0, 2)
			if draft.Exists {
				result = append(result, cloneDraft(draft))
			}
			if discardDraft.Exists {
				result = append(result, cloneDraft(discardDraft))
			}
			return result, nil
		},
		GetDraft: func(_ *http.Request, collection adminstudio.Collection, id, locale string) (adminstudio.DraftDocument, error) {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" {
				return adminstudio.DraftDocument{}, fmt.Errorf("draft not found")
			}
			switch id {
			case draft.ID:
				if locale == "tr" {
					return cloneDraft(translationDraft), nil
				}
				return cloneDraft(draft), nil
			case discardDraft.ID:
				return cloneDraft(discardDraft), nil
			default:
				return adminstudio.DraftDocument{}, fmt.Errorf("draft not found")
			}
		},
		CanSaveDraft: func(_ *http.Request, collection adminstudio.Collection, id, locale string) bool {
			return collection.ID == "story" && id == "story-1" && (locale == "en" || locale == "tr")
		},
		SaveDraft: func(_ *http.Request, collection adminstudio.Collection, id, locale string, fields map[string]any, expectedDraft, expectedPublished int64) error {
			mu.Lock()
			defer mu.Unlock()
			current := &draft
			if locale == "tr" {
				current = &translationDraft
			}
			if collection.ID != "story" || id != current.ID || expectedDraft != current.Version || expectedPublished != current.PublishedVersion {
				return fmt.Errorf("draft changed")
			}
			current.Version++
			current.Fields = clone(fields)
			return nil
		},
		CanTransitionDraft: func(_ *http.Request, collection adminstudio.Collection, current adminstudio.DraftDocument, target adminstudio.PublicationState) bool {
			return collection.ID == "story" && target == adminstudio.StatePublished && (current.State == "review" || current.State == adminstudio.StateDraft)
		},
		TransitionDraft: func(_ *http.Request, collection adminstudio.Collection, id, locale string, target adminstudio.PublicationState, expectedDraft, expectedPublished int64) error {
			mu.Lock()
			defer mu.Unlock()
			current := &draft
			if locale == "tr" {
				current = &translationDraft
			}
			if collection.ID != "story" || id != current.ID || target != adminstudio.StatePublished || expectedDraft != current.Version || expectedPublished != current.PublishedVersion {
				return fmt.Errorf("draft changed")
			}
			current.Version++
			current.State = target
			if locale == "tr" {
				localizedStory[locale] = map[string]any{"title": current.Fields["title"]}
			} else {
				current.PublishedVersion = 2
				publishedStory = map[string]any{"id": id, "title": current.Fields["title"], "status": string(target), "version": float64(2)}
			}
			return nil
		},
		CanDiscardDraft: func(_ *http.Request, collection adminstudio.Collection, current adminstudio.DraftDocument) bool {
			return collection.ID == "story" && current.Exists
		},
		DiscardDraft: func(_ *http.Request, collection adminstudio.Collection, id, _ string, expected int64) error {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != discardDraft.ID || expected != discardDraft.Version {
				return fmt.Errorf("draft changed")
			}
			discardDraft.Exists = false
			return nil
		},
		Locales: func(_ *http.Request, collection adminstudio.Collection) (adminstudio.LocaleOptions, error) {
			if collection.ID != "story" {
				return adminstudio.LocaleOptions{}, fmt.Errorf("unexpected localized resource")
			}
			return adminstudio.LocaleOptions{Default: "en", Allowed: []string{"en", "tr"}}, nil
		},
		GetLocalized: func(_ *http.Request, collection adminstudio.Collection, id, locale string) (map[string]any, int64, error) {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != "story-1" || locale != "tr" {
				return nil, 0, fmt.Errorf("translation not found")
			}
			return clone(localizedStory[locale]), 1, nil
		},
		SaveLocalized: func(_ *http.Request, collection adminstudio.Collection, id, locale string, fields map[string]any, expected int64) error {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != "story-1" || locale != "tr" || expected != 1 {
				return fmt.Errorf("translation changed")
			}
			localizedStory[locale] = clone(fields)
			return nil
		},
		ListLocalized: func(_ *http.Request, collection adminstudio.Collection, ids []string, locale string) (map[string]map[string]any, error) {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || locale != "tr" {
				return nil, fmt.Errorf("unexpected translation list")
			}
			result := make(map[string]map[string]any)
			for _, id := range ids {
				if id == "story-1" {
					result[id] = clone(localizedStory[locale])
				}
			}
			return result, nil
		},
		CanCopyLocale: func(_ *http.Request, collection adminstudio.Collection, id, from string) bool {
			return collection.ID == "story" && id == "story-1" && from == "en"
		},
		CopyLocale: func(_ *http.Request, collection adminstudio.Collection, id, from, to string, expected int64) error {
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != "story-1" || from != "en" || to != "tr" || expected != 1 {
				return fmt.Errorf("translation copy rejected")
			}
			localizedStory[to] = map[string]any{"title": publishedStory["title"]}
			return nil
		},
		ListRevisions: func(_ *http.Request, collection adminstudio.Collection, _ string, limit int) ([]adminstudio.RevisionSummary, error) {
			if collection.ID != "story" || limit != 50 {
				return nil, fmt.Errorf("unexpected revisions request")
			}
			return nil, nil
		},
		GetRevision: func(*http.Request, adminstudio.Collection, string, int64) (adminstudio.RevisionSnapshot, error) {
			return adminstudio.RevisionSnapshot{}, fmt.Errorf("revision not found")
		},
		CanRestore: func(*http.Request, adminstudio.Collection, string, int64) bool { return false },
		RestoreRevision: func(*http.Request, adminstudio.Collection, string, int64, int64) error {
			return fmt.Errorf("revision not found")
		},
		CSRFToken: func(*http.Request) string { return "generated-richtext-e2e-csrf" },
		ValidateCSRF: func(r *http.Request) bool {
			return r.ParseForm() == nil && r.FormValue("_csrf") == "generated-richtext-e2e-csrf"
		},
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "generated-editor", Email: "editor@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}}).Register(mux); err != nil {
		panic(err)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	address := os.Getenv("HYDRO_E2E_ADDRESS")
	if address == "" {
		address = "127.0.0.1:8093"
	}
	if err := http.ListenAndServe(address, mux); err != nil {
		panic(err)
	}
}

func cloneDraft(draft adminstudio.DraftDocument) adminstudio.DraftDocument {
	fields := make(map[string]any, len(draft.Fields))
	for key, value := range draft.Fields {
		fields[key] = value
	}
	draft.Fields = fields
	return draft
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func clone(record map[string]any) map[string]any {
	encoded, _ := json.Marshal(record)
	var copy map[string]any
	_ = json.Unmarshal(encoded, &copy)
	return copy
}
