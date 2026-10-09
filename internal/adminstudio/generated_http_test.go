package adminstudio

import (
	"bytes"
	"context"
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

func TestGeneratedFormUsesEditionBlockEditorAdapter(t *testing.T) {
	collection := Collection{ID: "article", Label: "Articles", Fields: []Field{{ID: "body", Label: "Body", Kind: FieldBlocks}}, Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}}}
	var out bytes.Buffer
	err := generatedFormFields(collection, map[string]string{"body": `{"blocks":[]}`}, nil, "article-1", false, "tr", "tr", false, false, true, BlockEditorConfig{EditorScript: "/assets/editor.js", PreviewURL: "/api/preview"}).Render(context.Background(), &out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-editorjs-mount="body"`, `data-editorjs-script="/assets/editor.js"`, `hx-post="/api/preview"`, `name="body"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("generated block field missing %q: %s", want, out.String())
		}
	}
}

func testGeneratedAdmin(t *testing.T, api http.Handler, authorize func(*http.Request) bool) *http.ServeMux {
	t.Helper()
	registry, err := NewRegistry([]Collection{{
		ID: "project", Label: "Projects", Group: "Content", TitleField: "title", Fields: []Field{
			{ID: "title", Label: "Title", Kind: FieldText, Required: true, VisibleInList: true, Searchable: true},
			{ID: "description", Label: "Description", Kind: FieldText, Widget: "richtext"},
			{ID: "status", Label: "Status", Kind: FieldSelect, Required: true, VisibleInList: true, Filterable: true, Options: []string{"draft", "active"}},
			{ID: "publish_date", Label: "Publish date", Kind: FieldDate},
			{ID: "starts_at", Label: "Starts at", Kind: FieldText, Widget: "datetime-local"},
			{ID: "budget", Label: "Budget", Kind: FieldNumber, Widget: "currency"},
			{ID: "featured", Label: "Featured", Kind: FieldBoolean},
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

type generatedAdminRoundTripper struct{ handler http.Handler }

func (rt generatedAdminRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		defer request.Body.Close()
	}
	recorder := httptest.NewRecorder()
	rt.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

func generatedAdminClient(t *testing.T, handler http.Handler) (*http.Client, string) {
	t.Helper()
	return &http.Client{Transport: generatedAdminRoundTripper{handler: handler}}, "http://admin.test"
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
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	list, err := client.Get(baseURL + "/admin/resources/project?search=Roof&filter%5Bstatus%5D=active&sort=-title")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	listHTML, _ := io.ReadAll(list.Body)
	if list.StatusCode != http.StatusOK || !strings.Contains(string(listHTML), `class="text-link" href="/admin/resources/project/p-1">Roof &amp; wall</a>`) || !strings.Contains(string(listHTML), `href="/admin/resources/project"`) || !strings.Contains(string(listHTML), `<option value="">Tümü</option>`) || list.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("generated resource list was not rendered safely: status=%d body=%s", list.StatusCode, listHTML)
	}
	form, err := client.Get(baseURL + "/admin/resources/project/new")
	if err != nil {
		t.Fatal(err)
	}
	defer form.Body.Close()
	formHTML, _ := io.ReadAll(form.Body)
	if form.StatusCode != http.StatusOK || !strings.Contains(string(formHTML), `name="_csrf"`) || !strings.Contains(string(formHTML), `name="id"`) || !strings.Contains(string(formHTML), `name="status"`) || !strings.Contains(string(formHTML), `class="richtext-field" data-richtext`) || !strings.Contains(string(formHTML), `name="description"`) || !strings.Contains(string(formHTML), `id="publish_date" name="publish_date" type="date"`) || !strings.Contains(string(formHTML), `id="starts_at" name="starts_at" type="datetime-local"`) || !strings.Contains(string(formHTML), `id="budget" name="budget" type="number"`) || !strings.Contains(string(formHTML), `id="featured" name="featured" type="checkbox"`) {
		t.Fatalf("generated create form missing contract fields: status=%d body=%s", form.StatusCode, formHTML)
	}
}

func TestGeneratedAdminUsesInterfaceLocaleForSearchEmptyState(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/project" {
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	})
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))
	request, err := http.NewRequest(http.MethodGet, baseURL+"/admin/resources/project?search=missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "hydro_admin_locale", Value: "en"})
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Try changing your search or filters. No records match them.") {
		t.Fatalf("English search empty state status=%d body=%s", response.StatusCode, body)
	}
}

func TestGeneratedAdminArrayFieldCreatesStructuredListAndRestoresIt(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title",
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText}, {ID: "tags", Label: "Tags", Kind: FieldArray}},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/article":
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Errorf("decode generated array payload: %v", err)
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			saved["id"] = "article-1"
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/article/article-1":
			_, _ = io.WriteString(w, `{"id":"article-1","title":"List article","tags":["first","second"]}`)
		default:
			t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
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
	form := httptest.NewRecorder()
	mux.ServeHTTP(form, httptest.NewRequest(http.MethodGet, "/admin/resources/article/new", nil))
	if form.Code != http.StatusOK || !strings.Contains(form.Body.String(), `data-item-list`) || !strings.Contains(form.Body.String(), `name="tags"`) {
		t.Fatalf("array editor not rendered: status=%d body=%s", form.Code, form.Body.String())
	}
	values := url.Values{"_csrf": {"csrf-value"}, "title": {"List article"}, "tags": {"first\nsecond"}}
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/new", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(create, request)
	want := []any{"first", "second"}
	if create.Code != http.StatusSeeOther || !reflect.DeepEqual(saved["tags"], want) {
		t.Fatalf("array create did not preserve structured rows: status=%d payload=%#v body=%s", create.Code, saved, create.Body.String())
	}
	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/article-1", nil))
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `[&#34;first&#34;,&#34;second&#34;]`) {
		t.Fatalf("saved array rows did not return to the editor: status=%d body=%s", edit.Code, edit.Body.String())
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
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, api, func(*http.Request) bool { return true }))

	response, err := client.Get(baseURL + "/admin/resources/project/p-1")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("edit page status = %d", response.StatusCode)
	}

	form := url.Values{"title": {"Stale"}, "description": {"draft"}, "status": {"active"}, "_csrf": {"csrf-value"}}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/admin/resources/project/p-1", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), "Bu kayıt başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin.") {
		t.Fatalf("conflict response = %d %s", response.StatusCode, body)
	}
}

func TestGeneratedAdminLocalizesAndSanitizesResourceFailure(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/project/project-1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"project-1","title":"Existing title","status":"active"}`)
			return
		}
		if !((r.Method == http.MethodPost && r.URL.Path == "/api/v1/project") || (r.Method == http.MethodPut && r.URL.Path == "/api/v1/project/project-1") || (r.Method == http.MethodDelete && r.URL.Path == "/api/v1/project/project-1")) {
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, "postgres://secret@db.internal hydrogo panic /srv/customer/data", http.StatusInternalServerError)
	})
	mux := testGeneratedAdmin(t, api, func(*http.Request) bool { return true })
	for _, tc := range []struct{ locale, title, message, deleteTitle, deleteMessage string }{
		{"tr", "Değişiklikleriniz kaydedilemedi", "Girdiğiniz bilgiler burada. Biraz sonra yeniden kaydetmeyi deneyin.", "Kayıt silinemedi", "Yeniden denemeden önce kaydın hâlâ listede olup olmadığını kontrol edin."},
		{"en", "Your changes could not be saved", "Your entries are still here. Try saving again in a moment.", "This record could not be deleted", "Check whether the record is still listed before trying again."},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			for _, action := range []struct{ name, path, title, message string }{
				{"create", "/admin/resources/project/new", tc.title, tc.message},
				{"update", "/admin/resources/project/project-1", tc.title, tc.message},
				{"delete", "/admin/resources/project/project-1/delete", tc.deleteTitle, tc.deleteMessage},
			} {
				t.Run(action.name, func(t *testing.T) {
					form := url.Values{"id": {"project-1"}, "title": {"Typed title"}, "status": {"active"}, "_csrf": {"csrf-value"}}
					request := httptest.NewRequest(http.MethodPost, action.path, strings.NewReader(form.Encode()))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					request.Header.Set("X-CSRF-Test", "valid")
					request.AddCookie(&http.Cookie{Name: "hydro_admin_locale", Value: tc.locale})
					response := httptest.NewRecorder()
					mux.ServeHTTP(response, request)
					body := response.Body.String()
					wantTitle, wantMessage, wantFieldValue := action.title, action.message, `value="Typed title"`
					if action.name == "delete" {
						wantFieldValue = `value="Existing title"`
					}
					if response.Code != http.StatusBadGateway || !strings.Contains(body, wantTitle) || !strings.Contains(body, wantMessage) || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, `<form`) || !strings.Contains(body, `name="title"`) || !strings.Contains(body, wantFieldValue) || response.Header().Get("Location") != "" {
						t.Fatalf("failure form status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), body)
					}
					if action.name == "delete" && (!strings.Contains(body, `action="/admin/resources/project/project-1/delete"`) || !strings.Contains(body, `name="_csrf"`) || !strings.Contains(body, `Existing title`)) {
						t.Fatalf("delete failure did not restore the record and delete action: %s", body)
					}
					for _, internal := range []string{"postgres://secret", "db.internal", "/srv/customer/data", "panic"} {
						if strings.Contains(body, internal) {
							t.Fatalf("internal API detail %q leaked: %s", internal, body)
						}
					}
				})
			}
		})
	}
}

func TestGeneratedAdminValidationRendersLocalizedFieldErrorsAndPreservesValues(t *testing.T) {
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/project" {
			t.Errorf("unexpected validation API request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"code":"validation_failed","fields":{"title":"required","status":"invalid","internal":"/var/lib/hydrogo/db.sqlite"},"error":"database unavailable"}`)
	}), func(*http.Request) bool { return true }))
	for _, test := range []struct {
		locale string
		copy   []string
	}{
		{locale: "tr", copy: []string{"Bu alanları kontrol edin", "Bu alan zorunludur.", "Bu değeri kontrol edip tekrar deneyin."}},
		{locale: "en", copy: []string{"Review these fields", "This field is required.", "Check this value and try again."}},
	} {
		t.Run(test.locale, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, baseURL+"/admin/resources/project/new", strings.NewReader("title=Typed+title&status=active"))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("X-CSRF-Test", "valid")
			request.AddCookie(&http.Cookie{Name: "hydro_admin_locale", Value: test.locale})
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			html := string(body)
			if response.StatusCode != http.StatusUnprocessableEntity || response.Header.Get("Location") != "" {
				t.Fatalf("validation response status=%d location=%q body=%s", response.StatusCode, response.Header.Get("Location"), html)
			}
			for _, marker := range append(test.copy, `role="alert"`, `href="#title"`, `href="#status"`, `id="title"`, `id="status"`, `value="Typed title"`, `value="active"`) {
				if !strings.Contains(html, marker) {
					t.Errorf("validation response missing %q: %s", marker, html)
				}
			}
			for _, leak := range []string{"database unavailable", "/var/lib/hydrogo", "internal"} {
				if strings.Contains(html, leak) {
					t.Errorf("validation response leaked API content %q: %s", leak, html)
				}
			}
		})
	}
}

func TestGeneratedAdminUpdateValidationReloadsUntouchedValues(t *testing.T) {
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/project/project-1" {
			t.Errorf("unexpected update validation API path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"id":"project-1","title":"Stored title","description":"Untouched copy","status":"draft"}`)
			return
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected update validation method %s", r.Method)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"code":"validation_failed","fields":{"title":"invalid"}}`)
	}), func(*http.Request) bool { return true }))
	request, err := http.NewRequest(http.MethodPost, baseURL+"/admin/resources/project/project-1", strings.NewReader("title=Edited+title&status=active"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	html := string(body)
	for _, marker := range []string{`value="Edited title"`, `Untouched copy`, `value="active"`, `href="#title"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("update validation response missing %q: %s", marker, html)
		}
	}
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("update validation status=%d body=%s", response.StatusCode, html)
	}
}

func TestGeneratedAdminValidationLinksWidgetFocusTargets(t *testing.T) {
	collection := Collection{Fields: []Field{
		{ID: "body", Label: "Body", Kind: FieldText, Widget: "richtext"},
		{ID: "layout", Label: "Layout", Kind: FieldBlocks},
		{ID: "cover", Label: "Cover", Kind: FieldMedia},
		{ID: "tags", Label: "Tags", Kind: FieldArray},
		{ID: "hidden", Label: "Hidden", Kind: FieldText, ReadOnly: true},
	}}
	fieldErrors, valid := generatedAdminValidationErrors(collection, []byte(`{"code":"validation_failed","fields":{"body":"required","layout":"required","cover":"required","tags":"required","hidden":"required","unknown":"required"}}`), "en")
	if !valid {
		t.Fatal("structured validation response was not recognized")
	}
	want := []string{"body-editor", "layout-mount", "cover-field", "tags"}
	if len(fieldErrors) != len(want) {
		t.Fatalf("field errors=%#v, want %v", fieldErrors, want)
	}
	for i, fieldError := range fieldErrors {
		if fieldError.FieldID != want[i] {
			t.Errorf("field error %d target=%q, want %q", i, fieldError.FieldID, want[i])
		}
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
	resourceReads := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/article/a1" {
			t.Errorf("unexpected revision API call: %s %s", r.Method, r.URL.String())
		}
		resourceReads++
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
		CanRestore: func(r *http.Request, _ Collection, id string, version int64) bool {
			return r.Header.Get("X-Test-Role") == "admin" && id == "a1" && version == 3
		},
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
	post := func(expected, role string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/revisions/3/restore", strings.NewReader("_csrf=csrf-value&expected_version="+expected))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("X-Test-Role", role)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	readsBeforeDeniedRestore := resourceReads
	deniedRestore := httptest.NewRecorder()
	deniedRestoreRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/revisions/3/restore", strings.NewReader("expected_version=4"))
	deniedRestoreRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(deniedRestore, deniedRestoreRequest)
	if deniedRestore.Code != http.StatusForbidden || restored || resourceReads != readsBeforeDeniedRestore {
		t.Fatalf("revision restore without CSRF was not rejected before reads or writes: status=%d reads=%d restored=%v", deniedRestore.Code, resourceReads, restored)
	}
	if response := post("3", "admin"); response.Code != http.StatusConflict || restored {
		t.Fatalf("stale restore should conflict before the Edition callback: status=%d restored=%v", response.Code, restored)
	}
	for _, role := range []string{"editor", "viewer"} {
		if response := post("4", role); response.Code != http.StatusForbidden || restored {
			t.Fatalf("%s restore should be denied by Edition capability: status=%d restored=%v body=%s", role, response.Code, restored, response.Body.String())
		}
	}
	if response := post("4", "admin"); response.Code != http.StatusSeeOther || !restored {
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
		ResourceAccess: func(r *http.Request, _ Collection, id string) ResourceAccess {
			return ResourceAccess{Update: id != "" && r.Header.Get("X-Test-Role") == "admin"}
		},
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
		CanCopyLocale: func(r *http.Request, _ Collection, id, from string) bool {
			return r.Header.Get("X-Test-Role") == "admin" && id == "a1" && from == "en"
		},
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
	editRequest := httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1?locale=fr", nil)
	editRequest.Header.Set("X-Test-Role", "admin")
	mux.ServeHTTP(edit, editRequest)
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), `value="Titre"`) || !strings.Contains(edit.Body.String(), `value="published" readonly`) || !strings.Contains(edit.Body.String(), `/article/a1/locale?locale=fr`) || !strings.Contains(edit.Body.String(), `name="locale_version" value="2"`) || !strings.Contains(edit.Body.String(), "EN içeriğini buraya kopyala") {
		t.Fatalf("localized editor contract missing: status=%d err=%v body=%s", edit.Code, renderErr, edit.Body.String())
	}
	deniedSave := httptest.NewRecorder()
	deniedSaveRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale?locale=fr", strings.NewReader("locale_version=2&title=Nouveau"))
	deniedSaveRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(deniedSave, deniedSaveRequest)
	if deniedSave.Code != http.StatusForbidden || saved != nil {
		t.Fatalf("localized save without CSRF was not rejected before storage: status=%d fields=%#v", deniedSave.Code, saved)
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale?locale=fr", strings.NewReader("_csrf=csrf-value&locale_version=2&title=Nouveau&status=published"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Test-Role", "admin")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || savedVersion != 2 || saved["title"] != "Nouveau" || len(saved) != 1 {
		t.Fatalf("localized mutation did not isolate translated fields or preserve expected version: status=%d version=%d fields=%#v body=%s", response.Code, savedVersion, saved, response.Body.String())
	}
	deniedCopyRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale-copy?locale=fr", strings.NewReader("from_locale=en&expected_version=2"))
	deniedCopyRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deniedCopyRequest.Header.Set("X-Test-Role", "admin")
	deniedCopy := httptest.NewRecorder()
	mux.ServeHTTP(deniedCopy, deniedCopyRequest)
	if deniedCopy.Code != http.StatusForbidden || copied {
		t.Fatalf("locale copy without CSRF was not rejected before storage: status=%d copied=%v", deniedCopy.Code, copied)
	}
	copyRequest := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale-copy?locale=fr", strings.NewReader("_csrf=csrf-value&from_locale=en&expected_version=2"))
	copyRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	copyRequest.Header.Set("X-Test-Role", "admin")
	copyResponse := httptest.NewRecorder()
	mux.ServeHTTP(copyResponse, copyRequest)
	if copyResponse.Code != http.StatusSeeOther || !copied {
		t.Fatalf("authorized locale copy was not delegated with the current locale version: status=%d copied=%v body=%s", copyResponse.Code, copied, copyResponse.Body.String())
	}
	for _, role := range []string{"editor", "viewer"} {
		localizedWrite := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale?locale=fr", strings.NewReader("_csrf=csrf-value&locale_version=2&title=Forged"))
		localizedWrite.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		localizedWrite.Header.Set("X-Test-Role", role)
		localizedWriteResponse := httptest.NewRecorder()
		mux.ServeHTTP(localizedWriteResponse, localizedWrite)
		if localizedWriteResponse.Code != http.StatusForbidden || saved["title"] != "Nouveau" {
			t.Fatalf("%s localized write was not denied before persistence: status=%d fields=%#v", role, localizedWriteResponse.Code, saved)
		}
		localeCopy := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/locale-copy?locale=fr", strings.NewReader("_csrf=csrf-value&from_locale=en&expected_version=2"))
		localeCopy.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		localeCopy.Header.Set("X-Test-Role", role)
		localeCopyResponse := httptest.NewRecorder()
		mux.ServeHTTP(localeCopyResponse, localeCopy)
		if localeCopyResponse.Code != http.StatusForbidden || !copied {
			t.Fatalf("%s locale copy was not denied before persistence: status=%d copied=%v", role, localeCopyResponse.Code, copied)
		}
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
	apiCalls := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/project" {
			t.Errorf("unexpected resource API call %s %s", r.Method, r.URL.String())
		}
		if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
			t.Errorf("decode forwarded create: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux := testGeneratedAdmin(t, api, func(*http.Request) bool { return true })
	for _, tc := range []struct {
		name string
		path string
		form url.Values
	}{
		{name: "create", path: "/admin/resources/project/new", form: url.Values{"id": {"p-2"}, "title": {"New project"}, "status": {"draft"}}},
		{name: "update", path: "/admin/resources/project/p-2", form: url.Values{"title": {"Changed project"}, "status": {"draft"}}},
		{name: "delete", path: "/admin/resources/project/p-2/delete", form: url.Values{"_csrf": {"csrf-value"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden || apiCalls != 0 {
				t.Fatalf("mutation without valid CSRF status=%d, API calls=%d", response.Code, apiCalls)
			}
		})
	}

	values := url.Values{"id": {"p-2"}, "title": {"New project"}, "status": {"draft"}, "_csrf": {"csrf-value"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/project/new", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || apiCalls != 1 || created["id"] != "p-2" || created["title"] != "New project" || created["status"] != "draft" {
		t.Fatalf("resource create was not forwarded: status=%d calls=%d created=%#v", response.Code, apiCalls, created)
	}
}

func TestGeneratedBasicCRUDUsesEditionRoleCapabilities(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title",
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText, Required: true}},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	apiMutations := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			t.Errorf("unexpected read while testing direct writes: %s %s", r.Method, r.URL.Path)
		}
		apiMutations++
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut, http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected generated API method %s", r.Method)
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api,
		Authorize: func(r *http.Request) bool {
			role := r.Header.Get("X-Test-Role")
			return role == "admin" || role == "editor" || role == "viewer"
		},
		CanReadResource: func(*http.Request, Collection) bool { return true },
		ResourceAccess: func(r *http.Request, _ Collection, id string) ResourceAccess {
			switch r.Header.Get("X-Test-Role") {
			case "admin":
				return ResourceAccess{Create: true, Update: id != "", Delete: id != ""}
			case "editor":
				return ResourceAccess{Create: true, Update: id != ""}
			default:
				return ResourceAccess{}
			}
		},
		CSRFToken:    func(*http.Request) string { return "csrf-value" },
		ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	actions := []struct {
		name string
		path string
		body string
		want int
	}{
		{"create", "/admin/resources/article/new", "id=a1&title=Created&_csrf=csrf-value", 0},
		{"update", "/admin/resources/article/a1", "title=Updated&_csrf=csrf-value", 1},
		{"delete", "/admin/resources/article/a1/delete", "_csrf=csrf-value", 2},
	}
	for _, role := range []string{"admin", "editor", "viewer"} {
		for _, action := range actions {
			t.Run(role+"/"+action.name, func(t *testing.T) {
				before := apiMutations
				request := httptest.NewRequest(http.MethodPost, action.path, strings.NewReader(action.body))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Set("X-Test-Role", role)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				allowed := role == "admin" || role == "editor" && action.name != "delete"
				wantStatus := http.StatusForbidden
				wantMutations := before
				if allowed {
					wantStatus = http.StatusSeeOther
					wantMutations++
				}
				if response.Code != wantStatus || apiMutations != wantMutations {
					t.Fatalf("%s %s status=%d mutations=%d, want status=%d mutations=%d body=%s", role, action.name, response.Code, apiMutations, wantStatus, wantMutations, response.Body.String())
				}
			})
		}
	}
}

func TestGeneratedAdminAllMutationRoutesRejectInvalidCSRFBeforePersistence(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title", Versioned: true, Revisions: true, Localization: true, Drafts: true,
		Workflow: &Workflow{StateField: "status", States: []PublicationState{StateDraft, "review", StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {"review"}, "review": {StatePublished}}},
		Fields:   []Field{{ID: "title", Label: "Title", Kind: FieldText, Localized: true, VisibleInList: true}, {ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "review", "published"}}},
		Actions:  []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	apiCalls, persistenceCalls := 0, 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"a1","version":1,"title":"Article","status":"published"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return true },
		ResourceAccess: func(*http.Request, Collection, string) ResourceAccess {
			return ResourceAccess{Create: true, Update: true, Delete: true}
		},
		CSRFToken:    func(*http.Request) string { return "csrf" },
		ValidateCSRF: func(*http.Request) bool { return false },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
		CanTransition: func(*http.Request, Collection, map[string]any, PublicationState) bool {
			persistenceCalls++
			return true
		},
		Transition: func(*http.Request, Collection, map[string]any, PublicationState, int64) error {
			persistenceCalls++
			return nil
		},
		CanRestore: func(*http.Request, Collection, string, int64) bool { persistenceCalls++; return true },
		ListRevisions: func(*http.Request, Collection, string, int) ([]RevisionSummary, error) {
			return nil, nil
		},
		GetRevision: func(*http.Request, Collection, string, int64) (RevisionSnapshot, error) {
			return RevisionSnapshot{}, nil
		},
		RestoreRevision: func(*http.Request, Collection, string, int64, int64) error {
			persistenceCalls++
			return nil
		},
		Locales: func(*http.Request, Collection) (LocaleOptions, error) {
			return LocaleOptions{Default: "en", Allowed: []string{"en", "fr"}}, nil
		},
		GetLocalized: func(*http.Request, Collection, string, string) (map[string]any, int64, error) {
			persistenceCalls++
			return map[string]any{"title": "Article"}, 1, nil
		},
		SaveLocalized: func(*http.Request, Collection, string, string, map[string]any, int64) error {
			persistenceCalls++
			return nil
		},
		ListLocalized: func(*http.Request, Collection, []string, string) (map[string]map[string]any, error) {
			return nil, nil
		},
		CanCopyLocale: func(*http.Request, Collection, string, string) bool { persistenceCalls++; return true },
		CopyLocale: func(*http.Request, Collection, string, string, string, int64) error {
			persistenceCalls++
			return nil
		},
		ListDrafts: func(*http.Request, Collection, string, int) ([]DraftDocument, error) { return nil, nil },
		GetDraft: func(*http.Request, Collection, string, string) (DraftDocument, error) {
			persistenceCalls++
			return DraftDocument{}, nil
		},
		CanSaveDraft: func(*http.Request, Collection, string, string) bool {
			persistenceCalls++
			return true
		},
		SaveDraft: func(*http.Request, Collection, string, string, map[string]any, int64, int64) error {
			persistenceCalls++
			return nil
		},
		CanTransitionDraft: func(*http.Request, Collection, DraftDocument, PublicationState) bool {
			persistenceCalls++
			return true
		},
		TransitionDraft: func(*http.Request, Collection, string, string, PublicationState, int64, int64) error {
			persistenceCalls++
			return nil
		},
		CanDiscardDraft: func(*http.Request, Collection, DraftDocument) bool {
			persistenceCalls++
			return true
		},
		DiscardDraft: func(*http.Request, Collection, string, string, int64) error {
			persistenceCalls++
			return nil
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{name: "create", path: "/admin/resources/article/new", body: "id=a2&draft_version=0&published_version=0&title=New"},
		{name: "update", path: "/admin/resources/article/a1", body: "title=Changed"},
		{name: "delete", path: "/admin/resources/article/a1/delete", body: "_csrf=bad"},
		{name: "workflow", path: "/admin/resources/article/a1/workflow", body: "target=review&expected_draft_version=1&expected_published_version=1"},
		{name: "revision restore", path: "/admin/resources/article/a1/revisions/1/restore", body: "expected_version=1"},
		{name: "locale save", path: "/admin/resources/article/a1/locale?locale=fr", body: "locale_version=1&title=Traduction"},
		{name: "locale copy", path: "/admin/resources/article/a1/locale-copy?locale=fr", body: "from_locale=en&expected_version=1"},
		{name: "draft save", path: "/admin/resources/article/a1/draft", body: "draft_version=0&published_version=1&title=Draft"},
		{name: "draft discard", path: "/admin/resources/article/a1/draft/discard", body: "expected_version=1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beforeAPI, beforePersistence := apiCalls, persistenceCalls
			request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("invalid CSRF status=%d, want 403; body=%s", response.Code, response.Body.String())
			}
			if apiCalls != beforeAPI || persistenceCalls != beforePersistence {
				t.Fatalf("invalid CSRF reached persistence boundary: API calls %d->%d, callbacks %d->%d", beforeAPI, apiCalls, beforePersistence, persistenceCalls)
			}
		})
	}
}

func TestGeneratedAdminRequiresEditionAuthorizationAndBoundaries(t *testing.T) {
	registry, _ := NewRegistry([]Collection{{ID: "project", Label: "Projects", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Open", Href: "/admin/resources/project"}}}})
	if err := (GeneratedAdminHandler{Registry: registry}).Register(http.NewServeMux()); err == nil {
		t.Fatal("registration without Edition security boundaries succeeded")
	}
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized request reached resource API") })
	client, baseURL := generatedAdminClient(t, testGeneratedAdmin(t, api, func(*http.Request) bool { return false }))
	response, err := client.Get(baseURL + "/admin/resources/project")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthorized generated Admin request status=%d", response.StatusCode)
	}
}

func TestGeneratedAdminResourceReadPolicyHidesIndexAndBlocksCollectionRoutes(t *testing.T) {
	registry, err := NewRegistry([]Collection{{ID: "article", Label: "Articles", Fields: []Field{{ID: "title", Label: "Title", Kind: FieldText}}, Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}}}})
	if err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("read-denied resource reached the API") })
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return false },
		CSRFToken:       func(*http.Request) string { return "csrf" },
		ValidateCSRF:    func(*http.Request) bool { return true },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	index := httptest.NewRecorder()
	mux.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/admin/resources", nil))
	if index.Code != http.StatusOK || strings.Contains(index.Body.String(), "/admin/resources/article") {
		t.Fatalf("resource index exposed a read-denied collection: status=%d body=%s", index.Code, index.Body.String())
	}
	for _, path := range []string{"/admin/resources/article", "/admin/resources/article/new", "/admin/resources/article/a1"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("read-denied route %s status=%d, want 404", path, response.Code)
		}
	}
}

func TestGeneratedAdminSurfacesResourceAPIDenialWithoutSuccessRedirect(t *testing.T) {
	calls := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/project" {
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	mux := testGeneratedAdmin(t, api, func(*http.Request) bool { return true })
	request := httptest.NewRequest(http.MethodPost, "/admin/resources/project/new", strings.NewReader("title=Denied"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Test", "valid")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "Bu işlem için yetkiniz yok.") || response.Header().Get("Location") != "" || calls != 1 {
		t.Fatalf("API write denial was not presented safely: status=%d location=%q calls=%d body=%s", response.Code, response.Header().Get("Location"), calls, response.Body.String())
	}
}

func TestGeneratedAdminResourceCapabilitiesHideAndDenyBasicCRUD(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title",
		Fields: []Field{
			{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true},
			{ID: "internal_note", Label: "Internal note", Kind: FieldText},
		},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	apiCalls, mutations := 0, 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		if r.Method != http.MethodGet {
			mutations++
			t.Errorf("resource mutation passed a denied Admin capability: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/article" {
			_, _ = io.WriteString(w, `[{"id":"a1","title":"Read-only article"}]`)
			return
		}
		_, _ = io.WriteString(w, `{"id":"a1","title":"<script>bad()</script>","internal_note":"Do not expose"}`)
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return true },
		ResourceAccess:  func(_ *http.Request, _ Collection, _ string) ResourceAccess { return ResourceAccess{} },
		CSRFToken:       func(*http.Request) string { return "csrf" },
		ValidateCSRF:    func(*http.Request) bool { return true },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil))
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), "Yeni kayıt") || strings.Contains(list.Body.String(), `href="/admin/resources/article/a1"`) || !strings.Contains(list.Body.String(), "Read-only article") {
		t.Fatalf("read-only list exposed CRUD affordances or hid content: status=%d body=%s", list.Code, list.Body.String())
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		create := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/admin/resources/article/new", strings.NewReader("title=Denied"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		mux.ServeHTTP(create, request)
		if create.Code != http.StatusForbidden {
			t.Errorf("create %s with no capability status=%d, want 403", method, create.Code)
		}
	}
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1", nil))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "&lt;script&gt;bad()&lt;/script&gt;") || !strings.Contains(detail.Body.String(), "Listeye dön") || strings.Contains(detail.Body.String(), "<form") || strings.Contains(detail.Body.String(), "<script") || strings.Contains(detail.Body.String(), "Do not expose") {
		t.Fatalf("read-only record view is missing or exposed mutation markup: status=%d body=%s", detail.Code, detail.Body.String())
	}
	for _, path := range []string{"/admin/resources/article/a1", "/admin/resources/article/a1/delete"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("title=Denied"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("mutation %s with no capability status=%d, want 403", path, response.Code)
		}
	}
	if mutations != 0 || apiCalls != 2 {
		t.Fatalf("denied capability reached mutation API: api calls=%d mutations=%d", apiCalls, mutations)
	}
}

func TestGeneratedAdminResourceCapabilitiesAllowUpdateWithoutCreateOrDelete(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title",
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText, VisibleInList: true}},
		Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mutations := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/article":
			_, _ = io.WriteString(w, `[{"id":"a1","title":"Editable article"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/article/a1":
			_, _ = io.WriteString(w, `{"id":"a1","title":"Editable article"}`)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/article/a1":
			mutations++
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected resource API request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return true },
		ResourceAccess: func(_ *http.Request, _ Collection, id string) ResourceAccess {
			return ResourceAccess{Update: id != ""}
		},
		CSRFToken:    func(*http.Request) string { return "csrf" },
		ValidateCSRF: func(*http.Request) bool { return true },
		RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
			return body.Render(r.Context(), w)
		},
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/admin/resources/article", nil))
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), "Yeni kayıt") || !strings.Contains(list.Body.String(), `href="/admin/resources/article/a1"`) {
		t.Fatalf("update-only resource affordances are incorrect: status=%d body=%s", list.Code, list.Body.String())
	}
	edit := httptest.NewRecorder()
	mux.ServeHTTP(edit, httptest.NewRequest(http.MethodGet, "/admin/resources/article/a1", nil))
	if edit.Code != http.StatusOK || strings.Contains(edit.Body.String(), `action="/admin/resources/article/a1/delete"`) {
		t.Fatalf("update-only record edit/delete UI is incorrect: status=%d body=%s", edit.Code, edit.Body.String())
	}
	update := httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1", strings.NewReader("title=Updated"))
	update.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updated := httptest.NewRecorder()
	mux.ServeHTTP(updated, update)
	if updated.Code != http.StatusSeeOther || updated.Header().Get("Location") != "/admin/resources/article" || mutations != 1 {
		t.Fatalf("allowed update was not completed: status=%d location=%q API mutations=%d", updated.Code, updated.Header().Get("Location"), mutations)
	}
	deleted := httptest.NewRecorder()
	mux.ServeHTTP(deleted, httptest.NewRequest(http.MethodPost, "/admin/resources/article/a1/delete", strings.NewReader("")))
	if deleted.Code != http.StatusForbidden || mutations != 1 {
		t.Fatalf("denied delete reached resource API: status=%d mutations=%d", deleted.Code, mutations)
	}
}

func TestGeneratedAdminGlobalRequiresUpdateCapability(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "settings", Type: "global", Label: "Settings",
		Fields:  []Field{{ID: "title", Label: "Title", Kind: FieldText}},
		Actions: []Action{{Label: "Settings", Href: "/admin/resources/settings"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
		API:             http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("denied global settings reached API") }),
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return true },
		ResourceAccess:  func(*http.Request, Collection, string) ResourceAccess { return ResourceAccess{} },
		CSRFToken:       func(*http.Request) string { return "csrf" },
		ValidateCSRF:    func(*http.Request) bool { return true },
		RenderPage:      func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
	}}
	if err := handler.Register(mux); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/resources/settings", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("global settings ignored update capability: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestGeneratedAdminBlockFieldsRequireEditionAssetAndNonce(t *testing.T) {
	registry, err := NewRegistry([]Collection{{ID: "article", Label: "Articles", Fields: []Field{{ID: "body", Label: "Body", Kind: FieldBlocks}}, Actions: []Action{{Label: "Articles", Href: "/admin/resources/article"}}}})
	if err != nil {
		t.Fatal(err)
	}
	options := GeneratedAdminOptions{
		API: http.NotFoundHandler(), Authorize: func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, Collection) bool { return true }, CSRFToken: func(*http.Request) string { return "csrf" },
		ValidateCSRF: func(*http.Request) bool { return true },
		RenderPage:   func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
	}
	if err := (GeneratedAdminHandler{Registry: registry, Options: options}).Register(http.NewServeMux()); err == nil {
		t.Fatal("block editor registered without Edition asset and nonce")
	}
	options.BlockEditor = BlockEditorConfig{EditorScript: "https://editor.example/editor.js"}
	if err := (GeneratedAdminHandler{Registry: registry, Options: options}).Register(http.NewServeMux()); err == nil {
		t.Fatal("cross-origin editor asset was accepted")
	}
	options.BlockEditor.EditorScript = "/assets/editor.js"
	if err := (GeneratedAdminHandler{Registry: registry, Options: options}).Register(http.NewServeMux()); err != nil {
		t.Fatalf("valid Edition block editor composition rejected: %v", err)
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
	publishedReads, publishedWrites, saves, transitions, discards := 0, 0, 0, 0, 0
	allowSave, allowTransition, allowDiscard := true, true, true
	validCSRF := true
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
		CanSaveDraft: func(*http.Request, Collection, string, string) bool { return allowSave },
		SaveDraft: func(_ *http.Request, _ Collection, id, _ string, fields map[string]any, expectedDraft, expectedPublished int64) error {
			saves++
			if id != "a1" || expectedDraft != 2 || expectedPublished != 4 || fields["title"] != "Edited draft" {
				return errors.New("unexpected draft CAS")
			}
			return nil
		},
		CanTransitionDraft: func(_ *http.Request, _ Collection, got DraftDocument, target PublicationState) bool {
			return allowTransition && got.Version == 2 && target == "review"
		},
		TransitionDraft: func(_ *http.Request, _ Collection, id, _ string, target PublicationState, expectedDraft, expectedPublished int64) error {
			transitions++
			if id != "a1" || target != "review" || expectedDraft != 2 || expectedPublished != 4 {
				return errors.New("unexpected transition CAS")
			}
			return nil
		},
		CanDiscardDraft: func(*http.Request, Collection, DraftDocument) bool { return allowDiscard },
		DiscardDraft: func(_ *http.Request, _ Collection, id, _ string, expected int64) error {
			discards++
			if id != "a1" || expected != 2 {
				return errors.New("unexpected discard CAS")
			}
			return nil
		},
		CSRFToken: func(*http.Request) string { return "csrf-value" }, ValidateCSRF: func(r *http.Request) bool { return validCSRF && r.FormValue("_csrf") == "csrf-value" },
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
	if edit.Code != http.StatusOK || !strings.Contains(edit.Body.String(), "Working title") || !strings.Contains(edit.Body.String(), `action="/admin/resources/article/a1/draft"`) || !strings.Contains(edit.Body.String(), `data-confirm-action="Yalnızca yayımlanmamış taslak silinecek. Devam edilsin mi?"`) || !strings.Contains(edit.Body.String(), `name="draft_version" value="2"`) || !strings.Contains(edit.Body.String(), `name="expected_published_version" value="4"`) {
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
	validCSRF = false
	if got := post("/admin/resources/article/a1/draft", "_csrf=csrf-value&title=Edited+draft&status=draft&draft_version=2&published_version=4"); got.Code != http.StatusForbidden || saves != 0 {
		t.Fatalf("draft save without valid CSRF must be rejected before storage: status=%d saves=%d", got.Code, saves)
	}
	validCSRF = true
	allowSave = false
	if got := post("/admin/resources/article/a1/draft", "_csrf=csrf-value&title=Edited+draft&status=draft&draft_version=2&published_version=4"); got.Code != http.StatusForbidden || saves != 0 {
		t.Fatalf("unauthorized draft save must be rejected before storage: status=%d saves=%d", got.Code, saves)
	}
	allowSave = true
	if got := post("/admin/resources/article/a1/draft", "_csrf=csrf-value&title=Edited+draft&status=draft&draft_version=2&published_version=4"); got.Code != http.StatusSeeOther || saves != 1 {
		t.Fatalf("valid draft save was not delegated: status=%d saves=%d body=%s", got.Code, saves, got.Body.String())
	}
	if got := post("/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=1&expected_published_version=4"); got.Code != http.StatusConflict || transitions != 0 {
		t.Fatalf("stale draft must block transition: status=%d transitions=%d", got.Code, transitions)
	}
	allowTransition = false
	if got := post("/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=2&expected_published_version=4"); got.Code != http.StatusForbidden || transitions != 0 {
		t.Fatalf("unauthorized draft transition must be rejected before storage: status=%d transitions=%d", got.Code, transitions)
	}
	allowTransition = true
	validCSRF = false
	if got := post("/admin/resources/article/a1/workflow", "target=review&expected_draft_version=2&expected_published_version=4"); got.Code != http.StatusForbidden || transitions != 0 {
		t.Fatalf("draft transition without valid CSRF must be rejected before storage: status=%d transitions=%d", got.Code, transitions)
	}
	validCSRF = true
	if got := post("/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=2&expected_published_version=4"); got.Code != http.StatusSeeOther || transitions != 1 {
		t.Fatalf("valid transition was not delegated: status=%d transitions=%d body=%s", got.Code, transitions, got.Body.String())
	}
	allowDiscard = false
	if got := post("/admin/resources/article/a1/draft/discard", "_csrf=csrf-value&expected_version=2"); got.Code != http.StatusForbidden || discards != 0 {
		t.Fatalf("unauthorized draft discard must be rejected before storage: status=%d discards=%d", got.Code, discards)
	}
	allowDiscard = true
	validCSRF = false
	if got := post("/admin/resources/article/a1/draft/discard", "expected_version=2"); got.Code != http.StatusForbidden || discards != 0 {
		t.Fatalf("draft discard without valid CSRF must be rejected before storage: status=%d discards=%d", got.Code, discards)
	}
	validCSRF = true
	if got := post("/admin/resources/article/a1/draft/discard", "_csrf=csrf-value&expected_version=2"); got.Code != http.StatusSeeOther || discards != 1 {
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

func TestGeneratedEditorialMutationsRespectEditorAndViewerCapabilities(t *testing.T) {
	registry, err := NewRegistry([]Collection{{
		ID: "article", Label: "Articles", TitleField: "title", Versioned: true, Revisions: true, Drafts: true,
		Fields:   []Field{{ID: "title", Label: "Title", Kind: FieldText}, {ID: "status", Label: "Status", Kind: FieldSelect, Options: []string{"draft", "review", "published"}}},
		Actions:  []Action{{Label: "Articles", Href: "/admin/resources/article"}},
		Workflow: &Workflow{StateField: "status", States: []PublicationState{StateDraft, "review", StatePublished}, Transitions: map[PublicationState][]PublicationState{StateDraft: {"review"}, "review": {StatePublished}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	draft := DraftDocument{ID: "a1", Version: 2, PublishedVersion: 4, State: StateDraft, Fields: map[string]any{"title": "Working title"}, Exists: true}
	mutations := 0
	for _, role := range []string{"admin", "editor", "viewer"} {
		t.Run(role, func(t *testing.T) {
			mutations = 0
			mux := http.NewServeMux()
			handler := GeneratedAdminHandler{Registry: registry, Options: GeneratedAdminOptions{
				API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"id":"a1","title":"Published title","status":"published","version":4}`)
						return
					}
					t.Errorf("editorial mutation unexpectedly reached published API: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected", http.StatusInternalServerError)
				}),
				Authorize: func(r *http.Request) bool {
					got := r.Header.Get("X-Test-Role")
					return got == "admin" || got == "editor" || got == "viewer"
				},
				CanReadResource: func(*http.Request, Collection) bool { return true },
				ListDrafts: func(*http.Request, Collection, string, int) ([]DraftDocument, error) {
					return []DraftDocument{draft}, nil
				},
				GetDraft:     func(*http.Request, Collection, string, string) (DraftDocument, error) { return draft, nil },
				CanSaveDraft: func(r *http.Request, _ Collection, _, _ string) bool { return r.Header.Get("X-Test-Role") == "admin" },
				SaveDraft: func(*http.Request, Collection, string, string, map[string]any, int64, int64) error {
					mutations++
					return nil
				},
				CanTransitionDraft: func(r *http.Request, _ Collection, _ DraftDocument, _ PublicationState) bool {
					return r.Header.Get("X-Test-Role") == "admin"
				},
				TransitionDraft: func(*http.Request, Collection, string, string, PublicationState, int64, int64) error {
					mutations++
					return nil
				},
				CanDiscardDraft: func(r *http.Request, _ Collection, _ DraftDocument) bool {
					return r.Header.Get("X-Test-Role") == "admin"
				},
				DiscardDraft:  func(*http.Request, Collection, string, string, int64) error { mutations++; return nil },
				ListRevisions: func(*http.Request, Collection, string, int) ([]RevisionSummary, error) { return nil, nil },
				GetRevision: func(*http.Request, Collection, string, int64) (RevisionSnapshot, error) {
					return RevisionSnapshot{}, errors.New("missing")
				},
				CanRestore: func(*http.Request, Collection, string, int64) bool { return false },
				RestoreRevision: func(*http.Request, Collection, string, int64, int64) error {
					return errors.New("restore denied")
				},
				CSRFToken:    func(*http.Request) string { return "csrf-value" },
				ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "csrf-value" },
				RenderPage: func(w http.ResponseWriter, r *http.Request, _ string, body templ.Component) error {
					return body.Render(r.Context(), w)
				},
			}}
			if err := handler.Register(mux); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				path string
				body string
			}{
				{"/admin/resources/article/a1/draft", "_csrf=csrf-value&draft_version=2&published_version=4&title=Updated&status=draft"},
				{"/admin/resources/article/a1/workflow", "_csrf=csrf-value&target=review&expected_draft_version=2&expected_published_version=4"},
				{"/admin/resources/article/a1/draft/discard", "_csrf=csrf-value&expected_version=2"},
			} {
				req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("X-Test-Role", role)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, req)
				want := http.StatusForbidden
				if role == "admin" {
					want = http.StatusSeeOther
				}
				if response.Code != want {
					t.Errorf("%s %s status=%d, want %d: %s", role, tc.path, response.Code, want, response.Body.String())
				}
			}
			wantMutations := 0
			if role == "admin" {
				wantMutations = 3
			}
			if mutations != wantMutations {
				t.Errorf("%s editorial persistence callbacks=%d, want %d", role, mutations, wantMutations)
			}
		})
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
