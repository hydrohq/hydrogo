package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
	commercetransport "github.com/ardaozyildirim/hydrogo/modules/commerce/transport"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// commerceService builds the commerce module's service, independent of the
// route registration below so serve_workers.go's reservation-expiry sweep
// can use the same Service without going through HTTP.
func (a *serveApp) commerceService() commerce.Service {
	store := commercestore.Store{Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	return commerce.Service{
		Transactions: a.transactioner,
		Repository:   store,
		Outbox:       outboxstore.Store{Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope},
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
		Service:            a.commerceService(),
		DB:                 a.db,
		Secure:             a.cfg.Security.SecureCookies,
		RequireTenantScope: a.requireTenantScope,
		CartLimiter:        a.newRateLimiter(60, time.Minute), // generous for normal browsing, still bounded
		CheckoutLimiter:    a.newRateLimiter(10, time.Minute),
	}
	if a.modules["members"] {
		handler.MemberID = commerceMemberIDResolver(a.memberService)
	}
	routes := make([]router.Route, 0, 5)
	for _, route := range handler.Routes() {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		routes = append(routes, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
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
		if _, err := worker.ExpireDue(ctx, db, time.Now().UTC()); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "hydro commerce reservation sweep: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
