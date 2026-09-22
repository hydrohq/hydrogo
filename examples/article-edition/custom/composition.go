package custom

import (
	"context"
	"database/sql"
	"net/http"

	generated "github.com/ardaozyildirim/hydrogo/examples/article-edition/generated/article"
)

// NewDemoHandler is a local composition example; production editions must
// replace demoPolicy with tenant/member-aware authorization.
func NewDemoHandler(db *sql.DB) http.Handler {
	return NewHandler(db, "sqlite")
}

// NewHandler is the Edition composition point. The caller owns the database
// lifecycle and must pass the dialect selected by the deployment profile.
func NewHandler(db *sql.DB, dialect string) http.Handler {
	mux := http.NewServeMux()
	if err := generated.RegisterArticleRoute(mux, db, dialect, demoPolicy{}); err != nil {
		panic(err)
	}
	return mux
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool   { return true }
func (demoPolicy) CanCreate(context.Context) bool         { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool { return true }
func (demoPolicy) CanDelete(context.Context, string) bool { return true }
