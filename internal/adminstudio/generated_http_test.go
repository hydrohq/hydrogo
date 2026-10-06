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

	"github.com/a-h/templ"
)

func testGeneratedAdmin(t *testing.T, api http.Handler, authorize func(*http.Request) bool) *http.ServeMux {
	t.Helper()
	registry, err := NewRegistry([]Collection{{
		ID: "project", Label: "Projects", Group: "Content", TitleField: "title", Fields: []Field{
			{ID: "title", Label: "Title", Kind: FieldText, Required: true, VisibleInList: true, Searchable: true},
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
	if list.StatusCode != http.StatusOK || !strings.Contains(string(listHTML), `href="/admin/resources/project/p-1">Roof &amp; wall</a>`) || !strings.Contains(string(listHTML), `href="/admin/resources/project"`) || list.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("generated resource list was not rendered safely: status=%d body=%s", list.StatusCode, listHTML)
	}
	form, err := http.Get(server.URL + "/admin/resources/project/new")
	if err != nil {
		t.Fatal(err)
	}
	defer form.Body.Close()
	formHTML, _ := io.ReadAll(form.Body)
	if form.StatusCode != http.StatusOK || !strings.Contains(string(formHTML), `name="_csrf"`) || !strings.Contains(string(formHTML), `name="id"`) || !strings.Contains(string(formHTML), `name="status"`) {
		t.Fatalf("generated create form missing contract fields: status=%d body=%s", form.StatusCode, formHTML)
	}
}

func TestGeneratedAdminWorkflowUsesEditionTransitionBoundary(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title",
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
			if updated["status"] != "draft" {
				t.Errorf("ordinary document edit changed workflow state: %#v", updated)
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
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `action="/admin/resources/article/article-1/workflow"`) || !strings.Contains(edit.Body.String(), `name="target" value="review"`) || !strings.Contains(edit.Body.String(), `name="expected_version" value="4"`) {
		t.Fatalf("workflow transition was not rendered: status=%d body=%s", edit.Code, edit.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1/workflow", strings.NewReader("_csrf=csrf-value&target=review&expected_version=4"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || transitioned != "review" {
		t.Fatalf("workflow transition not delegated to Edition service: status=%d target=%q body=%s", response.Code, transitioned, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/admin/resources/article/article-1", strings.NewReader("_csrf=csrf-value&title=Changed&status=published"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("ordinary document edit failed: status=%d body=%s", response.Code, response.Body.String())
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
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/author" || r.URL.Query().Get("limit") != "50" || r.URL.Query().Get("offset") != "0" {
			t.Errorf("relation choices were not bounded: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"author-1","name":"Ada"}]`)
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
