package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestAuthStoresRejectMissingDatabaseAndTransactions(t *testing.T) {
	ctx := context.Background()
	if _, err := NewUserStore(nil, database.SQLite).GetByID(ctx, "user"); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing user database error = %v", err)
	}
	if _, err := NewSessionStore(nil, database.SQLite).GetByTokenHash(ctx, "token"); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing session database error = %v", err)
	}

	db := &sql.DB{}
	store := NewUserStore(db, database.SQLite)
	if err := store.CreateUserTx(ctx, nil, &auth.User{}); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing transaction error = %v", err)
	}
	if err := store.PutFactor(ctx, nil, "user", nil, false); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing MFA transaction error = %v", err)
	}
}
