//go:build !hydromodules

package main

import (
	"net/http"

	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/notify"
	"github.com/hydrohq/hydrogo/modules/commerce"
	commercetransport "github.com/hydrohq/hydrogo/modules/commerce/transport"
)

// Payment gateways, shipping carriers and e-invoicing providers are optional add-on
// modules distributed separately. Without them (no `hydromodules` build tag) these
// stand-ins keep the commerce core working: no gateway, carrier or invoicing
// provider is registered, and no extra routes are added.
// Custom providers can still be plugged in through RegisterShippingProvider and
// RegisterInvoiceProvider.

func paymentsEnabled(map[string]bool, config.Config) bool             { return false }
func paymentsCSRFExemptPaths(map[string]bool, config.Config) []string { return nil }

func (a *serveApp) buildPayments(commerce.Service, notify.Sender) (commerce.PaymentProvider, []commercetransport.Route) {
	return nil, nil
}

func (a *serveApp) methodDiscounts() map[string]float64 { return nil }

func shippingCSRFExemptPaths(map[string]bool, config.Config) []string { return nil }

func (a *serveApp) shippingRoutes(commerce.Service, func(*http.Request) string) []commercetransport.Route {
	return nil
}

func (a *serveApp) buildCarriers() commerce.ShippingProvider { return nil }

func (a *serveApp) configuredInvoiceProvider() commerce.InvoiceProvider { return nil }

func (a *serveApp) paymentMethodNames() []string { return nil }

func (a *serveApp) invoiceAutoIssueOnPaid() bool { return false }
