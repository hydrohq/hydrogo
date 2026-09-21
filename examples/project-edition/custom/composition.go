package custom

import (
	"context"
	"database/sql"
	"net/http"

	generated "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated/project"
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
	repository := generated.ProjectRepository{DB: db, Dialect: dialect}
	service := generated.NewProjectService(repository)
	return generated.NewProjectHTTPHandler(service, demoPolicy{})
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool   { return true }
func (demoPolicy) CanCreate(context.Context) bool         { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool { return true }
func (demoPolicy) CanDelete(context.Context, string) bool { return true }
