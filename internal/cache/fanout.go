package cache

import "sync"

type Invalidation struct {
	Key        string
	Generation uint64
}

// Fanout is a bounded process-local invalidation bus. A slow subscriber is
// marked for resynchronization instead of causing unbounded memory growth.
type Fanout struct {
	mu      sync.Mutex
	nextID  uint64
	closed  bool
	current uint64
	subs    map[uint64]*fanoutSub
}

type fanoutSub struct {
	ch       chan Invalidation
	resyncCh chan struct{}
	resync   bool
}

type Subscription struct {
	Events <-chan Invalidation
	Resync <-chan struct{}
	cancel func()
}

func NewFanout(buffer int) *Fanout {
	if buffer < 1 {
		buffer = 1
	}
	return &Fanout{subs: make(map[uint64]*fanoutSub), current: 1}
}

func (f *Fanout) Subscribe(buffer int) *Subscription {
	if buffer < 1 {
		buffer = 1
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return &Subscription{Events: closedInvalidations(), Resync: closedSignals()}
	}
	f.nextID++
	id := f.nextID
	resync := make(chan struct{}, 1)
	sub := &fanoutSub{ch: make(chan Invalidation, buffer), resyncCh: resync}
	f.subs[id] = sub
	return &Subscription{Events: sub.ch, Resync: resync, cancel: func() { f.unsubscribe(id) }}
}

func (s *Subscription) Close() {
	if s != nil && s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (f *Fanout) Publish(key string) uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return f.current
	}
	f.current++
	event := Invalidation{Key: key, Generation: f.current}
	for _, sub := range f.subs {
		select {
		case sub.ch <- event:
		default:
			if !sub.resync {
				sub.resync = true
				sub.resyncCh <- struct{}{}
			}
		}
	}
	return f.current
}

func (f *Fanout) Generation() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

func (f *Fanout) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return
	}
	f.closed = true
	for id, sub := range f.subs {
		close(sub.ch)
		delete(f.subs, id)
	}
}

func (f *Fanout) unsubscribe(id uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sub, ok := f.subs[id]; ok {
		close(sub.ch)
		delete(f.subs, id)
	}
}

func closedInvalidations() <-chan Invalidation { ch := make(chan Invalidation); close(ch); return ch }
func closedSignals() <-chan struct{}           { ch := make(chan struct{}); close(ch); return ch }
