//go:build !hydro_no_graphql

package main

import (
	"database/sql"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestGraphQLBuildFeature(t *testing.T) {
	handler, enabled := newGraphQLHandler((*sql.DB)(nil), service.ContentService{}, true)
	if handler == nil || !enabled {
		t.Fatal("default build must retain the GraphQL handler")
	}
}
