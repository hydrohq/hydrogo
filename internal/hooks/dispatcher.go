// Package hooks defines the small, dialect-free event hook contract shared by
// Core modules and Editions.
package hooks

import (
	"context"
	"errors"
	"strings"
	"sync"
)

var ErrInvalidEvent = errors.New("invalid hook event")

type Event struct {
	ID          string
	Type        string
	AggregateID string
	Version     int
}

type Handler func(context.Context, Event) error

// Dispatcher invokes registered handlers once per event ID and bounds the
// in-memory deduplication window. A failed handler is not marked as delivered,
// so a later retry can safely run it again.
type Dispatcher struct {
	mu       sync.Mutex
	maxSeen  int
	seen     map[string]struct{}
	order    []string
	handlers []Handler
}

func New(maxSeen int, handlers ...Handler) *Dispatcher {
	if maxSeen < 1 {
		maxSeen = 1
	}
	return &Dispatcher{maxSeen: maxSeen, seen: make(map[string]struct{}), handlers: append([]Handler(nil), handlers...)}
}

func (d *Dispatcher) Dispatch(ctx context.Context, event Event) error {
	if d == nil || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Type) == "" || strings.TrimSpace(event.AggregateID) == "" || event.Version <= 0 {
		return ErrInvalidEvent
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[event.ID]; ok {
		return nil
	}
	for _, handler := range d.handlers {
		if handler == nil {
			continue
		}
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	d.seen[event.ID] = struct{}{}
	d.order = append(d.order, event.ID)
	if len(d.order) > d.maxSeen {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.seen, oldest)
	}
	return nil
}
