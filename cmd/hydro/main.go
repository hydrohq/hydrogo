package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

const version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hydro:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("a command is required")
	}

	switch args[0] {
	case "version":
		if len(args) != 1 {
			return errors.New("version does not accept arguments")
		}
		_, err := fmt.Fprintln(stdout, version)
		return err
	case "serve":
		return serve(args[1:], stderr)
	case "migrate":
		return migrate(args[1:], stderr)
	case "create-admin":
		return createAdmin(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.DefaultArgon2Params)

	return server.Serve(ctx, server.Config{
		Address: cfg.HTTP.Address,
		AuthSvc: authSvc,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibility(checkCtx, db, cfg.Database.Dialect, migrations.FS)
		},
	})
}

func migrate(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return database.ApplyMigrations(ctx, db, cfg.Database.Dialect, migrations.FS)
}

func createAdmin(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	email := flags.String("email", "", "Admin email address")
	password := flags.String("password", "", "Admin password")
	values := addRuntimeFlags(flags)

	if err := flags.Parse(args); err != nil {
		return err
	}
	if *email == "" || *password == "" {
		return errors.New("--email and --password are required")
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.DefaultArgon2Params)

	u, err := authSvc.CreateUser(ctx, *email, *password, auth.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	fmt.Fprintf(stdout, "Created admin user: %s (id: %s)\n", u.Email, u.ID)
	return nil
}

type runtimeFlags struct {
	configPath      *string
	address         *string
	databaseDialect *string
	databaseDSN     *string
}

func addRuntimeFlags(flags *flag.FlagSet) runtimeFlags {
	return runtimeFlags{
		configPath:      flags.String("config", "", "YAML config file"),
		address:         flags.String("address", "", "HTTP listen address"),
		databaseDialect: flags.String("database-dialect", "", "sqlite or postgres"),
		databaseDSN:     flags.String("database-dsn", "", "database connection string"),
	}
}

func loadRuntimeConfig(flags *flag.FlagSet, values runtimeFlags) (config.Config, error) {
	set := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) { set[item.Name] = true })
	overrides := config.Overrides{}
	if set["address"] {
		overrides.HTTPAddress = values.address
	}
	if set["database-dialect"] {
		overrides.DatabaseDialect = values.databaseDialect
	}
	if set["database-dsn"] {
		overrides.DatabaseDSN = values.databaseDSN
	}
	return config.Load(*values.configPath, os.LookupEnv, overrides)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro migrate [runtime flags]
  hydro create-admin --email EMAIL --password PASSWORD [runtime flags]
  hydro serve [runtime flags]
  hydro help

Runtime flags:
  --config PATH
  --address HOST:PORT
  --database-dialect sqlite|postgres
  --database-dsn DSN

`)
}
