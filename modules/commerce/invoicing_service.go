package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/notify"
)

const (
	defaultInvoiceClaimStaleWindow = 5 * time.Minute
	defaultInvoiceSettleDelay      = 15 * time.Second
	defaultInvoiceWorkerLimit      = 25
	defaultInvoiceWorkerInterval   = 30 * time.Second
)

var (
	// ErrNoInvoiceProvider is returned when an invoicing endpoint is invoked without a configured provider.
	ErrNoInvoiceProvider = errors.New("invoice provider is not configured")
	// ErrInvoiceProviderFailed wraps transport or provider errors returned while issuing an external invoice.
	ErrInvoiceProviderFailed = errors.New("invoice provider request failed")
)

// IssueOrderInvoice claims a paid, uninvoiced order, builds a kuruş-balanced
// invoice document (including optional checkout tax registration info), submits
// it to provider outside any database transaction, and persists the resulting
// invoice metadata and `order.invoiced` outbox event.
func (s Service) IssueOrderInvoice(ctx context.Context, provider InvoiceProvider, orderID string) (Order, error) {
	if provider == nil {
		return Order{}, ErrNoInvoiceProvider
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return Order{}, ErrOrderNotFound
	}

	now := time.Now().UTC()
	var order Order
	var items []OrderItem
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		claimed, err := s.Repository.ClaimInvoice(ctx, tx, orderID, provider.Name(), now, now.Add(-defaultInvoiceClaimStaleWindow))
		if err != nil {
			return err
		}
		if !claimed {
			if _, err := s.Repository.GetOrder(ctx, tx, orderID); err != nil {
				return err
			}
			return ErrOrderNotInvoiceable
		}
		order, err = s.Repository.GetOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		items, err = s.Repository.ListOrderItems(ctx, tx, orderID)
		return err
	})
	if err != nil {
		return Order{}, err
	}

	var taxInfo InvoiceCustomerInfo
	var hasTaxInfo bool
	if s.InvoiceCustomers != nil {
		taxInfo, hasTaxInfo, err = s.InvoiceCustomers.ResolveInvoiceCustomer(ctx, order.ID)
		if err != nil {
			s.failInvoice(ctx, order.ID, err.Error())
			return Order{}, err
		}
	}

	doc, err := BuildInvoiceDocument(order, items, s.Pricing, taxInfo, hasTaxInfo, now)
	if err != nil {
		s.failInvoice(ctx, order.ID, err.Error())
		return Order{}, err
	}

	result, err := provider.IssueInvoice(ctx, doc)
	if err != nil {
		s.failInvoice(ctx, order.ID, err.Error())
		return Order{}, fmt.Errorf("%w: %v", ErrInvoiceProviderFailed, err)
	}
	if strings.TrimSpace(result.Provider) == "" {
		result.Provider = provider.Name()
	}
	if strings.TrimSpace(result.InvoiceNo) == "" {
		result.InvoiceNo = doc.InvoiceNo
	}

	var issued Order
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		updated, err := s.Repository.RecordInvoiceIssued(ctx, tx, order.ID, result, time.Now().UTC())
		if err != nil {
			return err
		}
		if updated {
			if err := s.emit(ctx, tx, "order.invoiced", order.ID); err != nil {
				return err
			}
		}
		issued, err = s.Repository.GetOrder(ctx, tx, order.ID)
		return err
	})
	if err != nil {
		return Order{}, err
	}
	return issued, nil
}

func (s Service) failInvoice(ctx context.Context, orderID, reason string) {
	_ = s.Transactions.Within(ctx, func(tx database.Executor) error {
		_, err := s.Repository.RecordInvoiceFailed(ctx, tx, orderID, reason, time.Now().UTC())
		return err
	})
}

// InvoiceWorker periodically issues invoices for paid orders once the settle
// delay has elapsed so companion post-checkout records can finish persisting.
type InvoiceWorker struct {
	Service     Service
	Provider    InvoiceProvider
	Notifier    notify.Sender
	SettleDelay time.Duration
	Limit       int
	Interval    time.Duration
}

// IntervalOrDefault returns the polling cadence used when scheduling automatic invoicing.
func (w InvoiceWorker) IntervalOrDefault() time.Duration {
	if w.Interval <= 0 {
		return defaultInvoiceWorkerInterval
	}
	return w.Interval
}

func (w InvoiceWorker) settleDelay() time.Duration {
	if w.SettleDelay <= 0 {
		return defaultInvoiceSettleDelay
	}
	return w.SettleDelay
}

func (w InvoiceWorker) batchLimit() int {
	if w.Limit <= 0 {
		return defaultInvoiceWorkerLimit
	}
	return w.Limit
}

// IssueDue processes one batch of paid uninvoiced orders ready for automatic invoicing.
func (w InvoiceWorker) IssueDue(ctx context.Context) (int, error) {
	if w.Provider == nil {
		return 0, nil
	}
	cutoff := time.Now().UTC().Add(-w.settleDelay())
	var orders []Order
	err := w.Service.Transactions.Within(ctx, func(tx database.Executor) error {
		var queryErr error
		orders, queryErr = w.Service.Repository.ListOrdersDueForAutoInvoice(ctx, tx, cutoff, w.batchLimit())
		return queryErr
	})
	if err != nil {
		return 0, err
	}

	issued := 0
	var errs []error
	for _, order := range orders {
		if _, err := w.Service.IssueOrderInvoice(ctx, w.Provider, order.ID); err != nil {
			if errors.Is(err, ErrOrderNotInvoiceable) {
				continue
			}
			errs = append(errs, fmt.Errorf("order %s: %w", order.ID, err))
			if w.Notifier != nil {
				_ = w.Notifier.Send(ctx, notify.Message{
					Title: "e-Fatura kesilemedi",
					Body:  fmt.Sprintf("Sipariş: %s\nSağlayıcı: %s\nHata: %s", order.ID, w.Provider.Name(), err.Error()),
				})
			}
			continue
		}
		issued++
	}
	return issued, errors.Join(errs...)
}
