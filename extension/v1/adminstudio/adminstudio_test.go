package adminstudio

import (
	"net/http"
	"testing"

	"github.com/a-h/templ"
)

func TestPublicEditionCompositionContract(t *testing.T) {
	var _ func([]byte) (Registry, error) = ParseManifest
	var renderPage func(http.ResponseWriter, *http.Request, string, templ.Component) error
	var _ func(http.ResponseWriter, *http.Request, string, templ.Component) error = renderPage
	var _ GeneratedAdminHandler = GeneratedAdminHandler{
		Options: GeneratedAdminOptions{
			Authorize:       func(*http.Request) bool { return false },
			CanReadResource: func(*http.Request, Collection) bool { return false },
			CSRFToken:       func(*http.Request) string { return "" },
			ValidateCSRF:    func(*http.Request) bool { return false },
			RenderPage:      func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
		},
	}
	if FieldText == "" || StateDraft == "" {
		t.Fatal("public Admin Studio enum contract is empty")
	}
}
