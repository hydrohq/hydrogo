package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

func main() {
	var editorialStore *postgresEditorialStore
	if dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL"); dsn != "" {
		var err error
		editorialStore, err = openPostgresEditorialStore(dsn)
		if err != nil {
			panic(err)
		}
	}
	if editorialStore != nil {
		defer editorialStore.Close()
	}

	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "author", Label: "Authors", TitleField: "name", Fields: []adminstudio.Field{{ID: "name", Label: "Name", Kind: adminstudio.FieldText}}, Actions: []adminstudio.Action{{Label: "Authors", Href: "/admin/resources/author"}}},
		{
			ID: "article", Label: "Articles", Group: "Content", TitleField: "title", Versioned: true,
			Fields: []adminstudio.Field{
				{ID: "title", Label: "Title", Kind: adminstudio.FieldText, Required: true, VisibleInList: true},
				{ID: "tags", Label: "Tags", Kind: adminstudio.FieldArray},
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
			if editorialStore != nil {
				if r.Method != http.MethodGet {
					writeError(w, http.StatusMethodNotAllowed)
					return
				}
				if r.URL.Path == "/api/v1/story" {
					stories, err := editorialStore.PublishedList()
					if err != nil {
						writeError(w, http.StatusInternalServerError)
						return
					}
					writeJSON(w, http.StatusOK, stories)
					return
				}
				id := strings.TrimPrefix(r.URL.Path, "/api/v1/story/")
				story, err := editorialStore.Published(id)
				if errors.Is(err, sql.ErrNoRows) {
					http.NotFound(w, r)
					return
				}
				if err != nil {
					writeError(w, http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, story)
				return
			}
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
		if editorialStore != nil {
			handlePostgresArticleAPI(w, r, editorialStore)
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
	mux.Handle("/api/v1/article", api)
	mux.Handle("/api/v1/article/", api)
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
			if editorialStore != nil {
				return editorialStore.Drafts(locale, limit)
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
			if editorialStore != nil {
				if collection.ID != "story" {
					return adminstudio.DraftDocument{}, fmt.Errorf("draft not found")
				}
				return editorialStore.Draft(id, locale)
			}
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
			if editorialStore != nil {
				if collection.ID != "story" || id != "story-1" || locale != "en" && locale != "tr" {
					return fmt.Errorf("draft changed")
				}
				return editorialStore.SaveDraft(id, locale, fields, expectedDraft, expectedPublished)
			}
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
			if editorialStore != nil {
				if collection.ID != "story" || id != "story-1" || target != adminstudio.StatePublished || locale != "en" && locale != "tr" {
					return fmt.Errorf("draft changed")
				}
				return editorialStore.PublishDraft(id, locale, expectedDraft, expectedPublished)
			}
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
		DiscardDraft: func(_ *http.Request, collection adminstudio.Collection, id, locale string, expected int64) error {
			if editorialStore != nil {
				if collection.ID != "story" {
					return fmt.Errorf("draft changed")
				}
				return editorialStore.DiscardDraft(id, locale, expected)
			}
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
			if editorialStore != nil {
				if collection.ID != "story" || id != "story-1" || locale != "tr" {
					return nil, 0, fmt.Errorf("translation not found")
				}
				return editorialStore.Localized(id, locale)
			}
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != "story-1" || locale != "tr" {
				return nil, 0, fmt.Errorf("translation not found")
			}
			return clone(localizedStory[locale]), 1, nil
		},
		SaveLocalized: func(_ *http.Request, collection adminstudio.Collection, id, locale string, fields map[string]any, expected int64) error {
			if editorialStore != nil {
				if collection.ID != "story" || id != "story-1" || locale != "tr" {
					return fmt.Errorf("translation changed")
				}
				return editorialStore.SaveLocalized(id, locale, fields, expected)
			}
			mu.Lock()
			defer mu.Unlock()
			if collection.ID != "story" || id != "story-1" || locale != "tr" || expected != 1 {
				return fmt.Errorf("translation changed")
			}
			localizedStory[locale] = clone(fields)
			return nil
		},
		ListLocalized: func(_ *http.Request, collection adminstudio.Collection, ids []string, locale string) (map[string]map[string]any, error) {
			if editorialStore != nil {
				if collection.ID != "story" || locale != "tr" {
					return nil, fmt.Errorf("unexpected translation list")
				}
				result := make(map[string]map[string]any, len(ids))
				for _, id := range ids {
					fields, _, err := editorialStore.Localized(id, locale)
					if err != nil {
						return nil, err
					}
					if len(fields) > 0 {
						result[id] = fields
					}
				}
				return result, nil
			}
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
			if editorialStore != nil {
				if collection.ID != "story" || id != "story-1" || from != "en" || to != "tr" {
					return fmt.Errorf("translation copy rejected")
				}
				return editorialStore.CopyLocale(id, from, to, expected)
			}
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
	server := &http.Server{Addr: address, Handler: mux}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case err := <-serveErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			panic(err)
		}
	}
}

func handlePostgresArticleAPI(w http.ResponseWriter, r *http.Request, store *postgresEditorialStore) {
	const collectionPath = "/api/v1/article"
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == collectionPath {
		if r.Method == http.MethodGet {
			articles, err := store.Articles()
			if err != nil {
				writeError(w, http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, articles)
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed)
			return
		}
		var article map[string]any
		if err := json.NewDecoder(r.Body).Decode(&article); err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		created, err := store.CreateArticle(article)
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return
	}
	id := strings.TrimPrefix(path, collectionPath+"/")
	if id == path || id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		article, err := store.Article(id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, article)
	case http.MethodPut, http.MethodPost:
		var article map[string]any
		if err := json.NewDecoder(r.Body).Decode(&article); err != nil {
			writeError(w, http.StatusBadRequest)
			return
		}
		updated, err := store.UpdateArticle(id, article)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound)
			return
		}
		if errors.Is(err, errPostgresArticleConflict) {
			writeError(w, http.StatusConflict)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := store.DeleteArticle(id); err != nil {
			writeError(w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed)
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
