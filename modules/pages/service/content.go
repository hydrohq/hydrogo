package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
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
	CreateDraftContent(context.Context, database.Executor, pages.Page) error
	UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error
	Publish(context.Context, database.Executor, string, int) error
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
	PublicCache  *cache.Cache[pages.Page]
	Sanitize     func(string) string
}

func (s ContentService) GetPublished(ctx context.Context, db database.Executor, locale, slug string) (pages.Page, error) {
	if guards.IsNil(s.Pages) {
		return pages.Page{}, ErrDependencyRequired
	}
	key := strings.ToLower(strings.Trim(locale, "/")) + ":" + strings.Trim(strings.ToLower(slug), "/")
	load := func() (pages.Page, error) {
		return s.Pages.GetPublishedByRoute(ctx, db, strings.ToLower(strings.TrimSpace(locale)), strings.Trim(strings.ToLower(strings.TrimSpace(slug)), "/"))
	}
	if s.PublicCache != nil {
		if generation, ok := cache.GenerationFromContext(ctx); ok {
			return s.PublicCache.GetOrLoadAtGeneration(key, generation, load)
		}
		return s.PublicCache.GetOrLoad(key, load)
	}
	return load()
}

func (s ContentService) CreateDraft(ctx context.Context, page pages.Page) error {
	if s.Sanitize != nil {
		page.Body = s.Sanitize(page.Body)
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
	page.Locale = strings.ToLower(strings.TrimSpace(page.Locale))
	page.Slug = strings.Trim(strings.ToLower(strings.TrimSpace(page.Slug)), "/")
	page.Title = strings.TrimSpace(page.Title)
	if page.ID == "" || page.Locale == "" || page.Slug == "" || page.Title == "" {
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
