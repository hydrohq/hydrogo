package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func tenantCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("tenant command requires a subcommand (validate, backfill, list)")
	}

	switch args[0] {
	case "validate":
		return tenantValidateCommand(args[1:], stdout, stderr)
	case "backfill":
		return tenantBackfillCommand(args[1:], stdout, stderr)
	case "list":
		return tenantListCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown tenant subcommand %q (supported: validate, backfill, list)", args[0])
	}
}

func tenantValidateCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	jsonOutput := flags.Bool("json", false, "Print machine-readable JSON output")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		if strings.Contains(err.Error(), "database DSN is required") && (values.configPath == nil || *values.configPath == "") {
			if *jsonOutput {
				_ = json.NewEncoder(stdout).Encode(map[string]any{
					"enabled": false,
					"message": "tenancy is disabled (no configuration file or database provided)",
				})
				return nil
			}
			_, err = fmt.Fprintln(stdout, "tenancy-disabled enabled=false")
			return err
		}
		return err
	}

	if !cfg.Tenancy.Enabled {
		if *jsonOutput {
			_ = json.NewEncoder(stdout).Encode(map[string]any{
				"enabled": false,
				"message": "tenancy is disabled in current configuration",
			})
			return nil
		}
		_, err = fmt.Fprintln(stdout, "tenancy-disabled enabled=false")
		return err
	}

	hostErrors := make(map[string]string)
	validHosts := make(map[string]string)
	for host, tenantID := range cfg.Tenancy.Hosts {
		if _, err := tenant.NewScope(tenantID, tenantID); err != nil {
			hostErrors[host] = err.Error()
		} else {
			validHosts[host] = tenantID
		}
	}

	if len(hostErrors) > 0 {
		if *jsonOutput {
			_ = json.NewEncoder(stdout).Encode(map[string]any{
				"valid":        false,
				"errors":       hostErrors,
				"valid_hosts":  validHosts,
				"require_host": cfg.Tenancy.RequireHost,
			})
			return errors.New("invalid tenant host configuration")
		}
		for host, errStr := range hostErrors {
			_, _ = fmt.Fprintf(stderr, "invalid tenant host %s: %s\n", host, errStr)
		}
		return errors.New("invalid tenant host configuration")
	}

	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"valid":        true,
			"enabled":      true,
			"hosts":        validHosts,
			"host_count":   len(validHosts),
			"require_host": cfg.Tenancy.RequireHost,
		})
	}

	_, err = fmt.Fprintf(stdout, "tenant-config-valid enabled=true hosts=%d require_host=%t\n", len(validHosts), cfg.Tenancy.RequireHost)
	return err
}

func tenantBackfillCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant backfill", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	targetTenantID := flags.String("tenant-id", "", "Target tenant ID to assign unassigned rows to (required)")
	confirm := flags.Bool("confirm", false, "Confirm execution of the backfill migration")
	dryRun := flags.Bool("dry-run", false, "Simulate backfill and report unassigned row counts without modifying data")
	jsonOutput := flags.Bool("json", false, "Print output in JSON format")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	trimmedTenantID := strings.TrimSpace(*targetTenantID)
	if trimmedTenantID == "" {
		return errors.New("--tenant-id is required")
	}
	if _, err := tenant.NewScope(trimmedTenantID, trimmedTenantID); err != nil {
		return fmt.Errorf("invalid --tenant-id: %w", err)
	}

	if !*confirm && !*dryRun {
		return errors.New("tenant backfill requires either --dry-run or --confirm")
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
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

	if *dryRun {
		report, err := tenant.PlanBackfill(ctx, db, string(cfg.Database.Dialect), trimmedTenantID)
		if err != nil {
			return fmt.Errorf("plan backfill: %w", err)
		}
		if *jsonOutput {
			return json.NewEncoder(stdout).Encode(report)
		}
		_, err = fmt.Fprintf(stdout, "tenant-backfill-plan target_tenant_id=%s total_unassigned_rows=%d tables_affected=%d\n", report.TargetTenantID, report.TotalRows, len(report.RowsAffected))
		return err
	}

	report, err := tenant.ExecuteBackfill(ctx, db, string(cfg.Database.Dialect), trimmedTenantID)
	if err != nil {
		return fmt.Errorf("execute backfill: %w", err)
	}
	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(report)
	}
	_, err = fmt.Fprintf(stdout, "tenant-backfill-complete target_tenant_id=%s total_rows_migrated=%d tables_affected=%d\n", report.TargetTenantID, report.TotalRows, len(report.RowsAffected))
	return err
}

func tenantListCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	jsonOutput := flags.Bool("json", false, "Print output in JSON format")

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

	tenantsFound := make(map[string]bool)

	// Collect configured hosts
	for _, tenantID := range cfg.Tenancy.Hosts {
		if strings.TrimSpace(tenantID) != "" {
			tenantsFound[tenantID] = true
		}
	}

	// Query distinct tenants in pages, users, members
	for _, table := range []string{"pages", "users", "members"} {
		rows, err := db.QueryContext(ctx, "SELECT DISTINCT tenant_id FROM "+table+" WHERE tenant_id <> ''")
		if err != nil {
			continue
		}
		for rows.Next() {
			var tID string
			if scanErr := rows.Scan(&tID); scanErr == nil && strings.TrimSpace(tID) != "" {
				tenantsFound[tID] = true
			}
		}
		rows.Close()
	}

	sortedTenants := make([]string, 0, len(tenantsFound))
	for tID := range tenantsFound {
		sortedTenants = append(sortedTenants, tID)
	}
	sort.Strings(sortedTenants)

	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"tenants": sortedTenants,
			"count":   len(sortedTenants),
		})
	}

	for _, tID := range sortedTenants {
		_, _ = fmt.Fprintf(stdout, "tenant id=%s\n", tID)
	}
	_, err = fmt.Fprintf(stdout, "total_tenants=%d\n", len(sortedTenants))
	return err
}
