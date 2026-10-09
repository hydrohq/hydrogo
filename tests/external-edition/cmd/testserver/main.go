// Command testserver runs the independent Edition fixture for browser tests.
// It deliberately binds to loopback and uses test-only identity headers.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	generated "example.com/external-edition/generated"
	generatedarticle "example.com/external-edition/generated/article"
	generatedmediaarticle "example.com/external-edition/generated/media_article"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type resourcePolicy struct{}

type roleContextKey struct{}

func roleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(roleContextKey{}).(string)
	return role
}

func (resourcePolicy) CanRead(ctx context.Context, _ string) bool {
	role := roleFromContext(ctx)
	return role == "admin" || role == "editor" || role == "viewer"
}
func canWrite(ctx context.Context) bool {
	role := roleFromContext(ctx)
	return role == "admin" || role == "editor"
}
func (resourcePolicy) CanCreate(ctx context.Context) bool { return canWrite(ctx) }
func (resourcePolicy) CanUpdate(ctx context.Context, id string) bool {
	return canWrite(ctx) && !strings.HasPrefix(id, "locked-")
}
func (resourcePolicy) CanDelete(ctx context.Context, id string) bool {
	return canWrite(ctx) && !strings.HasPrefix(id, "locked-")
}

func main() {
	address := flag.String("address", "127.0.0.1:8099", "loopback listen address")
	dsn := flag.String("dsn", "", "database DSN")
	dialect := flag.String("dialect", "sqlite", "database dialect (sqlite or postgres)")
	skipMigrate := flag.Bool("skip-migrate", false, "skip the one-time test fixture migration")
	flag.Parse()

	host, _, err := net.SplitHostPort(*address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		log.Fatal("test server requires a loopback listen address")
	}
	if strings.TrimSpace(*dsn) == "" {
		log.Fatal("database DSN is required")
	}
	driver := "sqlite"
	if *dialect == "postgres" {
		driver = "pgx"
	} else if *dialect != "sqlite" {
		log.Fatalf("unsupported database dialect %q", *dialect)
	}
	db, err := sql.Open(driver, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if !*skipMigrate {
		if _, err := db.ExecContext(context.Background(), generatedarticle.MigrationSQL); err != nil {
			log.Fatal(err)
		}
		if _, err := db.ExecContext(context.Background(), generatedmediaarticle.MigrationSQL); err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	registerFixtureMediaRoutes(mux)
	resourceMux := http.NewServeMux()
	if err := generated.RegisterGeneratedResources(resourceMux, generated.GeneratedResourceOptions{
		DB: db, Dialect: *dialect,
		PolicyFor: func(string) generated.GeneratedResourcePolicy { return resourcePolicy{} },
		ResolveTenant: func(resourceID string, r *http.Request) (string, bool) {
			if resourceID != "article" && resourceID != "media_article" {
				return "", false
			}
			tenant := r.Header.Get("X-Test-Tenant")
			return tenant, tenant != ""
		},
	}); err != nil {
		log.Fatal(err)
	}
	validatedResourceAPI := validateFixtureMediaReference(resourceMux)
	resourceAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Header.Get("X-Test-Generated-API-Status") == "503" {
			http.Error(w, "fixture service unavailable", http.StatusServiceUnavailable)
			return
		}
		validatedResourceAPI.ServeHTTP(w, r)
	})
	mux.Handle("/api/v1/", resourceAPI)
	if err := generated.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: resourceAPI,
		Authorize: func(r *http.Request) bool {
			role := r.Header.Get("X-Test-Role")
			return role == "admin" || role == "editor" || role == "viewer"
		},
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ResourceAccess: func(r *http.Request, _ adminstudio.Collection, id string) adminstudio.ResourceAccess {
			if r.Header.Get("X-Test-Role") == "viewer" {
				return adminstudio.ResourceAccess{}
			}
			canMutateRecord := id != "" && !strings.HasPrefix(id, "locked-")
			return adminstudio.ResourceAccess{Create: id == "", Update: canMutateRecord, Delete: canMutateRecord}
		},
		CSRFToken:    func(*http.Request) string { return "external-edition-test-token" },
		ValidateCSRF: func(r *http.Request) bool { return r.FormValue("_csrf") == "external-edition-test-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(r *http.Request) adminstudio.PageIdentity {
			if r.Header.Get("X-Test-Role") == "editor" {
				return adminstudio.PageIdentity{UserID: "test-editor", Email: "editor@example.test", Role: adminstudio.PageRoleEditor}
			}
			if r.Header.Get("X-Test-Role") == "viewer" {
				return adminstudio.PageIdentity{UserID: "test-viewer", Email: "viewer@example.test", Role: adminstudio.PageRoleViewer}
			}
			return adminstudio.PageIdentity{UserID: "test-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}); err != nil {
		log.Fatal(err)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	withTestRole := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), roleContextKey{}, r.Header.Get("X-Test-Role"))
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
	server := &http.Server{Addr: *address, Handler: withTestRole, ReadHeaderTimeout: 5 * time.Second}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()
	go func() {
		<-stop.Done()
		_ = server.Shutdown(context.Background())
	}()
	fmt.Printf("edition-test-server-ready address=%s\n", listener.Addr())
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func fixtureMediaID(tenant string) string {
	digest := sha256.Sum256([]byte(tenant))
	return "fixture-" + hex.EncodeToString(digest[:8])
}

func registerFixtureMediaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/media", func(w http.ResponseWriter, r *http.Request) {
		if !fixtureRoleAllowed(r.Header.Get("X-Test-Role")) || strings.TrimSpace(r.Header.Get("X-Test-Tenant")) == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]string{{
			"id": fixtureMediaID(r.Header.Get("X-Test-Tenant")), "status": "ready",
			"mime_type": "image/svg+xml", "original_filename": "edition-cover.svg",
		}})
	})
	mux.HandleFunc("GET /media/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !fixtureRoleAllowed(r.Header.Get("X-Test-Role")) || strings.TrimSpace(r.Header.Get("X-Test-Tenant")) == "" || r.PathValue("id") != fixtureMediaID(r.Header.Get("X-Test-Tenant")) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 24"><rect width="32" height="24" fill="#dbeafe"/></svg>`)
	})
}

func fixtureRoleAllowed(role string) bool {
	return role == "admin" || role == "editor" || role == "viewer"
}

func validateFixtureMediaReference(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, generatedmediaarticle.MediaArticleRoute) && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			var payload struct {
				CoverMediaID string `json:"cover_media_id"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if payload.CoverMediaID != "" && payload.CoverMediaID != fixtureMediaID(r.Header.Get("X-Test-Tenant")) {
				http.Error(w, "media reference is not available to this tenant", http.StatusUnprocessableEntity)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(body)))
		}
		next.ServeHTTP(w, r)
	})
}
