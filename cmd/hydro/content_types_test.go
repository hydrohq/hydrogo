package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
	types, routes, err := contentTypeRoutes(ctx, filepath.Join("..", "..", "content-types"), service, nil, contenttypes.NewLocales(true, "en", []string{"en", "de"}), nil, nil, nil)
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

func TestRateLimitedPublicContentTripsPerCaller(t *testing.T) {
	limiter := security.NewAttemptLimiter(64, 2, time.Minute)
	calls := 0
	handler := rateLimitedPublicContent(limiter, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/content/products?search=a", nil)
		r.RemoteAddr = "203.0.113.9:1234"
		return r
	}
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req())
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: status %d, want 200", i, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req())
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third call: status %d, want 429", rec.Code)
	}
	if calls != 2 {
		t.Fatalf("handler called %d times, want 2 (the third must be rejected before reaching it)", calls)
	}
}
