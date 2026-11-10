package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

const (
	pendingTrackerWindow        = 30 * time.Minute
	pendingTrackerSweepInterval = time.Minute
)

// PendingOrderTracker limits open pending orders per client IP to prevent
// checkout abandonment attacks. The mutex only guards the in-memory map;
// it is never held across database reads.
type PendingOrderTracker struct {
	mu        sync.Mutex
	orders    map[string][]pendingOrderEntry
	lastSweep time.Time
}

type pendingOrderEntry struct {
	reservation string
	orderID     string
	createdAt   time.Time
}

func NewPendingOrderTracker() *PendingOrderTracker {
	return &PendingOrderTracker{orders: make(map[string][]pendingOrderEntry)}
}

// sweepLocked drops expired entries and empty IP keys. It scans every key at
// most once per pendingTrackerSweepInterval so request cost stays bounded;
// the key being served is always pruned. Callers must hold t.mu.
func (t *PendingOrderTracker) sweepLocked(ip string, now time.Time, window time.Duration) {
	prune := func(key string) {
		list := t.orders[key]
		kept := list[:0]
		for _, e := range list {
			if now.Sub(e.createdAt) <= window {
				kept = append(kept, e)
			}
		}
		if len(kept) == 0 {
			delete(t.orders, key)
		} else {
			t.orders[key] = kept
		}
	}
	if now.Sub(t.lastSweep) >= pendingTrackerSweepInterval {
		t.lastSweep = now
		for key := range t.orders {
			prune(key)
		}
		return
	}
	prune(ip)
}

// CountPending returns how many open pending orders ip holds. Order status is
// read from the database without holding the lock; the result is merged back
// under the lock, keeping reservations made while the reads were in flight.
func (t *PendingOrderTracker) CountPending(ctx context.Context, ip string, svc commerce.Service, now time.Time, window time.Duration) int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	t.sweepLocked(ip, now, window)
	entries := append([]pendingOrderEntry(nil), t.orders[ip]...)
	t.mu.Unlock()
	if len(entries) == 0 {
		return 0
	}

	activeOrders := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.orderID == "" {
			continue
		}
		order, err := svc.Order(ctx, e.orderID)
		if err == nil && order.PaymentStatus == commerce.PaymentPending {
			activeOrders[e.orderID] = true
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	current := t.orders[ip]
	merged := make([]pendingOrderEntry, 0, len(current))
	for _, e := range current {
		if e.orderID == "" || activeOrders[e.orderID] {
			merged = append(merged, e)
		}
	}
	if len(merged) == 0 {
		delete(t.orders, ip)
	} else {
		t.orders[ip] = merged
	}
	return len(merged)
}

// TryReserve atomically counts ip's open slots and, if fewer than max are
// taken, adds a placeholder and returns its token. Follow with Confirm once
// the order exists, or Release when no pending order resulted.
func (t *PendingOrderTracker) TryReserve(ip string, max int) (string, bool) {
	if t == nil {
		return "", true
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", false
	}
	id := hex.EncodeToString(token[:])
	now := time.Now().UTC()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sweepLocked(ip, now, pendingTrackerWindow)
	if len(t.orders[ip]) >= max {
		return "", false
	}
	t.orders[ip] = append(t.orders[ip], pendingOrderEntry{reservation: id, createdAt: now})
	return id, true
}

// Confirm binds a reservation to the order that was created for it.
func (t *PendingOrderTracker) Confirm(ip, reservation, orderID string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.orders[ip] {
		if t.orders[ip][i].reservation == reservation {
			t.orders[ip][i].orderID = orderID
			return
		}
	}
}

// Release frees a reservation that did not become a pending order.
func (t *PendingOrderTracker) Release(ip, reservation string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entries := t.orders[ip]
	for i := range entries {
		if entries[i].reservation == reservation {
			entries = append(entries[:i], entries[i+1:]...)
			break
		}
	}
	if len(entries) == 0 {
		delete(t.orders, ip)
	} else {
		t.orders[ip] = entries
	}
}

// Record adds an already-created pending order without a prior reservation.
func (t *PendingOrderTracker) Record(ip, orderID string, createdAt time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.orders[ip] = append(t.orders[ip], pendingOrderEntry{orderID: orderID, createdAt: createdAt})
}
