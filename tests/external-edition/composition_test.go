package externaledition

import (
	"net/http"
	"testing"

	"github.com/a-h/templ"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
)

func TestExternalEditionCanComposeGeneratedAdmin(t *testing.T) {
	registry, err := adminstudio.ParseManifest([]byte(`{"version":"v1","resources":[{"id":"projects","label":"Projects","fields":[{"id":"title","label":"Title","kind":"text","required":true,"visible_in_list":true}],"actions":[{"label":"Open","href":"/admin/projects"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	admin := adminstudio.GeneratedAdminHandler{
		Registry: registry,
		Options: adminstudio.GeneratedAdminOptions{
			API:             http.NotFoundHandler(),
			Authorize:       func(*http.Request) bool { return true },
			CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
			CSRFToken:       func(*http.Request) string { return "edition-token" },
			ValidateCSRF:    func(*http.Request) bool { return true },
			RenderPage:      func(http.ResponseWriter, *http.Request, string, templ.Component) error { return nil },
		},
	}
	_ = admin
}
