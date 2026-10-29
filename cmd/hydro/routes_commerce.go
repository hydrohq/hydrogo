package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/providers/yurticikargo"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
	commercetransport "github.com/ardaozyildirim/hydrogo/modules/commerce/transport"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// commercePaymentProviderName names the configured payment provider for
// Service.Reconcile's NotifiedButNotPaid check -- "" when none is
// configured, which correctly skips that check rather than flagging every
// unconfirmed order as an anomaly.
func (a *serveApp) commercePaymentProviderName() string {
	return ""
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
	return nil
}

// commerceShippingProvider builds the configured carrier integration, or
// nil when none is set up -- an order can always be paid and fulfilled by
// hand (no admin shipment-creation endpoint call) without one. Yurtiçi
// Kargo is the only carrier wired in today; ShippingProvider's job is to
// let a second one (Aras, MNG, ...) be added here later as its own
// provider package, without touching Service or transport.
func (a *serveApp) commerceShippingProvider() commerce.ShippingProvider {
	cfg := a.cfg.Commerce.YurticiKargo
	if !cfg.Enabled() {
		return nil
	}
	return yurticikargo.Provider{Config: yurticikargo.Config{WSUserName: cfg.WSUserName, WSPassword: cfg.WSPassword, BaseURL: cfg.ResolvedEndpoint()}}
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
	return notify.Telegram{BotToken: cfg.BotToken, ChatID: cfg.ChatID}
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
		},
		Prices: catalogPrices{records: func() collections.Service { return a.collectionService }},
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
		Service:           a.commerceService(),
		Secure:            a.cfg.Security.SecureCookies,
		CartLimiter:       a.newRateLimiter(60, time.Minute), // generous for normal browsing, still bounded
		CheckoutLimiter:   a.newRateLimiter(10, time.Minute),
		DefaultCurrency:   a.cfg.Commerce.Currencies.Default,
		DisplayCurrencies: a.cfg.Commerce.Currencies.Display,
		CurrencyRates:     a.cfg.Commerce.Currencies.Rates,
		Notifier:          a.commerceNotifier(),
	}
	if a.smtpSender != nil {
		handler.Mailer = mailer.NotificationSender{Sender: a.smtpSender}
	}
	if a.modules["members"] {
		handler.MemberID = commerceMemberIDResolver(a.memberService)
	}
	provider := a.commercePaymentProvider()
	handler.PaymentProvider = provider
	handler.ShippingProvider = a.commerceShippingProvider()

	transportRoutes := handler.Routes()
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
					Title: "Mutabakat uyarısı",
					Body: fmt.Sprintf("Bekleyen eski sipariş: %d, bildirimi gelip ödenmemiş: %d, stok tutarsızlığı: %d",
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
