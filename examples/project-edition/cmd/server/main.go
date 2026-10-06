package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	custom "github.com/ardaozyildirim/hydrogo/examples/project-edition/custom"
	generated "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated/project"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: edition-server migrate|serve [flags]")
	}
	switch args[0] {
	case "migrate":
		return migrate(args[1:])
	case "serve":
		return serve(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func flagsFor(name string, args []string) (database.Dialect, string, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	dialect := flags.String("dialect", "sqlite", "sqlite or postgres")
	dsn := flags.String("dsn", ".tmp/project-edition.sqlite", "database DSN")
	if err := flags.Parse(args); err != nil {
		return "", "", err
	}
	if flags.NArg() != 0 {
		return "", "", fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *dialect != "sqlite" && *dialect != "postgres" {
		return "", "", fmt.Errorf("unsupported dialect %q", *dialect)
	}
	return database.Dialect(*dialect), *dsn, nil
}

func migrate(args []string) error {
	dialect, dsn, err := flagsFor("edition migrate", args)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, dialect, dsn, database.OpenOptions{MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, generated.MigrationSQL); err != nil {
		return fmt.Errorf("apply reviewed project migration: %w", err)
	}
	fmt.Printf("migration-applied edition=project dialect=%s\n", dialect)
	return nil
}

func serve(args []string) error {
	flags := flag.NewFlagSet("edition serve", flag.ContinueOnError)
	dialect := flags.String("dialect", "sqlite", "sqlite or postgres")
	dsn := flags.String("dsn", ".tmp/project-edition.sqlite", "database DSN")
	address := flags.String("address", "127.0.0.1:8091", "listen address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*dialect != "sqlite" && *dialect != "postgres") {
		return errors.New("serve requires --dialect sqlite|postgres and no positional arguments")
	}
	db, err := database.OpenWithOptions(context.Background(), database.Dialect(*dialect), *dsn, database.OpenOptions{MaxOpenConns: 8, MaxIdleConns: 4})
	if err != nil {
		return err
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.Handle("/", custom.NewHandler(db, *dialect))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	server := &http.Server{Addr: *address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	return server.ListenAndServe()
}
