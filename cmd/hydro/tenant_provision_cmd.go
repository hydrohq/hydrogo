package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// tenantProvisionCommand deliberately accepts the initial password only over
// stdin. It is an operator action, never a public sign-up or payment flow.
func tenantProvisionCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant provision", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	tenantID := flags.String("tenant-id", "", "Tenant ID to provision (required)")
	adminEmail := flags.String("admin-email", "", "Initial tenant administrator email (required)")
	locale := flags.String("locale", "tr", "Default locale")
	timezone := flags.String("timezone", "Europe/Istanbul", "Default IANA timezone")
	passwordStdin := flags.Bool("admin-password-stdin", false, "Read the initial administrator password from stdin")
	jsonOutput := flags.Bool("json", false, "Print machine-readable JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !*passwordStdin {
		return errors.New("tenant provision requires --admin-password-stdin; passwords are never accepted as command arguments")
	}

	id := strings.TrimSpace(*tenantID)
	if _, err := tenant.NewScope(id, id); err != nil {
		return fmt.Errorf("invalid --tenant-id: %w", err)
	}
	email := auth.NormalizeEmail(*adminEmail)
	if !auth.ValidateEmail(email) {
		return errors.New("invalid --admin-email")
	}
	password, err := readProvisionPassword(stdin)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(password, auth.DefaultArgon2Params)
	if err != nil {
		return fmt.Errorf("hash initial administrator password: %w", err)
	}

	cfg, err := loadMaintenanceConfig(flags, values)
	if err != nil {
		return err
	}
	if !cfg.Tenancy.Enabled {
		return errors.New("tenant provision requires tenancy.enabled=true")
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return fmt.Errorf("schema is not ready for tenant provision: %w", err)
	}
	if err := tenant.VerifyProvisioningRLS(ctx, db, string(cfg.Database.Dialect)); err != nil {
		return fmt.Errorf("tenant RLS preflight: %w", err)
	}

	report, err := tenant.Provision(ctx, db, string(cfg.Database.Dialect), tenant.ProvisionRequest{TenantID: id, AdminEmail: email, AdminPasswordHash: hash, Locale: *locale, Timezone: *timezone})
	if err != nil {
		return fmt.Errorf("provision tenant: %w", err)
	}
	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(report)
	}
	_, err = fmt.Fprintf(stdout, "tenant-provision-complete tenant_id=%s state=%s admin_created=%t defaults_applied=%d\n", report.TenantID, report.State, report.AdminCreated, report.DefaultsApplied)
	return err
}

func readProvisionPassword(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", errors.New("initial administrator password input is required")
	}
	reader := bufio.NewReader(io.LimitReader(stdin, 4097))
	password, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read initial administrator password: %w", err)
	}
	password = strings.TrimSpace(password)
	if len(password) < 12 || len(password) > 4096 {
		return "", errors.New("initial administrator password must be 12-4096 characters")
	}
	return password, nil
}
