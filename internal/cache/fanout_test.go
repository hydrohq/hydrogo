package cache

import "testing"

func TestFanoutBoundsSlowSubscriberAndSignalsResync(t *testing.T) {
	fanout := NewFanout(1)
	sub := fanout.Subscribe(1)
	defer sub.Close()
	first := fanout.Publish("page:1")
	if first != 2 {
		t.Fatalf("first generation = %d", first)
	}
	fanout.Publish("page:2")
	select {
	case <-sub.Resync:
	default:
		t.Fatal("slow subscriber was not marked for resync")
	}
	event := <-sub.Events
	if event.Key != "page:1" || event.Generation != 2 {
		t.Fatalf("event = %#v", event)
	}
	if fanout.Generation() != 3 {
		t.Fatalf("generation = %d", fanout.Generation())
	}
	if sub.Reconcile(2) || !sub.Reconcile(3) {
		t.Fatal("generation reconciliation result is incorrect")
	}
}

func TestFanoutCloseClosesSubscribers(t *testing.T) {
	fanout := NewFanout(1)
	sub := fanout.Subscribe(1)
	fanout.Close()
	if _, ok := <-sub.Events; ok {
		t.Fatal("subscriber channel remained open")
	}
	if fanout.Publish("ignored") != 1 {
		t.Fatal("closed fanout advanced generation")
	}
}

func TestFanoutConcurrentSubscribersAndRace(t *testing.T) {
	fanout := NewFanout(4)
	subscribers := make([]*Subscription, 20)
	for i := range subscribers {
		subscribers[i] = fanout.Subscribe(2)
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			fanout.Publish("event")
		}
		close(done)
	}()

	for _, sub := range subscribers {
		go func(s *Subscription) {
			for {
				select {
				case <-s.Events:
				case <-s.Resync:
					s.Reconcile(fanout.Generation())
				case <-done:
					return
				}
			}
		}(sub)
	}

	<-done
	for _, sub := range subscribers {
		sub.Close()
	}
}
