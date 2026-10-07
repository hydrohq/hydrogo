package adminstudio

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	_ "modernc.org/sqlite"
)

// TestEditorialAdminSQLiteComposition drives the generated Admin handler,
// resource API, and Edition-owned stores together against SQLite.
func TestEditorialAdminSQLiteComposition(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "editorial.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE stories (id TEXT PRIMARY KEY, title TEXT NOT NULL, slug TEXT NOT NULL, status TEXT NOT NULL, author_id TEXT, cover_media_id TEXT, version INTEGER NOT NULL)`,
		`CREATE TABLE stories_drafts (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version INTEGER NOT NULL, published_version INTEGER NOT NULL, state TEXT NOT NULL, snapshot TEXT NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY(resource_id, locale))`,
		`CREATE TABLE stories_revisions (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version INTEGER NOT NULL, state TEXT NOT NULL, snapshot TEXT NOT NULL, actor_id TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(resource_id, locale, version))`,
		`CREATE TABLE pages (id TEXT PRIMARY KEY, title TEXT NOT NULL, slug TEXT NOT NULL, status TEXT NOT NULL, version INTEGER NOT NULL)`,
		`CREATE TABLE pages_locales (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version INTEGER NOT NULL, values_json TEXT NOT NULL, PRIMARY KEY(resource_id, locale))`,
		`CREATE TABLE pages_revisions (resource_id TEXT NOT NULL, locale TEXT NOT NULL, version INTEGER NOT NULL, state TEXT NOT NULL, snapshot TEXT NOT NULL, actor_id TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(resource_id, locale, version))`,
		`CREATE TABLE authors (id TEXT PRIMARY KEY, name TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO stories (id,title,slug,status,author_id,cover_media_id,version) VALUES ('story-1','Published title','published-title','published','author-1','media-1',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO pages (id,title,slug,status,version) VALUES ('page-1','English title','english-title','published',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO authors (id,name) VALUES ('author-1','Ada Lovelace')`); err != nil {
		t.Fatal(err)
	}
	if err := appendStoryRevision(db, "story-1", 1, map[string]any{"title": "Published title", "slug": "published-title", "status": "published", "author_id": "author-1", "cover_media_id": "media-1"}); err != nil {
		t.Fatal(err)
	}

	registry, err := NewRegistry([]Collection{
		{
			ID: "author", Label: "Authors", TitleField: "name", Fields: []Field{{ID: "name", Label: "Name", Kind: FieldText, VisibleInList: true}},
			Actions: []Action{{Label: "Authors", Href: "/admin/resources/author"}},
		},
		{
			ID: "story", Label: "Stories", TitleField: "title", Versioned: true, Revisions: true, Drafts: true,
			Fields: []Field{
				{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true},
				{ID: "slug", Label: "Slug", Kind: FieldText},
				{ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "review", "published"}},
				{ID: "author_id", Label: "Author", Kind: FieldRelation, Relation: &RelationRef{Collection: "author", Cardinality: "one"}},
				{ID: "cover_media_id", Label: "Cover", Kind: FieldMedia, Widget: "media"},
			},
			Actions:  []Action{{Label: "Stories", Href: "/admin/resources/story"}},
			Workflow: &Workflow{StateField: "status", States: []PublicationState{StateDraft, "review", StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {"review"}, "review": {StatePublished}}},
		},
		{
			ID: "page", Label: "Pages", TitleField: "title", Versioned: true, Revisions: true, Localization: true,
			Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText, Localized: true, VisibleInList: true}, {ID: "slug", Label: "Slug", Kind: FieldText}},
			Actions: []Action{{Label: "Pages", Href: "/admin/resources/page"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/"), "/")
		if len(parts) == 1 && parts[0] == "author" && r.Method == http.MethodGet {
			rows, err := db.Query(`SELECT id,name FROM authors ORDER BY name LIMIT 50`)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var values []map[string]any
			for rows.Next() {
				var id, name string
				if rows.Scan(&id, &name) != nil {
					http.Error(w, "database error", http.StatusInternalServerError)
					return
				}
				values = append(values, map[string]any{"id": id, "name": name})
			}
			writeEditorialJSON(w, values)
			return
		}
		if len(parts) < 1 || len(parts) > 2 || parts[0] != "story" && parts[0] != "page" {
			http.NotFound(w, r)
			return
		}
		table := editorialTable(parts[0])
		if len(parts) == 1 && r.Method == http.MethodGet {
			rows, err := db.Query(`SELECT id,title,slug,status,version FROM ` + table + ` ORDER BY title LIMIT 50`)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var values []map[string]any
			for rows.Next() {
				var id, title, slug, state string
				var version int64
				if rows.Scan(&id, &title, &slug, &state, &version) != nil {
					http.Error(w, "database error", http.StatusInternalServerError)
					return
				}
				values = append(values, map[string]any{"id": id, "title": title, "slug": slug, "status": state, "version": version})
			}
			writeEditorialJSON(w, values)
			return
		}
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		id := parts[1]
		switch r.Method {
		case http.MethodGet:
			var title, slug, state string
			var version int64
			if err := db.QueryRow(`SELECT title,slug,status,version FROM `+table+` WHERE id = ?`, id).Scan(&title, &slug, &state, &version); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.NotFound(w, r)
					return
				}
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			value := map[string]any{"id": id, "title": title, "slug": slug, "status": state, "version": version}
			if parts[0] == "story" {
				var authorID, coverID string
				_ = db.QueryRow(`SELECT COALESCE(author_id,''),COALESCE(cover_media_id,'') FROM stories WHERE id = ?`, id).Scan(&authorID, &coverID)
				value["author_id"], value["cover_media_id"] = authorID, coverID
			}
			writeEditorialJSON(w, value)
		case http.MethodPut:
			var value map[string]any
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)).Decode(&value) != nil {
				http.Error(w, "invalid payload", http.StatusBadRequest)
				return
			}
			result, err := db.Exec(`UPDATE `+table+` SET title = ?, slug = ?, status = ?, version = version + 1 WHERE id = ? AND version = ?`, value["title"], value["slug"], value["status"], id, recordVersion(value["version"]))
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			count, _ := result.RowsAffected()
			if count != 1 {
				http.Error(w, "conflict", http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			if _, err := db.Exec(`DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	can := func(*http.Request, Collection) bool { return true }
	options := GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: can,
		CSRFToken: func(*http.Request) string { return "csrf" }, ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
		Locales: func(*http.Request, Collection) (LocaleOptions, error) {
			return LocaleOptions{Default: "en", Allowed: []string{"en", "fr"}}, nil
		},
		ListRevisions: func(_ *http.Request, collection Collection, id string, limit int) ([]RevisionSummary, error) {
			if limit < 1 || limit > 50 {
				return nil, errors.New("invalid limit")
			}
			rows, err := db.Query(`SELECT version,state,actor_id,created_at FROM `+editorialTable(collection.ID)+`_revisions WHERE resource_id = ? AND locale = '' ORDER BY version DESC LIMIT ?`, id, limit)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var result []RevisionSummary
			for rows.Next() {
				var item RevisionSummary
				var created string
				if err := rows.Scan(&item.Version, &item.State, &item.Actor, &created); err != nil {
					return nil, err
				}
				item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				result = append(result, item)
			}
			return result, rows.Err()
		},
		GetRevision: func(_ *http.Request, collection Collection, id string, version int64) (RevisionSnapshot, error) {
			var state, snapshot string
			if err := db.QueryRow(`SELECT state,snapshot FROM `+editorialTable(collection.ID)+`_revisions WHERE resource_id = ? AND locale = '' AND version = ?`, id, version).Scan(&state, &snapshot); err != nil {
				return RevisionSnapshot{}, err
			}
			var fields map[string]string
			if err := json.Unmarshal([]byte(snapshot), &fields); err != nil {
				return RevisionSnapshot{}, err
			}
			return RevisionSnapshot{RevisionSummary: RevisionSummary{Version: version, State: state}, Fields: fields}, nil
		},
		CanRestore: func(_ *http.Request, collection Collection, id string, version int64) bool {
			var exists int
			return db.QueryRow(`SELECT 1 FROM `+editorialTable(collection.ID)+`_revisions WHERE resource_id = ? AND locale = '' AND version = ?`, id, version).Scan(&exists) == nil
		},
		RestoreRevision: func(_ *http.Request, collection Collection, id string, revision, expected int64) error {
			if collection.ID != "story" {
				return errors.New("page restore not configured")
			}
			var snapshot string
			if err := db.QueryRow(`SELECT snapshot FROM stories_revisions WHERE resource_id = ? AND locale = '' AND version = ?`, id, revision).Scan(&snapshot); err != nil {
				return err
			}
			var fields map[string]any
			if err := json.Unmarshal([]byte(snapshot), &fields); err != nil {
				return err
			}
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			result, err := tx.Exec(`UPDATE stories SET title=?,slug=?,status=?,author_id=?,cover_media_id=?,version=version+1 WHERE id=? AND version=?`, fields["title"], fields["slug"], fields["status"], fields["author_id"], fields["cover_media_id"], id, expected)
			if err != nil {
				return err
			}
			count, _ := result.RowsAffected()
			if count != 1 {
				return errors.New("published version conflict")
			}
			var version int64
			if err := tx.QueryRow(`SELECT version FROM stories WHERE id = ?`, id).Scan(&version); err != nil {
				return err
			}
			if err := appendStoryRevisionTx(tx, id, version, fields); err != nil {
				return err
			}
			return tx.Commit()
		},
		ListDrafts: func(_ *http.Request, _ Collection, locale string, limit int) ([]DraftDocument, error) {
			rows, err := db.Query(`SELECT resource_id,version,published_version,state,snapshot,updated_at FROM stories_drafts WHERE locale=? ORDER BY updated_at DESC LIMIT ?`, locale, limit)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var result []DraftDocument
			for rows.Next() {
				var item DraftDocument
				var snapshot, updated string
				if err := rows.Scan(&item.ID, &item.Version, &item.PublishedVersion, &item.State, &snapshot, &updated); err != nil {
					return nil, err
				}
				if err := json.Unmarshal([]byte(snapshot), &item.Fields); err != nil {
					return nil, err
				}
				item.Exists = true
				item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
				result = append(result, item)
			}
			return result, rows.Err()
		},
		GetDraft: func(_ *http.Request, _ Collection, id, locale string) (DraftDocument, error) {
			item := DraftDocument{ID: id}
			var snapshot, state string
			var updated string
			err := db.QueryRow(`SELECT version,published_version,state,snapshot,updated_at FROM stories_drafts WHERE resource_id=? AND locale=?`, id, locale).Scan(&item.Version, &item.PublishedVersion, &state, &snapshot, &updated)
			if errors.Is(err, sql.ErrNoRows) {
				err = db.QueryRow(`SELECT version FROM stories WHERE id=?`, id).Scan(&item.PublishedVersion)
				return item, err
			}
			if err != nil {
				return DraftDocument{}, err
			}
			if err := json.Unmarshal([]byte(snapshot), &item.Fields); err != nil {
				return DraftDocument{}, err
			}
			item.State, item.Exists = PublicationState(state), true
			item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
			return item, nil
		},
		CanSaveDraft: func(*http.Request, Collection, string, string) bool { return true },
		SaveDraft: func(_ *http.Request, _ Collection, id, locale string, fields map[string]any, expectedDraft, expectedPublished int64) error {
			encoded, err := json.Marshal(fields)
			if err != nil {
				return err
			}
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			var currentPublished int64
			if err := tx.QueryRow(`SELECT version FROM stories WHERE id=?`, id).Scan(&currentPublished); err != nil || currentPublished != expectedPublished {
				return errors.New("published version conflict")
			}
			var currentDraft int64
			err = tx.QueryRow(`SELECT version FROM stories_drafts WHERE resource_id=? AND locale=?`, id, locale).Scan(&currentDraft)
			if errors.Is(err, sql.ErrNoRows) {
				if expectedDraft != 0 {
					return errors.New("draft version conflict")
				}
			} else if err != nil || currentDraft != expectedDraft {
				return errors.New("draft version conflict")
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			_, err = tx.Exec(`INSERT INTO stories_drafts(resource_id,locale,version,published_version,state,snapshot,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(resource_id,locale) DO UPDATE SET version=excluded.version,published_version=excluded.published_version,state=excluded.state,snapshot=excluded.snapshot,updated_at=excluded.updated_at`, id, locale, expectedDraft+1, expectedPublished, string(StateDraft), string(encoded), now)
			if err != nil {
				return err
			}
			return tx.Commit()
		},
		CanTransitionDraft: func(_ *http.Request, _ Collection, _ DraftDocument, _ PublicationState) bool { return true },
		TransitionDraft: func(_ *http.Request, _ Collection, id, locale string, target PublicationState, expectedDraft, expectedPublished int64) error {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			var draftVersion, publishedVersion int64
			var snapshot string
			if err := tx.QueryRow(`SELECT version,published_version,snapshot FROM stories_drafts WHERE resource_id=? AND locale=?`, id, locale).Scan(&draftVersion, &publishedVersion, &snapshot); err != nil || draftVersion != expectedDraft || publishedVersion != expectedPublished {
				return errors.New("draft version conflict")
			}
			var currentPublished int64
			if err := tx.QueryRow(`SELECT version FROM stories WHERE id=?`, id).Scan(&currentPublished); err != nil || currentPublished != expectedPublished {
				return errors.New("published version conflict")
			}
			if target == StatePublished {
				var fields map[string]any
				if err := json.Unmarshal([]byte(snapshot), &fields); err != nil {
					return err
				}
				result, err := tx.Exec(`UPDATE stories SET title=?,slug=?,status='published',author_id=?,cover_media_id=?,version=version+1 WHERE id=? AND version=?`, fields["title"], fields["slug"], fields["author_id"], fields["cover_media_id"], id, expectedPublished)
				if err != nil {
					return err
				}
				count, _ := result.RowsAffected()
				if count != 1 {
					return errors.New("published version conflict")
				}
				var version int64
				if err := tx.QueryRow(`SELECT version FROM stories WHERE id=?`, id).Scan(&version); err != nil {
					return err
				}
				fields["status"] = string(StatePublished)
				if err := appendStoryRevisionTx(tx, id, version, fields); err != nil {
					return err
				}
				if _, err := tx.Exec(`DELETE FROM stories_drafts WHERE resource_id=? AND locale=? AND version=?`, id, locale, expectedDraft); err != nil {
					return err
				}
			} else {
				if _, err := tx.Exec(`UPDATE stories_drafts SET version=version+1,state=?,updated_at=? WHERE resource_id=? AND locale=? AND version=? AND published_version=?`, string(target), time.Now().UTC().Format(time.RFC3339Nano), id, locale, expectedDraft, expectedPublished); err != nil {
					return err
				}
			}
			return tx.Commit()
		},
		CanDiscardDraft: func(*http.Request, Collection, DraftDocument) bool { return true },
		DiscardDraft: func(_ *http.Request, _ Collection, id, locale string, expected int64) error {
			result, err := db.Exec(`DELETE FROM stories_drafts WHERE resource_id=? AND locale=? AND version=?`, id, locale, expected)
			if err != nil {
				return err
			}
			count, _ := result.RowsAffected()
			if count != 1 {
				return errors.New("draft version conflict")
			}
			return nil
		},
		GetLocalized: func(_ *http.Request, _ Collection, id, locale string) (map[string]any, int64, error) {
			var version int64
			var encoded string
			err := db.QueryRow(`SELECT version,values_json FROM pages_locales WHERE resource_id=? AND locale=?`, id, locale).Scan(&version, &encoded)
			if errors.Is(err, sql.ErrNoRows) {
				return map[string]any{}, 0, nil
			}
			if err != nil {
				return nil, 0, err
			}
			var values map[string]any
			err = json.Unmarshal([]byte(encoded), &values)
			return values, version, err
		},
		SaveLocalized: func(_ *http.Request, _ Collection, id, locale string, values map[string]any, expected int64) error {
			encoded, err := json.Marshal(values)
			if err != nil {
				return err
			}
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			var version int64
			err = tx.QueryRow(`SELECT version FROM pages_locales WHERE resource_id=? AND locale=?`, id, locale).Scan(&version)
			if errors.Is(err, sql.ErrNoRows) {
				if expected != 0 {
					return errors.New("locale version conflict")
				}
			} else if err != nil || version != expected {
				return errors.New("locale version conflict")
			}
			_, err = tx.Exec(`INSERT INTO pages_locales(resource_id,locale,version,values_json) VALUES(?,?,?,?) ON CONFLICT(resource_id,locale) DO UPDATE SET version=excluded.version,values_json=excluded.values_json`, id, locale, expected+1, string(encoded))
			if err != nil {
				return err
			}
			return tx.Commit()
		},
		ListLocalized: func(_ *http.Request, _ Collection, ids []string, locale string) (map[string]map[string]any, error) {
			result := make(map[string]map[string]any, len(ids))
			for _, id := range ids {
				values, _, err := func() (map[string]any, int64, error) {
					var version int64
					var encoded string
					err := db.QueryRow(`SELECT version,values_json FROM pages_locales WHERE resource_id=? AND locale=?`, id, locale).Scan(&version, &encoded)
					if errors.Is(err, sql.ErrNoRows) {
						return nil, 0, nil
					}
					if err != nil {
						return nil, 0, err
					}
					var value map[string]any
					err = json.Unmarshal([]byte(encoded), &value)
					return value, version, err
				}()
				if err != nil {
					return nil, err
				}
				if values != nil {
					result[id] = values
				}
			}
			return result, nil
		},
		CanCopyLocale: func(*http.Request, Collection, string, string) bool { return true },
		CopyLocale: func(_ *http.Request, _ Collection, id, from, to string, expected int64) error {
			if from != "en" || to != "fr" {
				return errors.New("unexpected locale direction")
			}
			var title string
			if err := db.QueryRow(`SELECT title FROM pages WHERE id=?`, id).Scan(&title); err != nil {
				return err
			}
			encoded, err := json.Marshal(map[string]any{"title": title})
			if err != nil {
				return err
			}
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			var version int64
			err = tx.QueryRow(`SELECT version FROM pages_locales WHERE resource_id=? AND locale=?`, id, to).Scan(&version)
			if errors.Is(err, sql.ErrNoRows) {
				if expected != 0 {
					return errors.New("locale version conflict")
				}
			} else if err != nil || version != expected {
				return errors.New("locale version conflict")
			}
			if _, err := tx.Exec(`INSERT INTO pages_locales(resource_id,locale,version,values_json) VALUES(?,?,?,?) ON CONFLICT(resource_id,locale) DO UPDATE SET version=excluded.version,values_json=excluded.values_json`, id, to, expected+1, string(encoded)); err != nil {
				return err
			}
			return tx.Commit()
		},
	}
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: options}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}

	get := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	post := func(path string, values url.Values) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder
	}

	newForm := get("/admin/resources/story/new")
	if newForm.Code != http.StatusOK || !strings.Contains(newForm.Body.String(), `<option value="author-1">Ada Lovelace</option>`) || !strings.Contains(newForm.Body.String(), `data-media-target="cover_media_id"`) {
		t.Fatalf("relation/media controls did not render from Edition APIs: status=%d body=%s", newForm.Code, newForm.Body.String())
	}
	edit := get("/admin/resources/story/story-1")
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), "Published title") || !strings.Contains(edit.Body.String(), `name="published_version" value="1"`) {
		t.Fatalf("published story editor did not load: status=%d body=%s", edit.Code, edit.Body.String())
	}
	draftSave := post("/admin/resources/story/story-1/draft", url.Values{"_csrf": {"csrf"}, "draft_version": {"0"}, "published_version": {"1"}, "title": {"Working title"}, "slug": {"working-title"}, "author_id": {"author-1"}, "cover_media_id": {"media-2"}, "status": {"draft"}})
	if draftSave.Code != http.StatusSeeOther {
		t.Fatalf("draft save failed: status=%d body=%s", draftSave.Code, draftSave.Body.String())
	}
	// A saved draft must survive the process/database connection lifecycle, not
	// only remain visible in the in-memory state of the composed handlers.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var publishedTitle, publishedCover string
	if err := db.QueryRow(`SELECT title,cover_media_id FROM stories WHERE id='story-1'`).Scan(&publishedTitle, &publishedCover); err != nil || publishedTitle != "Published title" || publishedCover != "media-1" {
		t.Fatalf("draft save changed the published record: title=%q cover=%q err=%v", publishedTitle, publishedCover, err)
	}
	drafts := get("/admin/resources/story/drafts")
	if drafts.Code != http.StatusOK || !strings.Contains(drafts.Body.String(), "Working title") {
		t.Fatalf("draft list did not show the SQLite working copy: status=%d body=%s", drafts.Code, drafts.Body.String())
	}
	review := post("/admin/resources/story/story-1/workflow", url.Values{"_csrf": {"csrf"}, "target": {"review"}, "expected_draft_version": {"1"}, "expected_published_version": {"1"}})
	if review.Code != http.StatusSeeOther {
		t.Fatalf("review transition failed: status=%d body=%s", review.Code, review.Body.String())
	}
	publish := post("/admin/resources/story/story-1/workflow", url.Values{"_csrf": {"csrf"}, "target": {"published"}, "expected_draft_version": {"2"}, "expected_published_version": {"1"}})
	if publish.Code != http.StatusSeeOther {
		t.Fatalf("publish transition failed: status=%d body=%s", publish.Code, publish.Body.String())
	}
	var version int64
	if err := db.QueryRow(`SELECT title,cover_media_id,version FROM stories WHERE id='story-1'`).Scan(&publishedTitle, &publishedCover, &version); err != nil || publishedTitle != "Working title" || publishedCover != "media-2" || version != 2 {
		t.Fatalf("publish did not atomically promote the draft: title=%q cover=%q version=%d err=%v", publishedTitle, publishedCover, version, err)
	}
	var draftsRemaining int
	if err := db.QueryRow(`SELECT count(*) FROM stories_drafts WHERE resource_id='story-1'`).Scan(&draftsRemaining); err != nil || draftsRemaining != 0 {
		t.Fatalf("published draft was not removed: count=%d err=%v", draftsRemaining, err)
	}
	revisionDetail := get("/admin/resources/story/story-1/revisions/2")
	if revisionDetail.Code != http.StatusOK || !strings.Contains(revisionDetail.Body.String(), "Sürüm 2") || !strings.Contains(revisionDetail.Body.String(), "değişen alan yok") {
		t.Fatalf("published revision was not visible: status=%d body=%s", revisionDetail.Code, revisionDetail.Body.String())
	}
	restore := post("/admin/resources/story/story-1/revisions/1/restore", url.Values{"_csrf": {"csrf"}, "expected_version": {"2"}})
	if restore.Code != http.StatusSeeOther {
		t.Fatalf("revision restore failed: status=%d body=%s", restore.Code, restore.Body.String())
	}
	if err := db.QueryRow(`SELECT title,version FROM stories WHERE id='story-1'`).Scan(&publishedTitle, &version); err != nil || publishedTitle != "Published title" || version != 3 {
		t.Fatalf("restore did not create a new published revision: title=%q version=%d err=%v", publishedTitle, version, err)
	}

	localeEditor := get("/admin/resources/page/page-1?locale=fr")
	if localeEditor.Code != http.StatusOK || !strings.Contains(localeEditor.Body.String(), `name="locale_version" value="0"`) {
		t.Fatalf("new locale editor did not start at version zero: status=%d body=%s", localeEditor.Code, localeEditor.Body.String())
	}
	localeSave := post("/admin/resources/page/page-1/locale?locale=fr", url.Values{"_csrf": {"csrf"}, "locale_version": {"0"}, "title": {"Titre français"}})
	if localeSave.Code != http.StatusSeeOther {
		t.Fatalf("localized save failed: status=%d body=%s", localeSave.Code, localeSave.Body.String())
	}
	localized := get("/admin/resources/page/page-1?locale=fr")
	base := get("/admin/resources/page/page-1?locale=en")
	if localized.Code != http.StatusOK || !strings.Contains(localized.Body.String(), "Titre français") || base.Code != http.StatusOK || !strings.Contains(base.Body.String(), "English title") {
		t.Fatalf("locale save leaked or failed: fr=%d %s en=%d %s", localized.Code, localized.Body.String(), base.Code, base.Body.String())
	}
	copy := post("/admin/resources/page/page-1/locale-copy?locale=fr", url.Values{"_csrf": {"csrf"}, "from_locale": {"en"}, "expected_version": {"1"}})
	if copy.Code != http.StatusSeeOther {
		t.Fatalf("locale copy failed: status=%d body=%s", copy.Code, copy.Body.String())
	}
	copied := get("/admin/resources/page/page-1?locale=fr")
	if copied.Code != http.StatusOK || !strings.Contains(copied.Body.String(), "English title") {
		t.Fatalf("locale copy did not use the Edition store: status=%d body=%s", copied.Code, copied.Body.String())
	}
}

func writeEditorialJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func appendStoryRevision(db *sql.DB, id string, version int64, fields map[string]any) error {
	return appendStoryRevisionTx(db, id, version, fields)
}

func appendStoryRevisionTx(tx interface {
	Exec(string, ...any) (sql.Result, error)
}, id string, version int64, fields map[string]any) error {
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO stories_revisions(resource_id,locale,version,state,snapshot,actor_id,created_at) VALUES(?,?,?,?,?,?,?)`, id, "", version, fields["status"], string(encoded), "editor-1", time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func editorialTable(id string) string {
	if id == "story" {
		return "stories"
	}
	return id + "s"
}
