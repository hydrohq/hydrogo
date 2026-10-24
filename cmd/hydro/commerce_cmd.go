package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

// commerceCommand is the `hydro commerce` namespace. reconcile is its only
// subcommand today: an on-demand run of the same read-only consistency
// sweep routes_commerce.go's runCommerceReconciliationAlert runs on a
// schedule, for an operator who wants a report right now (before a launch,
// after an incident, in a cron job feeding their own alerting) rather than
// waiting for the next five-minute tick. It never writes -- running it
// cannot make an inconsistency worse -- and it uses the same unscoped,
// cross-tenant database access `hydro jobs dead-letter` already does
// (openJobsDatabase), since this is operator tooling, not a
// request-scoped API call.
func commerceCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "reconcile" {
		return errors.New("commerce requires the reconcile subcommand")
	}
	flags := flag.NewFlagSet("commerce reconcile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	provider := flags.String("provider", "", "Payment provider to check for notified-but-unpaid orders, e.g. paytr; empty skips this check")
	hold := flags.Duration("hold", 30*time.Minute, "Payment hold window a pending order may sit in before it is stale (mirror the running server's commerce worker hold)")
	staleAfter := flags.Duration("stale-after", 15*time.Minute, "Grace period past --hold before a still-pending order is flagged")
	limit := flags.Int("limit", 100, "Maximum rows returned per check")
	jsonOutput := flags.Bool("json", false, "Print a machine-readable report")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *hold <= 0 || *staleAfter <= 0 || *limit < 1 || *limit > 10000 {
		return errors.New("reconcile requires positive --hold and --stale-after, and --limit between 1 and 10000")
	}

	cfg, _, db, err := openJobsDatabase(flags, values)
	if err != nil {
		return err
	}
	defer db.Close()

	service := commerce.Service{Repository: commercestore.Store{Dialect: cfg.Database.Dialect}}
	report, err := service.Reconcile(context.Background(), db, time.Now().UTC(), commerce.ReconcileConfig{Hold: *hold, StaleAfter: *staleAfter}, *provider, *limit)
	if err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}

	if *jsonOutput {
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}
	if _, err := fmt.Fprintf(stdout, "reconcile generated_at=%s stale_pending_orders=%d notified_but_not_paid=%d reservation_drift=%d clean=%t\n",
		report.GeneratedAt.Format(time.RFC3339), len(report.StalePendingOrders), len(report.NotifiedButNotPaid), len(report.ReservationDrift), report.Clean()); err != nil {
		return err
	}
	for _, order := range report.StalePendingOrders {
		if _, err := fmt.Fprintf(stdout, "  stale-pending order_id=%s created_at=%s total_cents=%d\n", order.ID, order.CreatedAt.Format(time.RFC3339), order.TotalCents); err != nil {
			return err
		}
	}
	for _, order := range report.NotifiedButNotPaid {
		if _, err := fmt.Fprintf(stdout, "  notified-but-not-paid order_id=%s payment_status=%s total_cents=%d\n", order.ID, order.PaymentStatus, order.TotalCents); err != nil {
			return err
		}
	}
	for _, drift := range report.ReservationDrift {
		if _, err := fmt.Fprintf(stdout, "  reservation-drift variant_id=%s reserved=%d expected=%d\n", drift.VariantID, drift.Reserved, drift.Expected); err != nil {
			return err
		}
	}
	return nil
}
