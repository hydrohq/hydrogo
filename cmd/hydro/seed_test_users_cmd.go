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
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// testRoleAccounts are the fixed, documented accounts "hydro generate tests"
// output logs in as; see internal/scaffold's e2e/README.md.
var testRoleAccounts = []struct {
	Role      auth.Role
	Email     string
	EnvPrefix string
}{
	{auth.RoleAdmin, "test-admin@hydrogo.local", "HYDRO_E2E_ADMIN"},
	{auth.RoleEditor, "test-editor@hydrogo.local", "HYDRO_E2E_EDITOR"},
	{auth.RoleViewer, "test-viewer@hydrogo.local", "HYDRO_E2E_VIEWER"},
}

// seedTestUsers ensures one account per built-in role exists with a known,
// freshly generated password, printing the env vars the generated Playwright
// specs expect. Idempotent: an existing account gets its password rotated
// and reprinted rather than erroring, so this is safe to rerun in CI.
func seedTestUsers(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("seed-test-users", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("seed-test-users takes no positional arguments")
	}

	cfg, err := loadMaintenanceConfig(flags, values)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.NewMobileAuthService(db, cfg.Database.Dialect), auth.DefaultArgon2Params)

	for _, account := range testRoleAccounts {
		password, err := randomPassword()
		if err != nil {
			return err
		}
		email := account.Email
		if _, err := authSvc.CreateUser(ctx, email, password, account.Role); err != nil {
			if !errors.Is(err, auth.ErrUserAlreadyExists) {
				return fmt.Errorf("seed %s test user: %w", account.Role, err)
			}
			existing, getErr := userStore.GetByEmail(ctx, auth.NormalizeEmail(email))
			if getErr != nil {
				return fmt.Errorf("seed %s test user: lookup: %w", account.Role, getErr)
			}
			hash, hashErr := auth.HashPassword(password, auth.DefaultArgon2Params)
			if hashErr != nil {
				return fmt.Errorf("seed %s test user: hash: %w", account.Role, hashErr)
			}
			if updateErr := userStore.UpdatePassword(ctx, db, existing.ID, hash); updateErr != nil {
				return fmt.Errorf("seed %s test user: update password: %w", account.Role, updateErr)
			}
		}
		fmt.Fprintf(stdout, "%s_EMAIL=%s\n%s_PASSWORD=%s\n", account.EnvPrefix, email, account.EnvPrefix, password)
	}
	return nil
}
