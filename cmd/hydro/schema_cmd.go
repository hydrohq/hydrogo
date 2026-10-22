package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

// schemaCommand compares signed release migration ledgers. It deliberately
// reports IDs/counts only; SQL dialect and migration bodies stay internal.
func schemaCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "diff" {
		return errors.New("schema requires the diff subcommand")
	}
	flags := flag.NewFlagSet("schema diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "Baseline signed release manifest")
	current := flags.String("current", "", "Current signed release manifest")
	publicKey := flags.String("public-key", "", "Trusted Ed25519 public key (base64)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*baseline) == "" || strings.TrimSpace(*current) == "" || strings.TrimSpace(*publicKey) == "" {
		return errors.New("schema diff requires --baseline, --current and --public-key")
	}
	base, err := readVerifiedReleaseManifest(*baseline, *publicKey)
	if err != nil {
		return fmt.Errorf("verify baseline manifest: %w", err)
	}
	next, err := readVerifiedReleaseManifest(*current, *publicKey)
	if err != nil {
		return fmt.Errorf("verify current manifest: %w", err)
	}
	diff, err := release.CompareMigrationChecksums(base.MigrationChecksums, next.MigrationChecksums)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "schema-compatible added=%d removed=%d changed=%d\n", len(diff.Added), len(diff.Removed), len(diff.Changed))
	return err
}
