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
