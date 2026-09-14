package sqlstore

import (
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapMemberCreateErrorNormalizesPostgresUniqueViolation(t *testing.T) {
	err := mapMemberCreateError(&pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"})
	if !errors.Is(err, members.ErrMemberExists) {
		t.Fatalf("expected ErrMemberExists, got %v", err)
	}
}

func TestMapMemberCreateErrorPreservesUnrelatedFailure(t *testing.T) {
	original := errors.New("connection reset")
	if got := mapMemberCreateError(original); !errors.Is(got, original) {
		t.Fatalf("unrelated error changed: %v", got)
	}
}
