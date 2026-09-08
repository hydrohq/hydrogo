package news

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	Transactions database.Transactor
	Repository   Repository
	Audit        AuditWriter
	Outbox       OutboxWriter
}

func (s Service) CreateDraft(ctx context.Context, item Item) error {
	if err := Validate(item); err != nil {
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
	if err := Validate(item); err != nil || expectedVersion <= 0 {
		return ErrInvalidItem
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
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
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
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.PublishScheduled(ctx, tx, id, generation, now.UTC()); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.published", id, int(generation))
	})
}

func (s Service) Publish(ctx context.Context, id string, expectedVersion int) error {
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.PublishNow(ctx, tx, id, expectedVersion); err != nil {
			return err
		}
		return s.events(ctx, tx, "news.published", id, expectedVersion)
	})
}

func (s Service) Retract(ctx context.Context, id string, expectedVersion int) error {
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
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
