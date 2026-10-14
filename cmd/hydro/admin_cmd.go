package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func createAdmin(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	email := flags.String("email", "", "Admin email address")
	passwordStdin := flags.Bool("password-stdin", false, "Read admin password from stdin")
	values := addRuntimeFlags(flags)

	if err := flags.Parse(args); err != nil {
		return err
	}
	if *email == "" || !*passwordStdin {
		return errors.New("--email and --password-stdin are required")
	}
	password, err := readPassword(stdin)
	if err != nil {
		return fmt.Errorf("failed to read admin password: %w", err)
	}
	if password == "" {
		return errors.New("admin password cannot be empty")
	}

	cfg, err := loadRuntimeConfig(flags, values)
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

	u, err := authSvc.CreateUser(ctx, *email, password, auth.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	fmt.Fprintf(stdout, "Created admin user: %s (id: %s)\n", u.Email, u.ID)
	return nil
}

func readPassword(stdin io.Reader) (string, error) {
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
