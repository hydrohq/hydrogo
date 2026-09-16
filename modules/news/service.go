package news

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/content"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
)

type Repository interface {
	CreateDraft(context.Context, database.Executor, Item) error
	UpdateDraft(context.Context, database.Executor, Item, int) error
	Schedule(context.Context, database.Executor, string, int, time.Time) (int64, error)
	PublishScheduled(context.Context, database.Executor, string, int64, time.Time) error
	PublishNow(context.Context, database.Executor, string, int) error
	Retract(context.Context, database.Executor, string, int) error
	Get(context.Context, database.Executor, string) (Item, error)
	GetPublishedByRoute(context.Context, database.Executor, string, string) (Item, error)
	ListScheduled(context.Context, database.Executor, time.Time) ([]Item, error)
}

type AuditWriter interface {
	Record(context.Context, database.Executor, string, string, string) error
}

type OutboxWriter interface {
	Enqueue(context.Context, database.Executor, string, string, string, int) error
}

type Service struct {
	Transactions     database.Transactor
	Repository       Repository
	Audit            AuditWriter
	Outbox           OutboxWriter
	SupportedLocales []string
	DefaultLocale    string
	Workflow         v1.EditorialWorkflow
}

var ErrInvalidWorkflowTransition = errors.New("invalid editorial workflow transition")

func defaultEditorialWorkflow() v1.EditorialWorkflow {
	return v1.EditorialWorkflow{
		States: []v1.EditorialState{v1.EditorialDraft, v1.EditorialScheduled, v1.EditorialPublished, v1.EditorialRetracted},
		Transitions: map[v1.EditorialState][]v1.EditorialState{
			v1.EditorialDraft:     {v1.EditorialScheduled, v1.EditorialPublished},
			v1.EditorialScheduled: {v1.EditorialPublished, v1.EditorialDraft, v1.EditorialRetracted},
			v1.EditorialPublished: {v1.EditorialRetracted, v1.EditorialDraft},
			v1.EditorialRetracted: {v1.EditorialScheduled, v1.EditorialPublished},
		},
	}
}

func (s Service) editorialWorkflow() v1.EditorialWorkflow {
	if len(s.Workflow.States) == 0 {
		return defaultEditorialWorkflow()
	}
	return s.Workflow
}

func (s Service) validateTransition(ctx context.Context, tx database.Executor, id string, target v1.EditorialState) error {
	item, err := s.Repository.Get(ctx, tx, id)
	if err != nil {
		return err
	}
	if err := v1.ValidateEditorialTransition(s.editorialWorkflow(), v1.EditorialState(item.Status), target); err != nil {
		return ErrInvalidWorkflowTransition
	}
	return nil
}

// GetPublishedByRoute applies the configured exact → primary → default
// locale chain. It falls back only when a candidate has no published row.
func (s Service) GetPublishedByRoute(ctx context.Context, db database.Executor, locale, slug string) (Item, error) {
	if err := s.requireRepository(); err != nil {
		return Item{}, err
	}
	requested := strings.TrimSpace(locale)
	if requested == "" {
		return Item{}, ErrInvalidItem
	}
	locales := []string{requested}
	if len(s.SupportedLocales) > 0 {
		chain, err := i18n.LocaleFallbackChain(requested, s.SupportedLocales, s.DefaultLocale)
		if err != nil {
			return Item{}, ErrInvalidItem
		}
		locales = chain
	}
	for _, candidate := range locales {
		item, err := s.Repository.GetPublishedByRoute(ctx, db, candidate, slug)
		if err == nil {
			return item, nil
		}
		if len(locales) == 1 || !errors.Is(err, sql.ErrNoRows) {
			return Item{}, err
		}
	}
	return Item{}, sql.ErrNoRows
}

func (s Service) requireRepository() error {
	if guards.IsNil(s.Repository) {
		return ErrRepositoryRequired
	}
	return nil
}

func (s Service) CreateDraft(ctx context.Context, item Item) error {
	if err := content.ValidateRichText(item.Body); err != nil {
		return ErrInvalidItem
	}
	if err := Canonicalize(&item); err != nil {
		return ErrInvalidItem
	}
	if err := Validate(item); err != nil {
		return err
	}
	if err := s.requireRepository(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.CreateDraft(ctx, tx, item); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.created", item.ID, 1)
	})
}

func (s Service) UpdateDraft(ctx context.Context, item Item, expectedVersion int) error {
	if err := content.ValidateRichText(item.Body); err != nil {
		return ErrInvalidItem
	}
	if err := Canonicalize(&item); err != nil {
		return ErrInvalidItem
	}
	if err := Validate(item); err != nil || expectedVersion <= 0 {
		return ErrInvalidItem
	}
	if err := s.requireRepository(); err != nil {
		return err
	}
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.UpdateDraft(ctx, tx, item, expectedVersion); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.updated", item.ID, expectedVersion+1)
	})
	if errors.Is(err, ErrConflict) {
		return ErrConflict
	}
	return err
}

func (s Service) Schedule(ctx context.Context, id string, expectedVersion int, at time.Time) error {
	if strings.TrimSpace(id) == "" || expectedVersion <= 0 || at.IsZero() {
		return ErrInvalidItem
	}
	if err := s.requireRepository(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.validateTransition(ctx, tx, id, v1.EditorialScheduled); err != nil {
			return err
		}
		generation, err := s.Repository.Schedule(ctx, tx, id, expectedVersion, at.UTC())
		if err != nil {
			return err
		}
		return s.events(ctx, tx, "news.scheduled", id, int(generation))
	})
}

func (s Service) RunScheduled(ctx context.Context, id string, generation int64, now time.Time) error {
	if strings.TrimSpace(id) == "" || generation <= 0 {
		return ErrScheduleStale
	}
	if err := s.requireRepository(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.validateTransition(ctx, tx, id, v1.EditorialPublished); err != nil {
			return err
		}
		if err := s.Repository.PublishScheduled(ctx, tx, id, generation, now.UTC()); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.published", id, int(generation))
	})
}

func (s Service) Publish(ctx context.Context, id string, expectedVersion int) error {
	if err := s.requireRepository(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.validateTransition(ctx, tx, id, v1.EditorialPublished); err != nil {
			return err
		}
		if err := s.Repository.PublishNow(ctx, tx, id, expectedVersion); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.published", id, expectedVersion)
	})
}

func (s Service) Retract(ctx context.Context, id string, expectedVersion int) error {
	if err := s.requireRepository(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.validateTransition(ctx, tx, id, v1.EditorialRetracted); err != nil {
			return err
		}
		if err := s.Repository.Retract(ctx, tx, id, expectedVersion); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.retracted", id, expectedVersion)
	})
}

func (s Service) events(ctx context.Context, tx database.Executor, action, id string, version int) error {
	if s.Audit != nil {
		if err := s.Audit.Record(ctx, tx, fmt.Sprintf("news-audit-%s-%s-%d", action, id, version), action, id); err != nil {
			return err
		}
	}
	if s.Outbox != nil {
		return s.Outbox.Enqueue(ctx, tx, fmt.Sprintf("news-outbox-%s-%s-%d", action, id, version), action, id, 1)
	}
	return nil
}
