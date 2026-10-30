package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func tenantExportCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant export", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	output := flags.String("output", "", "Output path for the snapshot (.tar.gz)")
	jsonOutput := flags.Bool("json", false, "Print machine-readable JSON output")

	posArgs, err := parseCommandArgs(flags, args)
	if err != nil {
		return err
	}
	if len(posArgs) < 1 {
		return errors.New("tenant export requires a tenant ID")
	}

	tenantID := strings.TrimSpace(posArgs[0])
	if tenantID == "" {
		return errors.New("tenant ID cannot be empty")
	}

	cfg, err := loadMaintenanceConfig(flags, values)
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

	outPath := *output
	if strings.TrimSpace(outPath) == "" {
		outPath = fmt.Sprintf("%s-snapshot.tar.gz", tenantID)
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	manifest, err := tenant.ExportSnapshot(ctx, db, string(cfg.Database.Dialect), tenantID, file)
	if err != nil {
		_ = os.Remove(outPath)
		return fmt.Errorf("export snapshot: %w", err)
	}

	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"ok":       true,
			"path":     outPath,
			"manifest": manifest,
		})
	}

	_, _ = fmt.Fprintf(stdout, "Exported tenant snapshot: %s\n", outPath)
	_, _ = fmt.Fprintf(stdout, "Tenant ID: %s\n", manifest.TenantID)
	_, _ = fmt.Fprintf(stdout, "Total records: %d across %d tables\n", manifest.TotalRows, len(manifest.TableCounts))
	_, _ = fmt.Fprintf(stdout, "Checksum: %s\n", manifest.Checksum)
	return nil
}

func tenantImportCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("tenant import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	newTenantID := flags.String("new-tenant-id", "", "Target tenant ID (overrides exported ID)")
	jsonOutput := flags.Bool("json", false, "Print machine-readable JSON output")

	posArgs, err := parseCommandArgs(flags, args)
	if err != nil {
		return err
	}
	if len(posArgs) < 1 {
		return errors.New("tenant import requires an archive path")
	}

	archivePath := strings.TrimSpace(posArgs[0])
	if archivePath == "" {
		return errors.New("archive path cannot be empty")
	}

	var reader io.Reader
	if archivePath == "-" {
		reader = stdin
	} else {
		file, err := os.Open(archivePath)
		if err != nil {
			return fmt.Errorf("open archive: %w", err)
		}
		defer file.Close()
		reader = file
	}

	cfg, err := loadMaintenanceConfig(flags, values)
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

	manifest, err := tenant.ImportSnapshot(ctx, db, string(cfg.Database.Dialect), reader, *newTenantID)
	if err != nil {
		return fmt.Errorf("import snapshot: %w", err)
	}

	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"ok":       true,
			"manifest": manifest,
		})
	}

	_, _ = fmt.Fprintf(stdout, "Imported tenant snapshot successfully.\n")
	_, _ = fmt.Fprintf(stdout, "Target Tenant ID: %s\n", manifest.TenantID)
	_, _ = fmt.Fprintf(stdout, "Total records: %d across %d tables\n", manifest.TotalRows, len(manifest.TableCounts))
	return nil
}

func parseCommandArgs(flags *flag.FlagSet, args []string) ([]string, error) {
	var reordered []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			reordered = append(reordered, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagName := strings.TrimLeft(arg, "-")
				f := flags.Lookup(flagName)
				if f != nil {
					// Check if boolean
					type boolFlag interface {
						IsBoolFlag() bool
					}
					if bf, ok := f.Value.(boolFlag); !ok || !bf.IsBoolFlag() {
						i++
						reordered = append(reordered, args[i])
					}
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}
	reordered = append(reordered, positional...)
	if err := flags.Parse(reordered); err != nil {
		return nil, err
	}
	return flags.Args(), nil
}
