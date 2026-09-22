//go:build !hydro_no_graphql

package main

import (
	"database/sql"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/modules/api/graphql"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func newGraphQLHandler(db *sql.DB, pages service.ContentService, pagesEnabled bool) (http.Handler, bool) {
	return graphql.Handler{DB: db, Pages: pages, PagesEnabled: pagesEnabled}, true
}
