package portfolio

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
)

type Repository interface{ sectoral.Repository[Project] }

type Service struct {
	Transactions database.Transactor
	Repository   Repository
	Audit        sectoral.AuditWriter
}

func (s Service) Create(ctx context.Context, item Project) error {
	if s.Repository == nil {
		return sectoral.ErrInvalid
	}
	if s.Audit == nil {
		return sectoral.ErrAuditRequired
	}
	item.Version = 1
	if err := item.Validate(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.Create(ctx, tx, item); err != nil {
			return err
		}
		return sectoral.RecordAudit(ctx, tx, s.Audit, "portfolio", "created", item.ID, item.Version)
	})
}

func (s Service) Update(ctx context.Context, item Project) error {
	if s.Repository == nil {
		return sectoral.ErrInvalid
	}
	if s.Audit == nil {
		return sectoral.ErrAuditRequired
	}
	if item.Version < 1 {
		return sectoral.ErrInvalid
	}
	item.Version++
	if err := item.Validate(); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.Update(ctx, tx, item); err != nil {
			return err
		}
		return sectoral.RecordAudit(ctx, tx, s.Audit, "portfolio", "updated", item.ID, item.Version)
	})
}

func (s Service) Get(ctx context.Context, db database.Executor, id string) (Project, error) {
	if s.Repository == nil {
		return Project{}, sectoral.ErrInvalid
	}
	return s.Repository.Get(ctx, db, id)
}
func (s Service) List(ctx context.Context, db database.QueryExecutor, limit int) ([]Project, error) {
	if s.Repository == nil {
		return nil, sectoral.ErrInvalid
	}
	return s.Repository.List(ctx, db, limit)
}
func (s Service) ListPublished(ctx context.Context, db database.QueryExecutor, locale string, limit int) ([]Project, error) {
	if s.Repository == nil {
		return nil, sectoral.ErrInvalid
	}
	return s.Repository.ListPublished(ctx, db, locale, limit)
}
func (s Service) Delete(ctx context.Context, id string) error {
	if s.Repository == nil {
		return sectoral.ErrInvalid
	}
	if s.Audit == nil {
		return sectoral.ErrAuditRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.Delete(ctx, tx, id); err != nil {
			return err
		}
		return sectoral.RecordAudit(ctx, tx, s.Audit, "portfolio", "deleted", id, 0)
	})
}
func (s Service) Clone(ctx context.Context, db database.Executor, sourceID, newID, newSlug string) error {
	item, err := s.Get(ctx, db, sourceID)
	if err != nil {
		return err
	}
	return s.Create(ctx, item.Clone(newID, newSlug))
}
