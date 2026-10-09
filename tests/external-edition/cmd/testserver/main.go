// Command testserver runs the independent Edition fixture for browser tests.
// It deliberately binds to loopback and uses test-only identity headers.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
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
func (resourcePolicy) CanUpdate(ctx context.Context, _ string) bool {
	return canWrite(ctx)
}
func (resourcePolicy) CanDelete(ctx context.Context, _ string) bool {
	return canWrite(ctx)
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
	}

	mux := http.NewServeMux()
	if err := generated.RegisterGeneratedResources(mux, generated.GeneratedResourceOptions{
		DB: db, Dialect: *dialect,
		PolicyFor: func(string) generated.GeneratedResourcePolicy { return resourcePolicy{} },
		ResolveTenant: func(_ string, r *http.Request) (string, bool) {
			tenant := r.Header.Get("X-Test-Tenant")
			return tenant, tenant != ""
		},
	}); err != nil {
		log.Fatal(err)
	}
	if err := generated.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: mux,
		Authorize: func(r *http.Request) bool {
			role := r.Header.Get("X-Test-Role")
			return role == "admin" || role == "editor"
		},
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "external-edition-test-token" },
		ValidateCSRF:    func(r *http.Request) bool { return r.FormValue("_csrf") == "external-edition-test-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(r *http.Request) adminstudio.PageIdentity {
			if r.Header.Get("X-Test-Role") == "editor" {
				return adminstudio.PageIdentity{UserID: "test-editor", Email: "editor@example.test", Role: adminstudio.PageRoleEditor}
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
