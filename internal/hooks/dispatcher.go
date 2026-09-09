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

type dispatchEventContextKey struct{}

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
	inFlight map[string]*delivery
}

type delivery struct {
	done chan struct{}
	err  error
}

func New(maxSeen int, handlers ...Handler) *Dispatcher {
	if maxSeen < 1 {
		maxSeen = 1
	}
	return &Dispatcher{maxSeen: maxSeen, seen: make(map[string]struct{}), handlers: append([]Handler(nil), handlers...), inFlight: make(map[string]*delivery)}
}

func (d *Dispatcher) Dispatch(ctx context.Context, event Event) error {
	if d == nil || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Type) == "" || strings.TrimSpace(event.AggregateID) == "" || event.Version <= 0 {
		return ErrInvalidEvent
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if currentID, ok := ctx.Value(dispatchEventContextKey{}).(string); ok && currentID == event.ID {
		return nil
	}
	d.mu.Lock()
	if d.seen == nil {
		d.seen = make(map[string]struct{})
	}
	if d.inFlight == nil {
		d.inFlight = make(map[string]*delivery)
	}
	if _, ok := d.seen[event.ID]; ok {
		d.mu.Unlock()
		return nil
	}
	if current, ok := d.inFlight[event.ID]; ok {
		d.mu.Unlock()
		select {
		case <-current.done:
			return current.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	delivery := &delivery{done: make(chan struct{})}
	d.inFlight[event.ID] = delivery
	handlers := append([]Handler(nil), d.handlers...)
	d.mu.Unlock()

	var err error
	handlerContext := context.WithValue(ctx, dispatchEventContextKey{}, event.ID)
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		if err = handler(handlerContext, event); err != nil {
			break
		}
	}

	d.mu.Lock()
	delete(d.inFlight, event.ID)
	delivery.err = err
	if err == nil {
		d.seen[event.ID] = struct{}{}
		d.order = append(d.order, event.ID)
		if len(d.order) > d.maxSeen {
			oldest := d.order[0]
			d.order = d.order[1:]
			delete(d.seen, oldest)
		}
	}
	close(delivery.done)
	d.mu.Unlock()
	return err
}
