package adminstudio

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func testGeneratedAdmin(t *testing.T, api http.Handler, authorize func(*http.Request) bool) *http.ServeMux {
	t.Helper()
	registry, err := NewRegistry([]Collection{{
		ID: "project", Label: "Projects", Fields: []Field{
			{ID: "title", Label: "Title", Kind: FieldText, Required: true, VisibleInList: true},
			{ID: "status", Label: "Status", Kind: FieldSelect, Required: true, VisibleInList: true, Options: []string{"draft", "active"}},
		}, Actions: []Action{{Label: "Open projects", Href: "/admin/resources/project"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: authorize, CSRFToken: func(*http.Request) string { return "csrf-value" },
		ValidateCSRF: func(r *http.Request) bool { return r.Header.Get("X-CSRF-Test") == "valid" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
			_, _ = io.WriteString(w, "<main><h1>"+title+"</h1>")
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"p-1","title":"Roof & wall","status":"active"}]`)
	})
	server := httptest.NewServer(testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	defer server.Close()
	list, err := http.Get(server.URL + "/admin/resources/project")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	listHTML, _ := io.ReadAll(list.Body)
	if list.StatusCode != http.StatusOK || !strings.Contains(string(listHTML), "Roof &amp; wall") || !strings.Contains(string(listHTML), "/admin/resources/project/p-1") || list.Header.Get("Cache-Control") != "private, no-store" {
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
