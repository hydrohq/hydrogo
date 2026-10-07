package custom

import (
	"context"
	"database/sql"
	"net/http"

	generatedadmin "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated"
	generated "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated/project"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
)

// NewDemoHandler wires generated persistence and transport for local contract
// tests only. A real Edition must replace demoPolicy with tenant/member-aware
// authorization before exposing the handler.
func NewDemoHandler(db *sql.DB) http.Handler {
	return NewHandler(db, "sqlite")
}

// NewHandler is the Edition composition point. The caller owns the database
// lifecycle and must pass the dialect selected by the deployment profile.
func NewHandler(db *sql.DB, dialect string) http.Handler {
	mux := http.NewServeMux()
	if err := generated.RegisterProjectRoute(mux, db, dialect, demoPolicy{}); err != nil {
		panic(err)
	}
	if err := generatedadmin.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: mux, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken: func(*http.Request) string { return "edition-demo-csrf" },
		ValidateCSRF: func(r *http.Request) bool {
			return r.ParseForm() == nil && r.FormValue("_csrf") == "edition-demo-csrf"
		},
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "demo-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}); err != nil {
		panic(err)
	}
	return mux
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool   { return true }
func (demoPolicy) CanCreate(context.Context) bool         { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool { return true }
func (demoPolicy) CanDelete(context.Context, string) bool { return true }
