package sqlstore

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
)

// TestMySQLAuditChainSerializesConcurrentWriters is the MySQL twin of the
// PostgreSQL contract: writers racing for one tenant's head queue on its row
// lock (SELECT ... FOR UPDATE) and leave a chain without forks or gaps.
func TestMySQLAuditChainSerializesConcurrentWriters(t *testing.T) {
	db := testmysql.Open(t)
	testmysql.MigrateAll(t, db)
	ctx := context.Background()
	SetChain(true)
	defer SetChain(false)
	store := Store{Dialect: database.MySQL}
	const writers, each = 8, 10
	var wg sync.WaitGroup
	errs := make(chan error, writers*each)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for n := 0; n < each; n++ {
				errs <- database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
					return store.Record(ctx, tx, fmt.Sprintf("w%d-%d", w, n), "content.updated", "r")
				})
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	report, err := store.VerifyChain(ctx, db, "", nil)
	if err != nil || len(report.Problems) != 0 || report.Links != writers*each {
		t.Fatalf("chain: %+v err=%v", report, err)
	}
}
