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
	repository := generated.ProjectRepository{DB: db, Dialect: "sqlite"}
	service := generated.NewProjectService(repository)
	return generated.NewProjectHTTPHandler(service, demoPolicy{})
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool              { return true }
func (demoPolicy) CanCreate(context.Context) bool                    { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool            { return true }
func (demoPolicy) CanDelete(context.Context, string) bool            { return true }

