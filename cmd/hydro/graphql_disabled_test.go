//go:build hydro_no_graphql

package main

import (
	"database/sql"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func TestGraphQLBuildFeatureIsAbsent(t *testing.T) {
	handler, enabled := newGraphQLHandler((*sql.DB)(nil), service.ContentService{}, true)
	if handler != nil || enabled {
		t.Fatal("hydro_no_graphql build must not expose a GraphQL handler")
	}
}
