package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// auditCommand inspects the audit chain (config audit.chain).
//
//	hydro audit head                         print the head of every chain (JSON) to keep outside the database
//	hydro audit verify [--anchor T:SEQ:HASH]  recompute every chain; exits non-zero when it does not add up
//
// An anchor is a head printed earlier: verify then also proves the chain still
// contains that exact link. T is the tenant id (empty for a single-tenant
// project, written as ":SEQ:HASH").
func auditCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || (args[0] != "head" && args[0] != "verify") {
		return errors.New("usage: hydro audit head|verify [--anchor TENANT:SEQ:HASH]")
	}
	action := args[0]
	flags := flag.NewFlagSet("audit "+action, flag.ContinueOnError)
	flags.SetOutput(stderr)
	anchorText := flags.String("anchor", "", "A head from hydro audit head, as TENANT:SEQ:HASH (verify)")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	var anchor *auditstore.ChainHead
	if *anchorText != "" {
		parts := strings.Split(*anchorText, ":")
		if len(parts) != 3 {
			return errors.New("--anchor must look like TENANT:SEQ:HASH")
		}
		seq, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || seq < 1 {
			return errors.New("--anchor has an invalid position")
		}
		anchor = &auditstore.ChainHead{TenantID: parts[0], Seq: seq, Hash: parts[2]}
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
	store := auditstore.Store{Dialect: cfg.Database.Dialect}
	heads, err := store.ChainHeads(ctx, db)
	if err != nil {
		return fmt.Errorf("the audit chain is not available (is audit.chain on and the database migrated?): %w", err)
	}
	if action == "head" {
		encoded, _ := json.MarshalIndent(heads, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return nil
	}
	if len(heads) == 0 {
		fmt.Fprintln(stdout, "No audit chain yet: nothing has been chained (turn on audit.chain).")
		return nil
	}
	broken := 0
	for _, head := range heads {
		var tenantAnchor *auditstore.ChainHead
		if anchor != nil && anchor.TenantID == head.TenantID {
			tenantAnchor = anchor
		}
		report, err := store.VerifyChain(ctx, db, head.TenantID, tenantAnchor)
		if err != nil {
			return err
		}
		name := head.TenantID
		if name == "" {
			name = "(default)"
		}
		if len(report.Problems) == 0 {
			fmt.Fprintf(stdout, "tenant %s: OK, %d links, head %d:%s (%d earlier events are not chained)\n", name, report.Links, report.Head.Seq, report.Head.Hash, report.Legacy)
			continue
		}
		broken++
		fmt.Fprintf(stdout, "tenant %s: BROKEN\n", name)
		for _, problem := range report.Problems {
			fmt.Fprintf(stdout, "  - %s\n", problem)
		}
	}
	if broken > 0 {
		return fmt.Errorf("audit chain verification failed for %d tenant(s)", broken)
	}
	return nil
}
