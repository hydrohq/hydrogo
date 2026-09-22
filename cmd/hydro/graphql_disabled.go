//go:build hydro_no_graphql

package main

import (
	"database/sql"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func newGraphQLHandler(_ *sql.DB, _ service.ContentService, _ bool) (http.Handler, bool) {
	return nil, false
}
