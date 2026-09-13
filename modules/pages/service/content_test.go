package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type countingPublicRepository struct {
	mu    sync.Mutex
	loads int
	page  pages.Page
}

func (r *countingPublicRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{}, nil
}
func (r *countingPublicRepository) GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error) {
	r.mu.Lock()
	r.loads++
	r.mu.Unlock()
	return r.page, nil
}
func (r *countingPublicRepository) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return []pages.Page{r.page}, nil
}
func (r *countingPublicRepository) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	return nil
}
func (r *countingPublicRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return nil
}
func (r *countingPublicRepository) Publish(context.Context, database.Executor, string, int) error {
	return nil
}

func (r *countingPublicRepository) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loads
}

func TestGetPublishedCoalescesConcurrentDBMisses(t *testing.T) {
	repo := &countingPublicRepository{page: pages.Page{ID: "page-1", Locale: "en", Slug: "home", Title: "Home"}}
	service := ContentService{Pages: repo, PublicCache: cache.New[pages.Page](4, time.Minute)}
	const callers = 64
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			page, err := service.GetPublished(context.Background(), nil, "EN", "/home/")
			if err != nil || page.ID != "page-1" {
				t.Errorf("public lookup = %+v/%v", page, err)
			}
		}()
	}
	wg.Wait()
	if got := repo.count(); got != 1 {
		t.Fatalf("repository calls = %d, want 1", got)
	}
}

func TestListPublishedRejectsInvalidPublicQuery(t *testing.T) {
	service := ContentService{Pages: &countingPublicRepository{}}
	for _, test := range []struct {
		locale string
		limit  int
	}{{"", 1}, {"tr", 0}, {"tr", 101}} {
		if _, err := service.ListPublished(context.Background(), nil, test.locale, test.limit); err != ErrInvalidPage {
			t.Errorf("ListPublished(%q, %d) error = %v", test.locale, test.limit, err)
		}
	}
}

type conflictRepository struct{ countingPublicRepository }

type testTransactions struct{}

func (testTransactions) Within(_ context.Context, fn func(database.Executor) error) error {
	return fn(nil)
}

type testAudit struct{}

func (testAudit) Record(context.Context, database.Executor, string, string, string) error { return nil }

type testOutbox struct{}

func (testOutbox) Enqueue(context.Context, database.Executor, string, string, string, int) error {
	return nil
}

func (conflictRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return pages.ErrConflict
}

func TestUpdateDraftPropagatesOptimisticLockConflict(t *testing.T) {
	repo := conflictRepository{}
	service := ContentService{Pages: repo, Transactions: testTransactions{}, Audit: testAudit{}, Outbox: testOutbox{}}
	err := service.UpdateDraft(context.Background(), pages.Page{ID: "page-1", Locale: "tr", Slug: "home", Title: "Home", Status: pages.StatusDraft}, 3)
	if err != pages.ErrConflict {
		t.Fatalf("error = %v, want optimistic-lock conflict", err)
	}
}
