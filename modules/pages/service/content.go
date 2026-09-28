package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

var (
	ErrInvalidPage        = errors.New("invalid page")
	ErrConflict           = pages.ErrConflict
	ErrDependencyRequired = errors.New("page service dependency is required")
)

type ContentRepository interface {
	GetContent(context.Context, database.Executor, string) (pages.Page, error)
	GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error)
	ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error)
	CreateDraftContent(context.Context, database.Executor, pages.Page) error
	UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error
	Publish(context.Context, database.Executor, string, int) error
}

// ListPublished exposes a bounded, locale-scoped public projection for
// headless discovery surfaces. It never reads draft fields or revisions.
func (s ContentService) ListPublished(ctx context.Context, db database.Executor, locale string, limit int) ([]pages.Page, error) {
	if guards.IsNil(s.Pages) {
		return nil, ErrDependencyRequired
	}
	locale = canonical.Locale(locale)
	if locale == "" || limit < 1 || limit > 100 {
		return nil, ErrInvalidPage
	}
	return s.Pages.ListPublished(ctx, db, locale, limit)
}

func (s ContentService) Related(ctx context.Context, db database.Executor, current pages.Page, limit int) ([]pages.Page, error) {
	if guards.IsNil(s.Pages) {
		return nil, ErrDependencyRequired
	}
	if limit < 1 || limit > 10 || current.ID == "" || current.Locale == "" {
		return nil, ErrInvalidPage
	}
	if repository, ok := s.Pages.(interface {
		ListRelatedPublished(context.Context, database.Executor, string, string, int) ([]pages.Page, error)
	}); ok {
		return repository.ListRelatedPublished(ctx, db, current.ID, canonical.Locale(current.Locale), limit)
	}
	return nil, ErrDependencyRequired
}

func (s ContentService) Get(ctx context.Context, db database.Executor, pageID string) (pages.Page, error) {
	if strings.TrimSpace(pageID) == "" {
		return pages.Page{}, ErrInvalidPage
	}
	if guards.IsNil(s.Pages) {
		return pages.Page{}, ErrDependencyRequired
	}
	return s.Pages.GetContent(ctx, db, pageID)
}

type ContentService struct {
	Transactions Transactions
	Pages        ContentRepository
	Audit        AuditWriter
	Outbox       OutboxWriter
	PublicCache  interface {
		GetOrLoad(string, func() (pages.Page, error)) (pages.Page, error)
		GetOrLoadAtGeneration(string, uint64, func() (pages.Page, error)) (pages.Page, error)
		Clear()
		Stats() cache.Stats
	}
	Sanitize func(string) string
	// SupportedLocales and DefaultLocale enable an opt-in, deterministic
	// public lookup fallback. An empty SupportedLocales preserves exact-only
	// behavior for deployments that do not configure i18n.
	SupportedLocales []string
	DefaultLocale    string
}

func (s ContentService) GetPublished(ctx context.Context, db database.Executor, locale, slug string) (pages.Page, error) {
	if guards.IsNil(s.Pages) {
		return pages.Page{}, ErrDependencyRequired
	}
	normalLocale := canonical.Locale(locale)
	normalSlug, err := canonical.Path(slug)
	if err != nil || normalLocale == "" || normalSlug == "" {
		return pages.Page{}, ErrInvalidPage
	}
	locales := []string{normalLocale}
	if len(s.SupportedLocales) > 0 {
		chain, fallbackErr := i18n.LocaleFallbackChain(normalLocale, s.SupportedLocales, s.DefaultLocale)
		if fallbackErr != nil {
			return pages.Page{}, ErrInvalidPage
		}
		locales = chain
	}
	for _, locale := range locales {
		key := locale + ":" + normalSlug
		load := func() (pages.Page, error) {
			return s.Pages.GetPublishedByRoute(ctx, db, locale, normalSlug)
		}
		var page pages.Page
		var err error
		if s.PublicCache != nil {
			if generation, ok := cache.GenerationFromContext(ctx); ok {
				page, err = s.PublicCache.GetOrLoadAtGeneration(key, generation, load)
			} else {
				page, err = s.PublicCache.GetOrLoad(key, load)
			}
		} else {
			page, err = load()
		}
		if err == nil {
			return page, nil
		}
		if len(locales) == 1 || !errors.Is(err, sql.ErrNoRows) {
			return pages.Page{}, err
		}
	}
	return pages.Page{}, sql.ErrNoRows
}

func (s ContentService) CreateDraft(ctx context.Context, page pages.Page) error {
	if s.Sanitize != nil {
		page.Body = s.Sanitize(page.Body)
	}
	if err := pages.ValidateBlocks(page.Blocks); err != nil {
		return ErrInvalidPage
	}
	if err := validatePage(&page); err != nil {
		return err
	}
	if err := s.requireMutationDependencies(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Pages.CreateDraftContent(ctx, tx, page); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, "audit-"+page.ID, "page.created", page.ID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, "outbox-"+page.ID, "page.created", page.ID, 1)
	})
}

func (s ContentService) UpdateDraft(ctx context.Context, page pages.Page, expectedVersion int) error {
	if s.Sanitize != nil {
		page.Body = s.Sanitize(page.Body)
	}
	if err := pages.ValidateBlocks(page.Blocks); err != nil {
		return ErrInvalidPage
	}
	if err := validatePage(&page); err != nil {
		return err
	}
	if err := s.requireMutationDependencies(); err != nil {
		return err
	}
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Pages.UpdateDraftContent(ctx, tx, page, expectedVersion); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, "audit-update-"+page.ID+"-"+fmt.Sprint(expectedVersion+1), "page.updated", page.ID); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, "outbox-update-"+page.ID+"-"+fmt.Sprint(expectedVersion+1), "page.updated", page.ID, 1)
	})
	if errors.Is(err, pages.ErrConflict) {
		return ErrConflict
	}
	return err
}

func (s ContentService) Publish(ctx context.Context, pageID string, expectedVersion int) error {
	if strings.TrimSpace(pageID) == "" || expectedVersion <= 0 {
		return ErrInvalidPage
	}
	if err := s.requireMutationDependencies(); err != nil {
		return err
	}
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Pages.Publish(ctx, tx, pageID, expectedVersion); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, "audit-publish-"+pageID+"-"+fmt.Sprint(expectedVersion+1), "page.published", pageID); err != nil {
			return err
		}
		if err := s.Outbox.Enqueue(ctx, tx, "outbox-publish-"+pageID+"-"+fmt.Sprint(expectedVersion+1), "page.published", pageID, 1); err != nil {
			return err
		}
		return nil
	})
	if err == nil && s.PublicCache != nil {
		// Invalidate only after the publication transaction commits. Clearing
		// inside the transaction can create a refill race for rolled-back data.
		s.PublicCache.Clear()
	}
	return err
}

func (s ContentService) requireMutationDependencies() error {
	if guards.IsNil(s.Transactions) || guards.IsNil(s.Pages) || guards.IsNil(s.Audit) || guards.IsNil(s.Outbox) {
		return ErrDependencyRequired
	}
	return nil
}

func validatePage(page *pages.Page) error {
	page.Locale = canonical.Locale(page.Locale)
	var err error
	page.Slug, err = canonical.Path(page.Slug)
	page.Title = strings.TrimSpace(page.Title)
	page.Tags = normalizePageTags(page.Tags)
	if err != nil || page.ID == "" || len(page.ID) > 128 || page.Locale == "" || len(page.Locale) > 16 || page.Slug == "" || len(page.Slug) > 256 || page.Title == "" || len(page.Title) > 256 || len(page.Body) > 1<<20 {
		return ErrInvalidPage
	}
	if page.Status == "" {
		page.Status = pages.StatusDraft
	}
	if page.Status != pages.StatusDraft {
		return ErrInvalidPage
	}
	return nil
}

func normalizePageTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, min(len(tags), 50))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || len(tag) > 64 {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
		if len(result) == 50 {
			break
		}
	}
	return result
}
