package news

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestRetryDelayIsBoundedExponential(t *testing.T) {
	base := 10 * time.Second
	if got := retryDelay(base, 1); got != 10*time.Second {
		t.Fatalf("first retry = %s", got)
	}
	if got := retryDelay(base, 3); got != 40*time.Second {
		t.Fatalf("third retry = %s", got)
	}
	if got := retryDelay(base, 20); got != time.Hour {
		t.Fatalf("capped retry = %s", got)
	}
}

func TestMalformedJobRecordsDeadLetterCallback(t *testing.T) {
	queue := jobs.NewQueue(1)
	created := time.Now().UTC()
	if err := queue.Enqueue(jobs.Job{ID: "bad", Type: "news.publish", Payload: "not-json", AvailableAt: created, CreatedAt: created}); err != nil {
		t.Fatal(err)
	}
	deadLetters := 0
	worker := Worker{Queue: queue, OnDeadLetter: func() { deadLetters++ }}
	processed, err := worker.RunOnce(context.Background(), "worker-1", created)
	if err != nil || !processed || deadLetters != 1 {
		t.Fatalf("malformed job result = processed:%v err:%v dead_letters:%d", processed, err, deadLetters)
	}
}

func TestTenantWorkerRequiresScopeWhenScheduling(t *testing.T) {
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	worker := Worker{Queue: jobs.NewQueue(2), TenantScopes: []tenant.Scope{scope}}
	if err := worker.EnqueueScheduleContext(context.Background(), "item-1", 1, time.Now()); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing tenant scope error = %v", err)
	}
	if err := worker.EnqueueScheduleContext(tenant.WithScope(context.Background(), scope), "item-1", 1, time.Now()); err != nil {
		t.Fatalf("scoped schedule enqueue = %v", err)
	}
}

func TestServiceFailureRecordsRetryCallback(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "worker.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	queue := jobs.NewQueue(1)
	if err := queue.Enqueue(jobs.Job{ID: "retry", Type: "news.publish", Payload: `{"item_id":"item-1","generation":1}`, AvailableAt: now, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	retries := 0
	worker := Worker{
		Queue:   queue,
		Service: Service{Transactions: database.NewTransactor(db), Repository: failingRepository{}},
		Retry:   time.Second,
		OnRetry: func() { retries++ },
	}
	processed, err := worker.RunOnce(context.Background(), "worker-1", now)
	if err != nil || !processed || retries != 1 {
		t.Fatalf("retry result = processed:%v err:%v retries:%d", processed, err, retries)
	}
}

type failingRepository struct{}

func (failingRepository) CreateDraft(context.Context, database.Executor, Item) error {
	return errors.New("not used")
}
func (failingRepository) UpdateDraft(context.Context, database.Executor, Item, int) error {
	return errors.New("not used")
}
func (failingRepository) Schedule(context.Context, database.Executor, string, int, time.Time) (int64, error) {
	return 0, errors.New("not used")
}
func (failingRepository) PublishScheduled(context.Context, database.Executor, string, int64, time.Time) error {
	return errors.New("temporary publish failure")
}
func (failingRepository) PublishNow(context.Context, database.Executor, string, int) error {
	return errors.New("not used")
}
func (failingRepository) Retract(context.Context, database.Executor, string, int) error {
	return errors.New("not used")
}
func (failingRepository) Get(context.Context, database.Executor, string) (Item, error) {
	return Item{}, errors.New("not used")
}
func (failingRepository) GetPublishedByRoute(context.Context, database.Executor, string, string) (Item, error) {
	return Item{}, errors.New("not used")
}
func (failingRepository) ListScheduled(context.Context, database.Executor, time.Time) ([]Item, error) {
	return nil, errors.New("not used")
}
