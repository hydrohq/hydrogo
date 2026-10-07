package adminstudio

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func testGeneratedAdmin(t *testing.T, api http.Handler, authorize func(*http.Request) bool) *http.ServeMux {
	t.Helper()
	registry, err := NewRegistry([]Collection{{
		ID: "project", Label: "Projects", Group: "Content", TitleField: "title", Fields: []Field{
			{ID: "title", Label: "Title", Kind: FieldText, Required: true, VisibleInList: true, Searchable: true},
			{ID: "description", Label: "Description", Kind: FieldText, Widget: "richtext"},
			{ID: "status", Label: "Status", Kind: FieldSelect, Required: true, VisibleInList: true, Filterable: true, Options: []string{"draft", "active"}},
		}, Actions: []Action{{Label: "Open projects", Href: "/admin/resources/project"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: authorize, CanReadResource: func(*http.Request, Collection) bool { return true }, CSRFToken: func(*http.Request) string { return "csrf-value" },
		ValidateCSRF: func(r *http.Request) bool { return r.Header.Get("X-CSRF-Test") == "valid" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
			_, _ = io.WriteString(w, `<aside><nav><a href="/admin">Home</a></nav></aside><main><h1>`+title+`</h1>`)
			err := body.Render(r.Context(), w)
			_, _ = io.WriteString(w, "</main>")
			return err
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	return mux
}

func TestGeneratedAdminRendersListAndCreateForm(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/project" || r.Method != http.MethodGet {
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("where[or][0][title][contains]") != "Roof" || r.URL.Query().Get("where[status][equals]") != "active" || r.URL.Query().Get("sort") != "-title" {
			t.Errorf("list controls were not translated to the bounded resource query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"p-1","title":"Roof & wall","status":"active"}]`)
	})
	server := httptest.NewServer(testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	defer server.Close()
	list, err := http.Get(server.URL + "/admin/resources/project?search=Roof&filter%5Bstatus%5D=active&sort=-title")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	listHTML, _ := io.ReadAll(list.Body)
	if list.StatusCode != http.StatusOK || !strings.Contains(string(listHTML), `class="text-link" href="/admin/resources/project/p-1">Roof &amp; wall</a>`) || !strings.Contains(string(listHTML), `href="/admin/resources/project"`) || !strings.Contains(string(listHTML), `<option value="">Tümü</option>`) || list.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("generated resource list was not rendered safely: status=%d body=%s", list.StatusCode, listHTML)
	}
	form, err := http.Get(server.URL + "/admin/resources/project/new")
	if err != nil {
		t.Fatal(err)
	}
	defer form.Body.Close()
	formHTML, _ := io.ReadAll(form.Body)
	if form.StatusCode != http.StatusOK || !strings.Contains(string(formHTML), `name="_csrf"`) || !strings.Contains(string(formHTML), `name="id"`) || !strings.Contains(string(formHTML), `name="status"`) || !strings.Contains(string(formHTML), `class="richtext-field" data-richtext`) || !strings.Contains(string(formHTML), `name="description"`) {
		t.Fatalf("generated create form missing contract fields: status=%d body=%s", form.StatusCode, formHTML)
	}
}

func TestGeneratedAdminShowsRecoverableConflictMessage(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/project/p-1" {
			t.Errorf("unexpected API path %s", r.URL.Path)
		}
		if r.Method == http.MethodPut {
			http.Error(w, "stale version", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"p-1","title":"Current","description":"","status":"active"}`)
	})
	server := httptest.NewServer(testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	defer server.Close()

	response, err := http.Get(server.URL + "/admin/resources/project/p-1")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("edit page status = %d", response.StatusCode)
	}

	form := url.Values{"title": {"Stale"}, "description": {"draft"}, "status": {"active"}, "_csrf": {"csrf-value"}}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/p-1", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "Bu kayıt başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin.") {
		t.Fatalf("conflict response = %d %s", response.StatusCode, body)
	}
}

func TestGeneratedAdminWorkflowUsesEditionTransitionBoundary(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title", Versioned: true, Revisions: true,
		Fields:   []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}, {ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "review", "published"}}},
		Actions:  []Action{{Label: "Articles", Href: "/admin/resources/article"}},
		Workflow: &Workflow{StateField: "status", States: []PublicationState{StateDraft, "review", StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {"review"}, "review": {StatePublished}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/article/article-1" {
			t.Errorf("unexpected workflow record read %s %s", r.Method, r.URL.String())
		}
		if r.Method == http.MethodPut {
			var updated map[string]any
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Errorf("decode workflow resource update: %v", err)
			}
			if updated["status"] != "draft" || updated["version"] != float64(4) {
				t.Errorf("ordinary document edit changed workflow state or lost expected version: %#v", updated)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected workflow method %s", r.Method)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"article-1","title":"A story","status":"draft","version":4}`)
	})
	var transitioned PublicationState
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, Collection) bool { return true },
		CanTransition: func(_ *http.Request, _ Collection, record map[string]any, target PublicationState) bool {
			return record["status"] == "draft" && target == "review"
		},
		Transition: func(_ *http.Request, _ Collection, record map[string]any, target PublicationState, expectedVersion int64) error {
			if record["version"] != float64(4) || expectedVersion != 4 {
				return errors.New("unexpected record version")
			}
			transitioned = target
			return nil
		},
		ListRevisions: func(*http.Request, Collection, string, int) ([]RevisionSummary, error) { return nil, nil },
		GetRevision: func(*http.Request, Collection, string, int64) (RevisionSnapshot, error) {
			return RevisionSnapshot{}, errors.New("not found")
		},
		CanRestore: func(*http.Request, Collection, string, int64) bool { return false },
		RestoreRevision: func(*http.Request, Collection, string, int64, int64) error {
			return errors.New("not found")
		},
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/article-1", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `action="/admin/resources/article/article-1/workflow"`) || !strings.Contains(edit.Body.String(), `name="target" value="review"`) || !strings.Contains(edit.Body.String(), `name="expected_version" value="4"`) || !strings.Contains(edit.Body.String(), `name="version" value="4"`) {
		t.Fatalf("workflow transition was not rendered: status=%d body=%s", edit.Code, edit.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1/workflow", strings.NewReader("_csrf=csrf-value&target=review&expected_version=4"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || transitioned != "review" {
		t.Fatalf("workflow transition not delegated to Edition service: status=%d target=%q body=%s", response.Code, transitioned, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1", strings.NewReader("_csrf=csrf-value&title=Changed&status=published&version=4"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("ordinary document edit failed: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestGeneratedAdminRevisionHistoryDiffAndRestoreAreBounded(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", Versioned: true, Revisions: true,
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText}},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/article/a1" {
			t.Errorf("unexpected revision API call: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"a1","title":"Current","version":4}`)
	})
	var restored bool
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, Collection) bool { return true },
		ListRevisions: func(_ *http.Request, _ Collection, id string, limit int) ([]RevisionSummary, error) {
			if id != "a1" || limit != 50 {
				t.Errorf("revision history query not bounded or scoped: id=%q limit=%d", id, limit)
			}
			return []RevisionSummary{{Version: 3, Actor: "Editor <admin@example.test>", State: "draft", CreatedAt: time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)}}, nil
		},
		GetRevision: func(_ *http.Request, _ Collection, id string, version int64) (RevisionSnapshot, error) {
			if id != "a1" || version != 3 {
				return RevisionSnapshot{}, errors.New("not found")
			}
			return RevisionSnapshot{RevisionSummary: RevisionSummary{Version: 3}, Fields: map[string]string{"title": `<script>alert(1)</script>`}}, nil
		},
		CanRestore: func(_ *http.Request, _ Collection, id string, version int64) bool { return id == "a1" && version == 3 },
		RestoreRevision: func(_ *http.Request, _ Collection, id string, revision, expected int64) error {
			if id != "a1" || revision != 3 || expected != 4 {
				return errors.New("unexpected restore arguments")
			}
			restored = true
			return nil
		},
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), "/revisions/3") || !strings.Contains(edit.Body.String(), "Editor &lt;admin@example.test&gt;") {
		t.Fatalf("revision history missing or unsafe: status=%d body=%s", edit.Code, edit.Body.String())
	}
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1/revisions/3", nil))
	if detail.Code != http.StatusOK || strings.Contains(detail.Body.String(), `<script>`) || !strings.Contains(detail.Body.String(), "&lt;script&gt;") {
		t.Fatalf("revision diff did not escape snapshot: status=%d body=%s", detail.Code, detail.Body.String())
	}
	post := func(expected string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/revisions/3/restore", strings.NewReader("_csrf=csrf-value&expected_version="+expected))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	if response := post("3"); response.Code != http.StatusConflict || restored {
		t.Fatalf("stale restore should conflict before the Edition callback: status=%d restored=%v", response.Code, restored)
	}
	if response := post("4"); response.Code != http.StatusSeeOther || !restored {
		t.Fatalf("restore was not delegated with expected version: status=%d restored=%v body=%s", response.Code, restored, response.Body.String())
	}
}

func TestGeneratedAdminLocalizesOnlyOptedInFieldsThroughEditionStore(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", Localization: true, Versioned: true, Revisions: true, TitleField: "title",
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText, Localized: true, VisibleInList: true}, {ID: "status", Label: "Status", Kind: FieldText}},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/article":
			_, _ = io.WriteString(w, `[{"id":"a1","title":"Base title","status":"published","version":4}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/article/a1":
			_, _ = io.WriteString(w, `{"id":"a1","title":"Base title","status":"published","version":4}`)
		default:
			t.Errorf("unexpected localized resource API call: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	})
	var saved map[string]any
	var savedVersion int64
	var copied bool
	var renderErr error
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, Collection) bool { return true },
		Locales: func(*http.Request, Collection) (LocaleOptions, error) {
			return LocaleOptions{Default: "en", Allowed: []string{"en", "fr"}}, nil
		},
		ListRevisions: func(*http.Request, Collection, string, int) ([]RevisionSummary, error) { return nil, nil },
		GetRevision: func(*http.Request, Collection, string, int64) (RevisionSnapshot, error) {
			return RevisionSnapshot{}, errors.New("not found")
		},
		CanRestore:      func(*http.Request, Collection, string, int64) bool { return false },
		RestoreRevision: func(*http.Request, Collection, string, int64, int64) error { return errors.New("not found") },
		GetLocalized: func(_ *http.Request, _ Collection, id, locale string) (map[string]any, int64, error) {
			if id != "a1" || locale != "fr" {
				return nil, 0, errors.New("not found")
			}
			return map[string]any{"title": "Titre"}, 2, nil
		},
		SaveLocalized: func(_ *http.Request, _ Collection, id, locale string, fields map[string]any, expected int64) error {
			if id != "a1" || locale != "fr" {
				return errors.New("wrong locale scope")
			}
			saved, savedVersion = fields, expected
			return nil
		},
		ListLocalized: func(_ *http.Request, _ Collection, ids []string, locale string) (map[string]map[string]any, error) {
			if len(ids) != 1 || ids[0] != "a1" || locale != "fr" {
				return nil, errors.New("wrong list locale scope")
			}
			return map[string]map[string]any{"a1": {"title": "Titre liste"}}, nil
		},
		CanCopyLocale: func(_ *http.Request, _ Collection, id, from string) bool { return id == "a1" && from == "en" },
		CopyLocale: func(_ *http.Request, _ Collection, id, from, to string, expected int64) error {
			if id != "a1" || from != "en" || to != "fr" || expected != 2 {
				return errors.New("wrong copy contract")
			}
			copied = true
			return nil
		},
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			renderErr = body.Render(r.Context(), w)
			return renderErr
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1?locale=fr", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `value="Titre"`) || !strings.Contains(edit.Body.String(), `value="published" readonly`) || !strings.Contains(edit.Body.String(), `/article/a1/locale?locale=fr`) || !strings.Contains(edit.Body.String(), `name="locale_version" value="2"`) || !strings.Contains(edit.Body.String(), "EN içeriğini buraya kopyala") {
		t.Fatalf("localized editor contract missing: status=%d err=%v body=%s", edit.Code, renderErr, edit.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale?locale=fr", strings.NewReader("_csrf=csrf-value&locale_version=2&title=Nouveau&status=published"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || savedVersion != 2 || saved["title"] != "Nouveau" || len(saved) != 1 {
		t.Fatalf("localized mutation did not isolate translated fields or preserve expected version: status=%d version=%d fields=%#v body=%s", response.Code, savedVersion, saved, response.Body.String())
	}
	copyRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale-copy?locale=fr", strings.NewReader("_csrf=csrf-value&from_locale=en&expected_version=2"))
	copyRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	copyResponse := httptest.NewRecorder()
	mux.ServeHTTP(copyResponse, copyRequest)
	if copyResponse.Code != http.StatusSeeOther || !copied {
		t.Fatalf("authorized locale copy was not delegated with the current locale version: status=%d copied=%v body=%s", copyResponse.Code, copied, copyResponse.Body.String())
	}
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/admin/resources/article?locale=fr", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "Titre liste") || !strings.Contains(list.Body.String(), `name="locale" value="en"`) {
		t.Fatalf("localized list or locale navigation missing: status=%d body=%s", list.Code, list.Body.String())
	}
	unsupported := httptest.NewRecorder()
	mux.ServeHTTP(unsupported, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1?locale=de", nil))
	if unsupported.Code != http.StatusNotFound {
		t.Fatalf("Edition-disallowed locale was accepted: status=%d", unsupported.Code)
	}
}

func TestGeneratedAdminUsesBoundedRelationPicker(t *testing.T) {
	registry, err := NewRegistry([]Collection{
		{ID: "author", Label: "Authors", TitleField: "name", Fields: []Field{{ID: "name", Label: "Name", Kind: FieldText, VisibleInList: true}}, Actions: []Action{{Label: "Authors", Href: "/admin/resources/author"}}},
		{ID: "article", Label: "Articles", TitleField: "title", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}, {ID: "author_id", Label: "Author", Kind: FieldRelation, Relation: &RelationRef{Collection: "author", Cardinality: "one"}}, {ID: "cover_id", Label: "Cover image", Kind: FieldMedia, Widget: "media"}}, Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/author" && r.URL.Query().Get("limit") == "50" && r.URL.Query().Get("offset") == "0":
			_, _ = io.WriteString(w, `[{"id":"author-1","name":"Ada"}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/article":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Errorf("decode generated relation record: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected relation API request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, Collection) bool { return true },
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(*http.Request) bool { return true },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/resources/article/new", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `name="author_id"`) || !strings.Contains(response.Body.String(), `<option value="author-1">Ada</option>`) || !strings.Contains(response.Body.String(), `data-media-target="cover_id"`) {
		t.Fatalf("relation picker missing title choices: status=%d body=%s", response.Code, response.Body.String())
	}
	values := url.Values{"_csrf": {"csrf-value"}, "title": {"A story"}, "author_id": {"author-1"}}
	create := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/new", strings.NewReader(values.Encode()))
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(create, createRequest)
	if create.Code != http.StatusSeeOther || created["title"] != "A story" || created["author_id"] != "author-1" {
		t.Fatalf("selected relation did not reach resource API: status=%d payload=%#v body=%s", create.Code, created, create.Body.String())
	}
	field := Field{Kind: FieldRelation, Relation: &RelationRef{Collection: "author", Cardinality: "many", MaxItems: 2}}
	parsed, err := parseAdminField(field, []string{"a", "b"})
	if err != nil || !reflect.DeepEqual(parsed, []string{"a", "b"}) {
		t.Fatalf("many relation values not preserved: value=%#v err=%v", parsed, err)
	}
	if _, err := parseAdminField(field, []string{"a", "b", "c"}); err == nil {
		t.Fatal("many relation limit was not enforced")
	}
}

func TestGeneratedAdminMutationUsesCSRFAndResourceAPI(t *testing.T) {
	var created map[string]any
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/project" {
			t.Errorf("unexpected resource API call %s %s", r.Method, r.URL.String())
		}
		if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
			t.Errorf("decode forwarded create: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	})
	server := httptest.NewServer(testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	defer server.Close()
	values := url.Values{"id": {"p-2"}, "title": {"New project"}, "status": {"draft"}, "_csrf": {"csrf-value"}}
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/new", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("mutation without CSRF was accepted: %d", response.StatusCode)
	}
	request, _ = http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/new", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response, err = (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther || created["id"] != "p-2" || created["title"] != "New project" || created["status"] != "draft" {
		t.Fatalf("resource create was not forwarded: status=%d created=%#v", response.StatusCode, created)
	}
}

func TestGeneratedAdminRequiresEditionAuthorizationAndBoundaries(t *testing.T) {
	registry, _ := NewRegistry([]Collection{{ID: "project", Label: "Projects", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/resources/project"}}}})
	if err := (GeneratedAdminHandler{Registry: registry}).Register(http.NewServeMux()); err == nil {
		t.Fatal("registration without Edition security boundaries succeeded")
	}
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized request reached resource API") })
	server := httptest.NewServer(testGeneratedAdmin(t, api, func(*http.Request) bool { return false }))
	defer server.Close()
	response, err := http.Get(server.URL + "/admin/resources/project")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthorized generated Admin request status=%d", response.StatusCode)
	}
}

func TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title", Versioned: true, Revisions: true, Drafts: true,
		Fields:   []Field{{ID: "title", Label: "Title", Kind: FieldText}, {ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "review", "published"}}},
		Actions:  []Action{{Label: "Articles", Href: "/admin/resources/article"}},
		Workflow: &Workflow{StateField: "status", States: []PublicationState{StateDraft, "review", StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {"review"}, "review": {StatePublished}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	publishedReads, publishedWrites, saves, transitions := 0, 0, 0, 0
	draft := DraftDocument{ID: "a1", Version: 2, PublishedVersion: 4, State: StateDraft, Fields: map[string]any{"title": "Working title"}, Exists: true}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/article" {
			_, _ = io.WriteString(w, `[]`)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/article/a1" {
			if r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodDelete {
				publishedWrites++
			}
			t.Errorf("draft flow touched unexpected published API: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusInternalServerError)
			return
		}
		publishedReads++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"a1","title":"Published title","status":"published","version":4}`)
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, Collection) bool { return true },
		ListDrafts: func(_ *http.Request, _ Collection, _ string, limit int) ([]DraftDocument, error) {
			if limit != 50 {
				t.Errorf("draft list limit=%d", limit)
			}
			return []DraftDocument{draft}, nil
		},
		GetDraft: func(_ *http.Request, _ Collection, id, _ string) (DraftDocument, error) {
			if id != "a1" {
				return DraftDocument{}, errors.New("missing")
			}
			return draft, nil
		},
		CanSaveDraft: func(*http.Request, Collection, string, string) bool { return true },
		SaveDraft: func(_ *http.Request, _ Collection, id, _ string, fields map[string]any, expectedDraft, expectedPublished int64) error {
			saves++
			if id != "a1" || expectedDraft != 2 || expectedPublished != 4 || fields["title"] != "Edited draft" {
				return errors.New("unexpected draft CAS")
			}
			return nil
		},
		CanTransitionDraft: func(_ *http.Request, _ Collection, got DraftDocument, target PublicationState) bool {
			return got.Version == 2 && target == "review"
		},
		TransitionDraft: func(_ *http.Request, _ Collection, id, _ string, target PublicationState, expectedDraft, expectedPublished int64) error {
			transitions++
			if id != "a1" || target != "review" || expectedDraft != 2 || expectedPublished != 4 {
				return errors.New("unexpected transition CAS")
			}
			return nil
		},
		CanDiscardDraft: func(*http.Request, Collection, DraftDocument) bool { return true },
		DiscardDraft: func(_ *http.Request, _ Collection, id, _ string, expected int64) error {
			if id != "a1" || expected != 2 {
				return errors.New("unexpected discard CAS")
			}
			return nil
		},
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
		ListRevisions: func(*http.Request, Collection, string, int) ([]RevisionSummary, error) { return nil, nil },
		GetRevision: func(*http.Request, Collection, string, int64) (RevisionSnapshot, error) {
			return RevisionSnapshot{}, errors.New("missing")
		},
		CanRestore:      func(*http.Request, Collection, string, int64) bool { return false },
		RestoreRevision: func(*http.Request, Collection, string, int64, int64) error { return errors.New("missing") },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}

	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), "Working title") || !strings.Contains(edit.Body.String(), `action="/admin/resources/article/a1/draft"`) || !strings.Contains(edit.Body.String(), `name="draft_version" value="2"`) || !strings.Contains(edit.Body.String(), `name="expected_published_version" value="4"`) {
		t.Fatalf("draft editor missing isolated working copy or CAS tokens: status=%d body=%s", edit.Code, edit.Body.String())
	}
	post := func(path, form string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	if got := post("/admin/resources/article/a1/draft", "_csrf=csrf-value&title=Edited+draft&status=draft&draft_version=2&published_version=3"); got.Code != http.StatusConflict || saves != 0 {
		t.Fatalf("stale published version must block draft save: status=%d saves=%d", got.Code, saves)
	}
	if got := post("/admin/resources/article/a1/draft", "_csrf=csrf-value&title=Edited+draft&status=draft&draft_version=2&published_version=4"); got.Code != http.StatusSeeOther || saves != 1 {
		t.Fatalf("valid draft save was not delegated: status=%d saves=%d body=%s", got.Code, saves, got.Body.String())
	}
	if got := post("/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=1&expected_published_version=4"); got.Code != http.StatusConflict || transitions != 0 {
		t.Fatalf("stale draft must block transition: status=%d transitions=%d", got.Code, transitions)
	}
	if got := post("/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=2&expected_published_version=4"); got.Code != http.StatusSeeOther || transitions != 1 {
		t.Fatalf("valid transition was not delegated: status=%d transitions=%d body=%s", got.Code, transitions, got.Body.String())
	}
	if got := post("/admin/resources/article/a1/draft/discard", "_csrf=csrf-value&expected_version=2"); got.Code != http.StatusSeeOther {
		t.Fatalf("draft discard was not delegated: status=%d body=%s", got.Code, got.Body.String())
	}
	if publishedReads < 4 || publishedWrites != 0 {
		t.Fatalf("draft operations modified the published API: reads=%d writes=%d", publishedReads, publishedWrites)
	}

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/admin/resources/article/drafts", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "Working title") {
		t.Fatalf("draft list missing title: status=%d body=%s", list.Code, list.Body.String())
	}
	collectionList := httptest.NewRecorder()
	mux.ServeHTTP(collectionList, httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil))
	if collectionList.Code != http.StatusOK || !strings.Contains(collectionList.Body.String(), `href="/admin/resources/article/drafts"`) {
		t.Fatalf("collection draft navigation missing: status=%d body=%s", collectionList.Code, collectionList.Body.String())
	}
}

func TestGeneratedAdminDraftRejectsNonlocalizedFieldWrites(t *testing.T) {
	collection := Collection{Localization: true, Fields: []Field{{ID: "title", Localized: true}, {ID: "slug"}}}
	if err := applyDraftFields(collection, map[string]any{}, map[string]any{"slug": "changed"}, "fr", "en"); err == nil {
		t.Fatal("secondary locale draft accepted a nonlocalized field")
	}
	if err := applyDraftFields(collection, map[string]any{}, map[string]any{"slug": "base", "title": "base"}, "en", "en"); err != nil {
		t.Fatalf("default locale draft should include base fields: %v", err)
	}
}
