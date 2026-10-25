package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// createUser makes (or updates) a staff account that is not an administrator:
// a role plus the permissions the project's content types declare. It is the
// way to open the first accounts of a project before e-mail is configured,
// since an invitation needs SMTP and create-admin only makes administrators.
//
//	hydro create-user --email EMAIL --password-stdin [--role viewer|editor] [--permissions a,b]
func createUser(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("create-user", flag.ContinueOnError)
	flags.SetOutput(stderr)
	email := flags.String("email", "", "Account e-mail address")
	passwordStdin := flags.Bool("password-stdin", false, "Read the password from stdin")
	role := flags.String("role", "viewer", "Built-in role: viewer or editor (use create-admin for administrators)")
	permissions := flags.String("permissions", "", "Comma-separated permissions to grant on top of the role")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *email == "" || !*passwordStdin {
		return errors.New("--email and --password-stdin are required")
	}
	r := auth.Role(strings.ToLower(strings.TrimSpace(*role)))
	if r != auth.RoleViewer && r != auth.RoleEditor {
		return errors.New("--role must be viewer or editor (administrators are made with create-admin)")
	}
	password, err := readPassword(stdin)
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}
	overrides := map[string]bool{}
	var granted []string
	for _, p := range strings.Split(*permissions, ",") {
		if p = strings.TrimSpace(p); p != "" {
			overrides[p] = true
			granted = append(granted, p)
		}
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
	authSvc := auth.NewAuthService(userStore, authstore.NewSessionStore(db, cfg.Database.Dialect), auth.NewMobileAuthService(db, cfg.Database.Dialect), auth.DefaultArgon2Params)

	verb := "Created"
	user, err := authSvc.CreateUser(ctx, *email, password, r)
	if errors.Is(err, auth.ErrUserAlreadyExists) {
		// Running it again is how a password, a role or a permission set is corrected.
		existing, getErr := userStore.GetByEmail(ctx, auth.NormalizeEmail(*email))
		if getErr != nil || existing == nil {
			return fmt.Errorf("user lookup failed: %w", getErr)
		}
		if existing.Role == auth.RoleAdmin {
			return errors.New("that account is an administrator; change it with create-admin")
		}
		hash, hashErr := auth.HashPassword(password, auth.DefaultArgon2Params)
		if hashErr != nil {
			return fmt.Errorf("failed to hash password: %w", hashErr)
		}
		if err := userStore.UpdatePassword(ctx, db, existing.ID, hash); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
		if existing.Role != r {
			if err := authSvc.UpdateUserRole(ctx, existing.ID, r); err != nil {
				return fmt.Errorf("failed to update role: %w", err)
			}
		}
		user, err, verb = existing, nil, "Updated"
	}
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	// Like service accounts, names are stored as given: a permission no content
	// type declares is ignored by the server (it fails closed).
	if len(overrides) > 0 || verb == "Updated" {
		if err := userStore.ReplacePermissionOverrides(ctx, user.ID, overrides); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}
	fmt.Fprintf(stdout, "%s user: %s (role: %s, id: %s)\n", verb, user.Email, r, user.ID)
	if len(granted) > 0 {
		fmt.Fprintf(stdout, "Permissions: %s\n", strings.Join(granted, ", "))
	}
	return nil
}
