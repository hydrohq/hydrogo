package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/mailer"
	"github.com/hydrohq/hydrogo/internal/notify"
	outboxstore "github.com/hydrohq/hydrogo/internal/outbox/sqlstore"
	"github.com/hydrohq/hydrogo/internal/router"
	"github.com/hydrohq/hydrogo/modules/commerce"
	commercestore "github.com/hydrohq/hydrogo/modules/commerce/repository/sqlstore"
	commercetransport "github.com/hydrohq/hydrogo/modules/commerce/transport"
	"github.com/hydrohq/hydrogo/modules/members"
)

var (
	registeredShippingProvider commerce.ShippingProvider
	registeredPaymentProvider  commerce.PaymentProvider
)

// RegisterShippingProvider registers a custom shipping carrier implementation.
func RegisterShippingProvider(p commerce.ShippingProvider) {
	registeredShippingProvider = p
}

// RegisterPaymentProvider registers a custom payment gateway implementation.
func RegisterPaymentProvider(p commerce.PaymentProvider) {
	registeredPaymentProvider = p
}

// commercePaymentProviderName names the configured payment provider for
// Service.Reconcile's NotifiedButNotPaid check -- "" when none is
// configured, which correctly skips that check rather than flagging every
// unconfirmed order as an anomaly.
func (a *serveApp) commercePaymentProviderName() string {
	if registeredPaymentProvider != nil {
		return "custom"
	}
	// Every payments-module provider records the order id as its notification
	// event id, which is what reconcile joins on.
	if !paymentsEnabled(a.modules, a.cfg) {
		return ""
	}
	return strings.Join(a.paymentMethodNames(), ",")
}

// commerceCSRFExemptPaths names the routes a payment provider calls
// directly, with no browser session to carry the CSRF cookie -- see
// security.CSRFMiddleware's own doc comment for why these need full
// exemption, not just a relaxed check.
func commerceCSRFExemptPaths(modules map[string]bool, cfg config.Commerce) []string {
	return nil
}

// commercePaymentProvider builds the configured payment provider, or nil
// when none is set up -- a project with no provider configured still gets
// a fully working cart/checkout, just no hosted payment redirect.
func (a *serveApp) commercePaymentProvider() commerce.PaymentProvider {
	return registeredPaymentProvider
}

// commerceShippingProvider builds the configured carrier integration, or
// nil when none is set up -- an order can always be paid and fulfilled by
// hand (no admin shipment-creation endpoint call) without one.
func (a *serveApp) commerceShippingProvider() commerce.ShippingProvider {
	if registeredShippingProvider != nil {
		return registeredShippingProvider
	}
	return a.buildCarriers()
}

// commerceNotifier builds the configured operational chat alert sender, or
// nil when Notifications.Telegram is not set up -- a project with no
// notification channel configured just gets no alerts, cart/checkout and
// reconciliation both still work fully without it.
func (a *serveApp) commerceNotifier() notify.Sender {
	cfg := a.cfg.Notifications.Telegram
	if !cfg.Enabled() {
		return nil
	}
	telegramSender := notify.Telegram{BotToken: cfg.BotToken, ChatID: cfg.ChatID}
	outboxStore := notify.OutboxStore{
		Dialect:            a.cfg.Database.Dialect,
		RequireTenantScope: a.requireTenantScope,
	}
	return notify.OutboxSender{
		Store:    outboxStore,
		DB:       a.db,
		Channel:  "telegram",
		Fallback: telegramSender,
	}
}

// commerceService builds the commerce module's service, independent of the
// route registration below so serve_workers.go's reservation-expiry sweep
// can use the same Service without going through HTTP.
func (a *serveApp) commerceService() commerce.Service {
	store := commercestore.Store{Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	tax, shipping := a.cfg.Commerce.Tax, a.cfg.Commerce.Shipping
	return commerce.Service{
		Transactions: a.transactioner,
		Repository:   store,
		Outbox:       outboxstore.Store{Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope},
		Pricing: commerce.PricingConfig{
			TaxRate: tax.Rate, TaxInclusive: tax.Inclusive,
			ShippingFlatRateCents: shipping.FlatRateCents, ShippingFreeAboveCents: shipping.FreeAboveCents,
			MethodDiscountPercent: a.methodDiscounts(),
		},
		Prices:                 catalogPrices{records: func() collections.Service { return a.collectionService }},
		InvoiceCustomers:       collectionInvoiceCustomers{records: func() collections.Service { return a.collectionService }},
		DiscountRequiresMember: a.cfg.Commerce.DiscountRequiresMember,
	}
}

// commerceRoutes registers the public cart/checkout API and the protected
// stock endpoint. Always nil when the module is off, per AGENTS.md rule 5:
// an inactive module adds no routes.
func (a *serveApp) commerceRoutes() []router.Route {
	if !a.modules["commerce"] {
		return nil
	}
	handler := commercetransport.Handler{
		Service:               a.commerceService(),
		Secure:                a.cfg.Security.SecureCookies,
		CartLimiter:           a.newRateLimiter(60, time.Minute), // generous for normal browsing, still bounded
		CheckoutLimiter:       a.newRateLimiter(10, time.Minute),
		PendingTracker:        commercetransport.NewPendingOrderTracker(),
		MaxPendingOrdersPerIP: a.cfg.Commerce.MaxPendingOrdersPerIP,
		MaxLineQuantity:       a.cfg.Commerce.MaxLineQuantity,
		DefaultCurrency:       a.cfg.Commerce.Currencies.Default,
		DisplayCurrencies:     a.cfg.Commerce.Currencies.Display,
		CurrencyRates:         a.cfg.Commerce.Currencies.Rates,
		Notifier:              a.commerceNotifier(),
	}
	if a.smtpSender != nil {
		handler.Mailer = mailer.NotificationSender{Sender: a.smtpSender}
	}
	if a.modules["members"] {
		handler.MemberID = commerceMemberIDResolver(a.memberService)
	}
	provider := a.commercePaymentProvider()
	var paymentRoutes []commercetransport.Route
	if provider == nil {
		provider, paymentRoutes = a.buildPayments(handler.Service, handler.Notifier)
	}
	handler.PaymentProvider = provider
	if refunder, ok := provider.(commerce.PaymentRefunder); ok {
		handler.Refunder = refunder
	}
	handler.ShippingProvider = a.commerceShippingProvider()
	handler.InvoiceProvider = a.commerceInvoiceProvider()
	handler.AutoIssueInvoiceOnPaid = a.invoiceAutoIssueOnPaid()

	transportRoutes := append(handler.Routes(), paymentRoutes...)
	transportRoutes = append(transportRoutes, a.shippingRoutes(handler.Service, handler.ClientIP)...)
	routes := make([]router.Route, 0, len(transportRoutes))
	for _, route := range transportRoutes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		routes = append(routes, router.Route{Pattern: route.Pattern, Access: access, Permission: route.Permission, Handler: route.Handler})
	}
	return routes
}

// commerceMemberIDResolver adapts modules/members' session cookie to the
// transport.Handler.MemberID hook, so a logged-in visitor's cart is tied to
// their account instead of an anonymous cookie. commerce itself never
// imports modules/members -- this glue lives in cmd/hydro, the composition
// root, same as MemberOrdersReader's wiring in serve_runtime.go.
func commerceMemberIDResolver(service members.Service) func(*http.Request) (string, bool) {
	return func(r *http.Request) (string, bool) {
		cookie, err := r.Cookie(members.SessionCookieName)
		if err != nil || cookie.Value == "" {
			return "", false
		}
		_, member, err := service.ValidateSession(r.Context(), cookie.Value)
		if err != nil || member == nil || !member.Active {
			return "", false
		}
		return member.ID, true
	}
}

// runCommerceReservationSweeper is the periodic check behind Worker.ExpireDue:
// an order whose payment never completed releases its held stock back to
// available. A plain ticker, not internal/jobs's lease queue -- there is one
// recurring thing to do, not a stream of discrete one-shot tasks; see
// modules/commerce/worker.go's own doc comment.
func runCommerceReservationSweeper(ctx context.Context, worker commerce.Worker, db database.Executor, stderr io.Writer) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if _, err := worker.ExpireDue(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "hydro commerce reservation sweep: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runCommerceReconciliationAlert is Service.Reconcile run on a schedule, so
// drift gets noticed without an operator remembering to run
// `hydro commerce reconcile` by hand. A plain ticker, like the sweeper
// above: this is read-only and idempotent to call repeatedly, so a crash
// between ticks just means the next tick reports the same thing again,
// never silently misses it. Every non-clean report is logged at Warn (not
// Error -- Reconcile finding something is the alert itself, not this
// job malfunctioning) with enough structure for an operator's existing log
// pipeline to turn into a real alert; see
// docs/decisions/0018-commerce-reconciliation.md. notifier is optional
// (nil when Notifications.Telegram is not configured): a non-clean report
// always logs at Warn regardless, notifier is an additional, best-effort
// push so an operator does not have to be watching the logs.
func runCommerceReconciliationAlert(ctx context.Context, service commerce.Service, db database.Executor, provider string, notifier notify.Sender, logger *slog.Logger, stderr io.Writer) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		report, err := service.Reconcile(ctx, db, time.Now().UTC(), commerce.ReconcileConfig{}, provider, 100)
		if err != nil {
			if ctx.Err() == nil {
				_, _ = fmt.Fprintf(stderr, "hydro commerce reconcile: %v\n", err)
			}
		} else if !report.Clean() {
			if logger != nil {
				logger.WarnContext(ctx, "commerce reconciliation found drift",
					"stale_pending_orders", len(report.StalePendingOrders),
					"notified_but_not_paid", len(report.NotifiedButNotPaid),
					"reservation_drift", len(report.ReservationDrift),
				)
			}
			if notifier != nil {
				_ = notifier.Send(ctx, notify.Message{
					Title: "Reconciliation alert",
					Body: fmt.Sprintf("Stale pending orders: %d, notified but unpaid: %d, stock mismatches: %d",
						len(report.StalePendingOrders), len(report.NotifiedButNotPaid), len(report.ReservationDrift)),
				})
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runCommerceRefundReminder periodically checks for unresolved refund_required
// records and alerts the operator through notifier (which routes via the outbox).
func runCommerceRefundReminder(ctx context.Context, service commerce.Service, notifier notify.Sender, logger *slog.Logger, stderr io.Writer) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	worker := commerce.RefundReminderWorker{
		Service:  service,
		Notifier: notifier,
		Interval: 6 * time.Hour,
	}
	for {
		count, err := worker.RemindDue(ctx)
		if err != nil && ctx.Err() == nil {
			if logger != nil {
				logger.WarnContext(ctx, "commerce refund reminder failed", "error", err)
			} else {
				_, _ = fmt.Fprintf(stderr, "hydro commerce refund reminder: %v\n", err)
			}
		} else if count > 0 && logger != nil {
			logger.InfoContext(ctx, "commerce refund reminder sent", "unresolved_count", count)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// pendingCheckRound orders the pending orders for one round of the pending-payment check: the ones checked
// least recently (or never) first, so a pile of unpaid orders cannot keep a paid one waiting behind it.
func pendingCheckRound(orders []commerce.Order, lastChecked map[string]time.Time) []commerce.Order {
	round := append([]commerce.Order(nil), orders...)
	sort.SliceStable(round, func(i, j int) bool {
		li, lj := lastChecked[round[i].ID], lastChecked[round[j].ID]
		if !li.Equal(lj) {
			return li.Before(lj)
		}
		return round[i].CreatedAt.Before(round[j].CreatedAt)
	})
	return round
}

// runCommercePendingPaymentCheck is the safety net for a shopper who paid and closed the browser: neither
// the return redirect nor a notification arrived, so the order would sit pending until its stock hold
// expires. Every couple of minutes it asks the payment provider about orders that have been pending
// for a little while and settles the ones that were paid. Orders are taken least-recently-checked first
// and each round has a time budget, so a flood of unpaid orders delays but never starves a paid one.
func runCommercePendingPaymentCheck(ctx context.Context, a *serveApp, service commerce.Service, db database.Executor, logger *slog.Logger) {
	// ponytail: the list is capped (fetchLimit) before it is rotated. Pending orders live at most one stock
	// hold (30 min) and checkout is rate-limited per IP, so this is far above real volume; a flood beyond it
	// delays the newest orders instead of starving anything for good. Keyset paging in the store would lift it.
	const (
		interval   = 2 * time.Minute
		budget     = 100 * time.Second
		chunk      = 10
		workers    = 4
		fetchLimit = 2000
	)
	lastChecked := map[string]time.Time{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		a.paymentCheckerMu.RLock()
		checker := a.paymentChecker
		a.paymentCheckerMu.RUnlock()
		if checker == nil {
			continue
		}
		orders, err := service.Repository.ListPendingOrdersOlderThan(ctx, db, time.Now().UTC().Add(-90*time.Second), fetchLimit)
		if err != nil {
			if ctx.Err() == nil && logger != nil {
				logger.WarnContext(ctx, "commerce pending-payment check could not list orders", "error", err)
			}
			continue
		}
		live := make(map[string]bool, len(orders))
		for _, o := range orders {
			live[o.ID] = true
		}
		for id := range lastChecked {
			if !live[id] {
				delete(lastChecked, id)
			}
		}

		roundCtx, cancel := context.WithTimeout(ctx, budget)
		round := pendingCheckRound(orders, lastChecked)
		jobs := make(chan []commerce.Order)
		var mu sync.Mutex
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for group := range jobs {
					checker.CheckPending(roundCtx, group)
					if roundCtx.Err() != nil {
						continue // the group may have been cut short: it was not fully checked
					}
					now := time.Now()
					mu.Lock()
					for _, o := range group {
						lastChecked[o.ID] = now
					}
					mu.Unlock()
				}
			}()
		}
	feed:
		for start := 0; start < len(round); start += chunk {
			select {
			case jobs <- round[start:min(start+chunk, len(round))]:
			case <-roundCtx.Done():
				break feed
			}
		}
		close(jobs)
		wg.Wait()
		cancel()
	}
}
