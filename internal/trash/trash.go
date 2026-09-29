package trash

import (
	"context"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

const Retention = 30 * 24 * time.Hour

var ErrNotFound = errors.New("trash record not found")

type Kind string

const (
	Pages       Kind = "pages"
	News        Kind = "news"
	Submissions Kind = "submissions"
	Media       Kind = "media"
)

type Entry struct {
	Kind      Kind
	ID        string
	Label     string
	DeletedAt time.Time
	Trashed   bool
}

type Repository interface {
	List(context.Context, int) ([]Entry, error)
	Move(context.Context, Kind, string, time.Time) error
	Restore(context.Context, Kind, string, time.Time) error
}

type TransactionalRepository interface {
	MoveWithin(context.Context, database.Executor, Kind, string, time.Time) error
	RestoreWithin(context.Context, database.Executor, Kind, string, time.Time) error
}

type TransactionRunner interface {
	Within(context.Context, func(database.Executor) error) error
}

type Service struct {
	Repository   Repository
	Transactions TransactionRunner
	Audit        audit.Writer
}

func (s Service) List(ctx context.Context, limit int) ([]Entry, error) {
	if s.Repository == nil {
		return nil, errors.New("trash repository is required")
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return s.Repository.List(ctx, limit)
}

func (s Service) Move(ctx context.Context, kind Kind, id string, now time.Time) error {
	if s.Repository == nil || !valid(kind) || id == "" || len(id) > 128 {
		return ErrNotFound
	}
	return s.mutate(ctx, kind, id, now.UTC(), false)
}

func (s Service) Restore(ctx context.Context, kind Kind, id string, now time.Time) error {
	if s.Repository == nil || !valid(kind) || id == "" || len(id) > 128 {
		return ErrNotFound
	}
	return s.mutate(ctx, kind, id, now.UTC().Add(-Retention), true)
}

func (s Service) mutate(ctx context.Context, kind Kind, id string, stamp time.Time, restore bool) error {
	transactional, supportsTransactions := s.Repository.(TransactionalRepository)
	if s.Transactions == nil || !supportsTransactions {
		if s.Audit != nil {
			return errors.New("trash audit requires a transactional repository")
		}
		if restore {
			return s.Repository.Restore(ctx, kind, id, stamp)
		}
		return s.Repository.Move(ctx, kind, id, stamp)
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if restore {
			err = transactional.RestoreWithin(ctx, tx, kind, id, stamp)
		} else {
			err = transactional.MoveWithin(ctx, tx, kind, id, stamp)
		}
		if err != nil || s.Audit == nil {
			return err
		}
		action := "content.trashed"
		if restore {
			action = "content.restored"
		}
		auditID := "trash-" + action + "-" + string(kind) + "-" + id + "-" + stamp.Format("20060102150405.000000000")
		return s.Audit.Record(ctx, tx, auditID, action, string(kind)+":"+id)
	})
}

func valid(kind Kind) bool {
	switch kind {
	case Pages, News, Submissions, Media:
		return true
	default:
		return false
	}
}
