package mailer

import (
	"context"
	"errors"
	"testing"
	"time"
)

type spoolFake struct {
	entry     SpoolEntry
	token     string
	claimed   bool
	completed bool
	retryAt   time.Time
	permanent bool
	lastError string
}

func (s *spoolFake) Claim(context.Context, string, time.Time, time.Duration) (SpoolEntry, string, bool, error) {
	return s.entry, s.token, s.claimed, nil
}
func (s *spoolFake) Delivered(context.Context, string, string, string, time.Time) error {
	s.completed = true
	return nil
}
func (s *spoolFake) Retry(_ context.Context, _, _, _ string, at time.Time, reason string) error {
	s.retryAt, s.lastError = at, reason
	return nil
}
func (s *spoolFake) FailPermanent(_ context.Context, _, _, _, reason string) error {
	s.permanent, s.lastError = true, reason
	return nil
}

type senderFake struct{ err error }

func (s senderFake) Send(context.Context, Message) error { return s.err }

func TestMailSpoolRetrySchedule(t *testing.T) {
	want := []time.Duration{5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour, 24 * time.Hour}
	for i, delay := range want {
		if got := RetryDelay(i + 1); got != delay {
			t.Errorf("RetryDelay(%d) = %v, want %v", i+1, got, delay)
		}
	}
	if RetryDelay(0) != 0 {
		t.Fatal("first send must be immediate")
	}
}

func TestSpoolWorkerRetriesAndMarksPermanentErrors(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	store := &spoolFake{entry: SpoolEntry{ID: "mail-1", Attempts: 1}, token: "lease", claimed: true}
	worker := SpoolWorker{Store: store, Sender: senderFake{err: errors.New("temporary outage")}, WorkerID: "worker-1", Now: func() time.Time { return now }}
	if worked, err := worker.RunOnce(context.Background()); err != nil || !worked {
		t.Fatalf("run = %v/%v", worked, err)
	}
	if want := now.Add(5 * time.Minute); !store.retryAt.Equal(want) || store.lastError != "temporary outage" {
		t.Fatalf("retry = %v error=%q", store.retryAt, store.lastError)
	}
	store = &spoolFake{entry: SpoolEntry{ID: "mail-2", Attempts: 1}, token: "lease", claimed: true}
	worker.Store = store
	worker.Sender = senderFake{err: ErrInvalidMessage}
	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !store.permanent {
		t.Fatal("invalid message was scheduled for retry")
	}
}

func TestSpoolWorkerSendsImmediatelyAndCompletes(t *testing.T) {
	store := &spoolFake{entry: SpoolEntry{ID: "mail-1", Attempts: 1}, token: "lease", claimed: true}
	worker := SpoolWorker{Store: store, Sender: senderFake{}, WorkerID: "worker-1"}
	worked, err := worker.RunOnce(context.Background())
	if err != nil || !worked || !store.completed {
		t.Fatalf("run = %v/%v completed=%v", worked, err, store.completed)
	}
}

func TestSpoolWorkerRunsBoundedScheduledTaskWithoutAnotherWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	called := make(chan time.Time, 1)
	worker := SpoolWorker{Store: &spoolFake{}, Sender: senderFake{}, WorkerID: "worker-1", OnSchedule: func(_ context.Context, now time.Time) error { called <- now; cancel(); return nil }}
	worker.Run(ctx, time.Millisecond)
	select {
	case now := <-called:
		if now.IsZero() {
			t.Fatal("scheduled callback received zero time")
		}
	case <-time.After(time.Second):
		t.Fatal("scheduled callback did not run")
	}
}
