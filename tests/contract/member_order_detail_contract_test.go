package contract_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	commerce "github.com/ardaozyildirim/hydrogo/modules/commerce"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func TestSQLiteMemberOrderDetailOwnershipContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "member-order-detail.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	runMemberOrderDetailOwnershipContract(t, db, database.SQLite)
}

func TestPostgresMemberOrderDetailOwnershipContract(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMemberOrderDetailOwnershipContract(t, openIsolatedPostgres(t, context.Background()), database.Postgres)
}

func runMemberOrderDetailOwnershipContract(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	now := time.Now().UTC()
	q := func(parts ...string) string {
		query := ""
		for i, part := range parts {
			if i > 0 {
				query += ","
			}
			query += part
		}
		return query
	}
	ph := func(position int) string {
		if dialect == database.Postgres {
			return "$" + strconv.Itoa(position)
		}
		return "?"
	}
	orderQuery := "INSERT INTO commerce_orders (id,user_id,idempotency_key,idempotency_hash,currency,total_cents,payment_status,shipping_status,created_at) VALUES (" + q(ph(1), ph(2), ph(3), ph(4), ph(5), ph(6), ph(7), ph(8), ph(9)) + ")"
	itemQuery := "INSERT INTO commerce_order_items (id,order_id,product_id,sku,title,quantity,unit_cents,total_cents) VALUES (" + q(ph(1), ph(2), ph(3), ph(4), ph(5), ph(6), ph(7), ph(8)) + ")"
	argsOrder := []any{"detail-order", "member-detail", "detail-key", "detail-hash", "TRY", 1250, "paid", "pending", now}
	argsItem := []any{"detail-item", "detail-order", "p-1", "SKU-1", "Reader item", 1, 1250, 1250}
	if _, err := db.ExecContext(ctx, orderQuery, argsOrder...); err != nil {
		t.Fatalf("insert order (%s): %v", dialect, err)
	}
	if _, err := db.ExecContext(ctx, itemQuery, argsItem...); err != nil {
		t.Fatalf("insert order item (%s): %v", dialect, err)
	}
	reader := commerce.CustomerOrderReader{Reader: commercestore.Store{Dialect: dialect}}
	detail, err := reader.GetOrderByUserID(ctx, db, "member-detail", "detail-order")
	if err != nil || detail.ID != "detail-order" || detail.PaymentStatus != "paid" || len(detail.Items) != 1 {
		t.Fatalf("owned detail (%s) = %#v, %v", dialect, detail, err)
	}
	if detail.Items[0].Title != "Reader item" {
		t.Fatalf("detail item (%s) = %#v", dialect, detail.Items[0])
	}
	if _, err := reader.GetOrderByUserID(ctx, db, "other-member", "detail-order"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-member detail (%s) = %v, want sql.ErrNoRows", dialect, err)
	}
}
