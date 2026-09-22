package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestGraphQLRoutesRespectBuildCapability(t *testing.T) {
	limiter := security.NewAttemptLimiter(10, 10, time.Minute)
	if routes := graphQLRoutes(nil, limiter); len(routes) != 0 {
		t.Fatalf("missing GraphQL handler registered routes: %+v", routes)
	}
	routes := graphQLRoutes(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), limiter)
	if len(routes) != 1 || routes[0].Pattern != "POST /api/graphql" {
		t.Fatalf("enabled GraphQL capability routes = %+v", routes)
	}
}
