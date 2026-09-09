package sqlstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/jobs"
)

func TestStoreRejectsMissingExecutor(t *testing.T) {
	store := Store{Dialect: database.SQLite}
	if err := store.Enqueue(context.Background(), nil, jobs.Job{ID: "job", Type: "test"}); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("enqueue error = %v, want %v", err, database.ErrDatabaseRequired)
	}
	if _, _, _, err := store.Claim(context.Background(), nil, "worker", time.Now(), time.Minute); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("claim error = %v, want %v", err, database.ErrDatabaseRequired)
	}
}
