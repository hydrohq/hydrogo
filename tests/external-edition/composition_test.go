package externaledition

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
)

func TestExternalEditionCanComposeGeneratedAdmin(t *testing.T) {
	manifest := []byte(`{"version":"v1","resources":[{"id":"projects","label":"Projects","fields":[{"id":"title","label":"Title","kind":"text","required":true,"visible_in_list":true}],"actions":[{"label":"Open","href":"/admin/projects"}]},{"id":"disabled","label":"Disabled","fields":[{"id":"title","label":"Title","kind":"text"}],"actions":[{"label":"Open","href":"/admin/disabled"}]}]}`)
	mux := http.NewServeMux()
	options := adminstudio.GeneratedAdminOptions{
		API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/projects" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		}),
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "edition-token" },
		ValidateCSRF:    func(*http.Request) bool { return true },
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "edition-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}
	if err := adminstudio.RegisterGenerated(mux, manifest, []string{"projects"}, options); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/resources/projects", nil))
	for _, want := range []string{`<aside class="studio-sidebar">`, `data-studio-role="admin"`, `href="/admin/resources/projects"`, "Projects"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("shared generated Admin shell missing %q", want)
		}
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Yeni kayıt") {
		t.Fatalf("registered generated Admin route did not render: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	inactive := httptest.NewRecorder()
	mux.ServeHTTP(inactive, httptest.NewRequest(http.MethodGet, "/admin/resources/disabled", nil))
	if inactive.Code != http.StatusNotFound {
		t.Fatalf("unselected generated resource route was registered: status=%d body=%s", inactive.Code, inactive.Body.String())
	}
}
