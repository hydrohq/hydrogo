package tenant

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type postgresExecStub struct {
	query string
	args  []any
}

func (s *postgresExecStub) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	s.query, s.args = query, args
	return nil, nil
}

func TestBindPostgresTransactionUsesTransactionLocalScope(t *testing.T) {
	scope, err := NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	stub := &postgresExecStub{}
	if err := BindPostgresTransaction(context.Background(), stub, scope); err != nil {
		t.Fatal(err)
	}
	if stub.query != "SELECT set_config('app.current_tenant_id', $1, true)" || len(stub.args) != 1 || stub.args[0] != "tenant-a" {
		t.Fatalf("RLS binding = %q %#v", stub.query, stub.args)
	}
}

func TestBindPostgresTransactionFailsClosedWithoutScope(t *testing.T) {
	if err := BindPostgresTransaction(context.Background(), &postgresExecStub{}, Scope{}); !errors.Is(err, ErrMissingScope) {
		t.Fatalf("missing scope error = %v", err)
	}
	if err := BindPostgresTransaction(context.Background(), nil, Scope{TenantID: "tenant-a", Slug: "alpha"}); !errors.Is(err, ErrMissingScope) {
		t.Fatalf("nil executor error = %v", err)
	}
}
