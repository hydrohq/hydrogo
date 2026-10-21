package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// serviceAccountDomain marks service accounts in audit logs and revisions:
// their e-mail address is service+<name>@hydrogo.invalid.
const serviceAccountDomain = "hydrogo.invalid"

var serviceAccountName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,39}$`)

// serviceAccountCommand creates and revokes service accounts: identities for
// side services (an accounting service, an importer) that call the admin API
// with a bearer token. A service account is a user nobody can sign in as (its
// password is random and never shown), holds exactly the permissions you
// give it on top of the viewer role, and its token expires.
//
//	hydro service-account create --name ledger --permissions deals.read,invoices.write [--expires-days 90]
//	hydro service-account revoke --name ledger
func serviceAccountCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || (args[0] != "create" && args[0] != "revoke") {
		return errors.New("usage: hydro service-account create|revoke --name NAME [--permissions a,b] [--expires-days N]")
	}
	action := args[0]
	flags := flag.NewFlagSet("service-account "+action, flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "Service name, lowercase letters, digits and dashes (e.g. ledger)")
	permissions := flags.String("permissions", "", "Comma-separated permissions to grant (create)")
	expiresDays := flags.Int("expires-days", 90, "Token lifetime in days, 1 to 365 (create)")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if !serviceAccountName.MatchString(*name) {
		return errors.New("--name must be 2 to 40 characters: lowercase letters, digits and dashes, starting with a letter")
	}
	if *expiresDays < 1 || *expiresDays > 365 {
		return errors.New("--expires-days must be between 1 and 365")
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
	authSvc := auth.NewAuthService(userStore, authstore.NewSessionStore(db, cfg.Database.Dialect), auth.NewMobileAuthService(db, cfg.Database.Dialect), auth.DefaultArgon2Params)
	authSvc.SetAPITokenStore(authstore.NewAPITokenStore(db, cfg.Database.Dialect))
	email := "service+" + *name + "@" + serviceAccountDomain

	if action == "revoke" {
		user, err := userStore.GetByEmail(ctx, email)
		if err != nil || user == nil {
			return fmt.Errorf("no service account %q", *name)
		}
		tokens, err := authSvc.ListAPITokens(ctx, user.ID)
		if err != nil {
			return err
		}
		for _, token := range tokens {
			if token.RevokedAt == nil {
				if err := authSvc.RevokeAPIToken(ctx, token.ID, user.ID); err != nil {
					return err
				}
			}
		}
		if err := userStore.SetActive(ctx, user.ID, false); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Revoked service account %s: tokens revoked, account disabled.\n", email)
		return nil
	}

	var granted []string
	overrides := map[string]bool{}
	for _, permission := range strings.Split(*permissions, ",") {
		if permission = strings.TrimSpace(permission); permission != "" {
			overrides[permission] = true
			granted = append(granted, permission)
		}
	}
	if len(overrides) == 0 {
		return errors.New("--permissions is required: name what the service may do (for example deals.read,invoices.write)")
	}
	secret := make([]byte, 33)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	// The password only has to satisfy the strength rules; nobody ever sees it.
	user, err := authSvc.CreateUser(ctx, email, "Svc-"+base64.RawURLEncoding.EncodeToString(secret)+"-1a", auth.RoleViewer)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			return fmt.Errorf("service account %q already exists (revoke it first to replace it)", *name)
		}
		return err
	}
	if err := userStore.ReplacePermissionOverrides(ctx, user.ID, overrides); err != nil {
		return err
	}
	expires := time.Now().UTC().Add(time.Duration(*expiresDays) * 24 * time.Hour)
	token, _, err := authSvc.IssueAPIToken(ctx, user.ID, "service:"+*name, auth.APITokenPersonal, nil, &expires)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Created service account %s\nPermissions: %s\nExpires: %s\n\nToken (shown once; send it as \"Authorization: Bearer <token>\"):\n%s\n",
		email, strings.Join(granted, ", "), expires.Format("2006-01-02"), token)
	fmt.Fprintln(stdout, "\nPermissions that no content type declares are ignored by the server (it fails closed).")
	return nil
}
