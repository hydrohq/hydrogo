package main

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestContentTypeRoutesRegisterNextToPublicLocaleRoutes(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "content.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}
	types, routes, err := contentTypeRoutes(ctx, filepath.Join("..", "..", "content-types"), service, nil, contenttypes.NewLocales(true, "en", []string{"en", "de"}), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) == 0 || len(routes) == 0 {
		t.Fatal("shipped content types produced no routes")
	}
	adminRoutes := 0
	for _, route := range routes {
		if strings.Contains(route.Pattern, contenttypes.AdminAPIPath) {
			adminRoutes++
			if route.Access != router.Protected {
				t.Fatalf("admin content route %q is not protected", route.Pattern)
			}
		}
	}
	if adminRoutes == 0 {
		t.Fatal("admin content routes missing")
	}
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	routes = append(routes, router.Route{Pattern: "GET /{locale}/", Access: router.Public, Handler: noop})
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("route registration panicked: %v", recovered)
		}
	}()
	if _, err := router.New(routes, func(*http.Request) bool { return true }); err != nil {
		t.Fatal(err)
	}
}
