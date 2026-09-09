package service

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	if s.Transactions == nil || s.Pages == nil || s.Audit == nil || s.Outbox == nil {
		return ErrDependencyRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Pages.InsertDraft(ctx, tx, id, title); err != nil {
			return err
		}
		if err := s.Audit.Record(ctx, tx, "audit-"+id, "page.created", id); err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, tx, "outbox-"+id, "page.created", id, 1)
	})
}
