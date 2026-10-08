package adminstudio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
)

func TestPublicEditionCompositionContract(t *testing.T) {
	var _ func(*http.Request, Collection, string) ResourceAccess = func(_ *http.Request, _ Collection, _ string) ResourceAccess {
		return ResourceAccess{Create: true, Update: true, Delete: true}
	}
	var _ func([]byte) (Registry, error) = ParseManifest
	var renderPage func(http.ResponseWriter, *http.Request, string, templ.Component) error
	var _ func(http.ResponseWriter, *http.Request, string, templ.Component) error = renderPage
	var _ GeneratedAdminHandler = GeneratedAdminHandler{
		Options: GeneratedAdminOptions{
			Authorize:       func(*http.Request) bool { return false },
			CanReadResource: func(*http.Request, Collection) bool { return false },
			ResourceAccess:  func(*http.Request, Collection, string) ResourceAccess { return ResourceAccess{} },
			CSRFToken:       func(*http.Request) string { return "" },
			ValidateCSRF:    func(*http.Request) bool { return false },
			RenderPage:      func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
		},
	}
	if FieldText == "" || StateDraft == "" {
		t.Fatal("public Admin Studio enum contract is empty")
	}
}

func TestStudioPageRendererRejectsInvalidPresentationIdentity(t *testing.T) {
	render := StudioPageRenderer(func(*http.Request) PageIdentity {
		return PageIdentity{UserID: "user", Role: "owner"}
	})
	response := httptest.NewRecorder()
	err := render(response, httptest.NewRequest(http.MethodGet, "/admin/resources", nil), "Resources", templ.ComponentFunc(func(_ context.Context, _ io.Writer) error { return nil }))
	if err == nil || response.Body.Len() != 0 {
		t.Fatalf("invalid identity error=%v output=%q", err, response.Body.String())
	}
}
