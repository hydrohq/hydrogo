package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/modules/commerce"
	"github.com/hydrohq/hydrogo/modules/invoicing/bizimhesap"
)

var registeredInvoiceProvider commerce.InvoiceProvider

// RegisterInvoiceProvider registers a custom invoice provider implementation (for tests or extensions).
func RegisterInvoiceProvider(p commerce.InvoiceProvider) {
	registeredInvoiceProvider = p
}

func (a *serveApp) commerceInvoiceProvider() commerce.InvoiceProvider {
	if registeredInvoiceProvider != nil {
		return registeredInvoiceProvider
	}
	cfg := a.cfg.Invoicing.BizimHesap
	if !a.modules["commerce"] || !cfg.Enabled() {
		return nil
	}
	return bizimhesap.Provider{
		Config: bizimhesap.Config{
			FirmID:      cfg.FirmID,
			EndpointURL: cfg.EndpointURL,
			APIKey:      cfg.APIKey,
			Token:       cfg.Token,
		},
	}
}

type collectionInvoiceCustomers struct {
	records func() collections.Service
}

func (c collectionInvoiceCustomers) ResolveInvoiceCustomer(ctx context.Context, orderID string) (commerce.InvoiceCustomerInfo, bool, error) {
	if c.records == nil {
		return commerce.InvoiceCustomerInfo{}, false, nil
	}
	svc := c.records()
	if svc.Repository == nil {
		return commerce.InvoiceCustomerInfo{}, false, nil
	}
	recs, err := svc.QueryRecords(ctx, "invoice_details", collections.Query{
		Limit: 1,
		Where: []collections.Condition{
			{Field: "order_id", Op: collections.Equals, Value: orderID},
		},
		Sort: &collections.Sort{Field: collections.SortCreatedAt, Desc: true},
	})
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) || errors.Is(err, collections.ErrInvalidQuery) {
			return commerce.InvoiceCustomerInfo{}, false, nil
		}
		return commerce.InvoiceCustomerInfo{}, false, err
	}
	if len(recs) == 0 {
		return commerce.InvoiceCustomerInfo{}, false, nil
	}
	data := recs[0].Data
	return commerce.InvoiceCustomerInfo{
		InvoiceType: stringField(data, "invoice_type"),
		TCKN:        stringField(data, "tckn"),
		CompanyName: stringField(data, "company_name"),
		VKN:         stringField(data, "vkn"),
		TaxOffice:   stringField(data, "tax_office"),
	}, true, nil
}

func stringField(data map[string]any, key string) string {
	raw, ok := data[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func runCommerceInvoiceWorker(ctx context.Context, a *serveApp, provider commerce.InvoiceProvider, logger *slog.Logger, stderr io.Writer) {
	if provider == nil {
		return
	}
	worker := commerce.InvoiceWorker{
		Service:  a.commerceService(),
		Provider: provider,
		Notifier: a.commerceNotifier(),
	}

	// Brief startup settle before polling due invoices.
	startupTimer := time.NewTimer(5 * time.Second)
	select {
	case <-ctx.Done():
		startupTimer.Stop()
		return
	case <-startupTimer.C:
	}

	ticker := time.NewTicker(worker.IntervalOrDefault())
	defer ticker.Stop()

	runPass := func() {
		if a.requireTenantScope {
			for _, scope := range a.tenantScopes {
				count, err := worker.IssueDue(tenant.WithScope(ctx, scope))
				logInvoicePass(ctx, logger, stderr, scope.TenantID, count, err)
			}
			return
		}
		count, err := worker.IssueDue(ctx)
		logInvoicePass(ctx, logger, stderr, "", count, err)
	}

	for {
		runPass()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func logInvoicePass(ctx context.Context, logger *slog.Logger, stderr io.Writer, tenantID string, count int, err error) {
	if err != nil && ctx.Err() == nil {
		if logger != nil {
			logger.WarnContext(ctx, "commerce invoice worker failed", "tenant", tenantID, "error", err)
		} else {
			_, _ = fmt.Fprintf(stderr, "hydro commerce invoice worker (%s): %v\n", tenantID, err)
		}
	}
	if count > 0 && logger != nil {
		logger.InfoContext(ctx, "commerce invoices auto-issued", "tenant", tenantID, "count", count)
	}
}
