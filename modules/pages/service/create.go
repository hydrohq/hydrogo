package service

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

type PageWriter interface {
	InsertDraft(context.Context, database.Executor, string, string) error
}

type AuditWriter interface {
	Record(context.Context, database.Executor, string, string, string) error
}

type OutboxWriter interface {
	Enqueue(context.Context, database.Executor, string, string, string, int) error
}

type Transactions interface {
	Within(context.Context, func(database.Executor) error) error
}

type CreateDraft struct {
	Transactions Transactions
	Pages        PageWriter
	Audit        AuditWriter
	Outbox       OutboxWriter
}

func (s CreateDraft) Execute(ctx context.Context, id, title string) error {
	if guards.IsNil(s.Transactions) || guards.IsNil(s.Pages) || guards.IsNil(s.Audit) || guards.IsNil(s.Outbox) {
		return ErrDependencyRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Pages.InsertDraft(ctx, tx, id, title); err != nil {
			return err
		}
		if err := audit.RecordVersion(ctx, tx, s.Audit, "audit-"+id, "page.created", id, 0, 1); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, "outbox-"+id, "page.created", id, 1)
	})
}
