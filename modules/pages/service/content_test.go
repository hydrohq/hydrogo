package service

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestNormalizePageTagsBoundsAndDeduplicates(t *testing.T) {
	tooLong := strings.Repeat("x", 65)
	tags := normalizePageTags([]string{" Go ", "go", "", tooLong, "Hydro"})
	if len(tags) != 2 || tags[0] != "go" || tags[1] != "hydro" {
		t.Fatalf("normalized tags = %#v", tags)
	}
	values := make([]string, 55)
	for i := range values {
		values[i] = "tag-" + strconv.Itoa(i)
	}
	if got := len(normalizePageTags(values)); got != 50 {
		t.Fatalf("normalized tag count = %d, want 50", got)
	}
}

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

type localePublicRepository struct {
	pages map[string]pages.Page
	seen  []string
}

func (r *localePublicRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{}, nil
}
func (r *localePublicRepository) GetPublishedByRoute(_ context.Context, _ database.Executor, locale, _ string) (pages.Page, error) {
	r.seen = append(r.seen, locale)
	page, ok := r.pages[locale]
	if !ok {
		return pages.Page{}, sql.ErrNoRows
	}
	return page, nil
}
func (r *localePublicRepository) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return nil, nil
}
func (r *localePublicRepository) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	return nil
}
func (r *localePublicRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return nil
}
func (r *localePublicRepository) Publish(context.Context, database.Executor, string, int) error {
	return nil
}

func TestGetPublishedUsesConfiguredLocaleFallback(t *testing.T) {
	repo := &localePublicRepository{pages: map[string]pages.Page{"en": {ID: "fallback", Locale: "en"}}}
	service := ContentService{Pages: repo, SupportedLocales: []string{"tr", "en"}, DefaultLocale: "en"}
	page, err := service.GetPublished(context.Background(), nil, "tr-TR", "home")
	if err != nil || page.ID != "fallback" {
		t.Fatalf("fallback lookup = %+v/%v", page, err)
	}
	if got, want := repo.seen, []string{"tr", "en"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("lookup chain = %#v, want %#v", got, want)
	}
}

func TestGetPublishedFallbackRequiresValidConfiguration(t *testing.T) {
	service := ContentService{Pages: &localePublicRepository{}, SupportedLocales: []string{"tr", "tr"}, DefaultLocale: "tr"}
	if _, err := service.GetPublished(context.Background(), nil, "tr", "home"); err != ErrInvalidPage {
		t.Fatalf("invalid fallback config error = %v", err)
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

func (*conflictRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return pages.ErrConflict
}

func TestUpdateDraftPropagatesOptimisticLockConflict(t *testing.T) {
	repo := &conflictRepository{}
	service := ContentService{Pages: repo, Transactions: testTransactions{}, Audit: testAudit{}, Outbox: testOutbox{}}
	err := service.UpdateDraft(context.Background(), pages.Page{ID: "page-1", Locale: "tr", Slug: "home", Title: "Home", Status: pages.StatusDraft}, 3)
	if err != pages.ErrConflict {
		t.Fatalf("error = %v, want optimistic-lock conflict", err)
	}
}

type publishPageRepository struct{ countingPublicRepository }

func (r *publishPageRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{ID: "page-1", Locale: "tr", Slug: "new-slug", PublishedRevision: func() *int { value := 2; return &value }()}, nil
}
func (*publishPageRepository) GetRevisionSlug(context.Context, database.Executor, string, int) (string, string, error) {
	return "en", "old-slug", nil
}

func TestPublishPurgesOldAndNewPublicRoutesAfterCommit(t *testing.T) {
	var purged []string
	repo := &publishPageRepository{}
	service := ContentService{
		Transactions: testTransactions{}, Pages: repo, Audit: testAudit{}, Outbox: testOutbox{},
		PublicCache:      cache.New[pages.Page](4, time.Minute),
		PublicRoutePurge: func(locale, slug string) { purged = append(purged, locale+"/"+slug) },
	}
	if err := service.Publish(context.Background(), "page-1", 3); err != nil {
		t.Fatal(err)
	}
	if len(purged) != 2 || purged[0] != "en/old-slug" || purged[1] != "tr/new-slug" {
		t.Fatalf("purged public routes = %#v", purged)
	}
}
