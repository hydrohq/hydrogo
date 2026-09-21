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
	repository := generated.ArticleRepository{DB: db, Dialect: "sqlite"}
	service := generated.NewArticleService(repository)
	return generated.NewArticleHTTPHandler(service, demoPolicy{})
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool           { return true }
func (demoPolicy) CanCreate(context.Context) bool                 { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool         { return true }
func (demoPolicy) CanDelete(context.Context, string) bool         { return true }

