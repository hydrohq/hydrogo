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
	_ "modernc.org/sqlite"
)

type resourcePolicy struct{}

func (resourcePolicy) CanRead(context.Context, string) bool   { return true }
func (resourcePolicy) CanCreate(context.Context) bool         { return true }
func (resourcePolicy) CanUpdate(context.Context, string) bool { return true }
func (resourcePolicy) CanDelete(context.Context, string) bool { return true }

func main() {
	address := flag.String("address", "127.0.0.1:8099", "loopback listen address")
	dsn := flag.String("dsn", "", "SQLite database path")
	flag.Parse()

	host, _, err := net.SplitHostPort(*address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		log.Fatal("test server requires a loopback listen address")
	}
	if strings.TrimSpace(*dsn) == "" {
		log.Fatal("SQLite DSN is required")
	}
	db, err := sql.Open("sqlite", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), generatedarticle.MigrationSQL); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	if err := generated.RegisterGeneratedResources(mux, generated.GeneratedResourceOptions{
		DB: db, Dialect: "sqlite",
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
			return r.Header.Get("X-Test-Role") == "admin"
		},
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "external-edition-test-token" },
		ValidateCSRF:    func(r *http.Request) bool { return r.FormValue("_csrf") == "external-edition-test-token" },
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "test-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
	}); err != nil {
		log.Fatal(err)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	server := &http.Server{Addr: *address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
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
