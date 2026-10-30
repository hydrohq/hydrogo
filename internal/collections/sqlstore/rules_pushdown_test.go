package sqlstore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestAccessRuleSQLPushdownPaginationAndIsolation(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "rules_pushdown.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite}}

	// Define collection schema
	colID := "messages"
	schema := collections.Schema{
		ID:    colID,
		Label: "Messages",
		Fields: []collections.Field{
			{ID: "text", Label: "Text", Kind: collections.Text},
			{ID: "sender_id", Label: "Sender", Kind: collections.Text},
			{ID: "status", Label: "Status", Kind: collections.Text},
		},
	}
	if err := service.CreateSchema(ctx, schema); err != nil {
		t.Fatalf("sync schema: %v", err)
	}

	// Insert 20 records for user A, then 20 records for user B
	for i := 1; i <= 20; i++ {
		_, err := service.CreateRecord(ctx, colID, map[string]any{
			"text":      fmt.Sprintf("Alice message %d", i),
			"sender_id": "usr_alice",
			"status":    "active",
		})
		if err != nil {
			t.Fatalf("save alice msg %d: %v", i, err)
		}
	}
	for i := 1; i <= 20; i++ {
		_, err := service.CreateRecord(ctx, colID, map[string]any{
			"text":      fmt.Sprintf("Bob message %d", i),
			"sender_id": "usr_bob",
			"status":    "active",
		})
		if err != nil {
			t.Fatalf("save bob msg %d: %v", i, err)
		}
	}

	// Test 1: User B queries with limit=5 and AccessRule="@auth.id != '' && @auth.id == record.sender_id"
	// Without SQL pushdown, user A's rows would fill the page and be filtered out in Go memory, returning 0 records.
	// With SQL pushdown, Bob must receive exactly 5 records belonging to Bob.
	queryBob := collections.Query{
		Limit:      5,
		Offset:     0,
		AccessRule: "@auth.id != '' && @auth.id == record.sender_id",
		AuthContext: collections.AuthContext{
			ID:       "usr_bob",
			Email:    "bob@example.com",
			Role:     "member",
			IsMember: true,
		},
	}

	recs, err := service.QueryRecords(ctx, colID, queryBob)
	if err != nil {
		t.Fatalf("query bob records: %v", err)
	}
	if len(recs) != 5 {
		t.Fatalf("expected exactly 5 records for user B, got %d", len(recs))
	}
	for _, rec := range recs {
		if rec.Data["sender_id"] != "usr_bob" {
			t.Fatalf("security violation: expected sender usr_bob, got %v", rec.Data["sender_id"])
		}
	}

	// Test 2: Keyset cursor pagination under row-level rules
	queryPage1 := collections.Query{
		Limit:      10,
		AccessRule: "@auth.id == record.sender_id",
		AuthContext: collections.AuthContext{
			ID: "usr_alice",
		},
	}
	page1, err := service.QueryRecords(ctx, colID, queryPage1)
	if err != nil {
		t.Fatalf("query alice page 1: %v", err)
	}
	if len(page1) != 10 {
		t.Fatalf("alice page 1 length = %d, want 10", len(page1))
	}

	// Page 2 using NextCursor
	lastRec := page1[len(page1)-1]
	cursorStr := collections.NextCursor(queryPage1, lastRec)
	if cursorStr == "" {
		t.Fatal("expected next cursor, got empty string")
	}
	parsedCursor, err := collections.DecodeCursor(cursorStr)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	queryPage2 := collections.Query{
		Limit:      10,
		After:      &parsedCursor,
		AccessRule: "@auth.id == record.sender_id",
		AuthContext: collections.AuthContext{
			ID: "usr_alice",
		},
	}
	page2, err := service.QueryRecords(ctx, colID, queryPage2)
	if err != nil {
		t.Fatalf("query alice page 2: %v", err)
	}
	if len(page2) != 10 {
		t.Fatalf("alice page 2 length = %d, want 10", len(page2))
	}
	for _, rec := range page2 {
		if rec.Data["sender_id"] != "usr_alice" {
			t.Fatalf("expected alice record, got %v (rec: %+v, cursor: %+v)", rec.Data["sender_id"], rec, parsedCursor)
		}
	}

	// Test 3: Fail-closed on unsupported dynamic function in access rule
	queryUnsupported := collections.Query{
		Limit:      5,
		AccessRule: "unknown_func(record.sender_id) == true",
		AuthContext: collections.AuthContext{
			ID: "usr_bob",
		},
	}
	_, err = service.QueryRecords(ctx, colID, queryUnsupported)
	if err == nil {
		t.Fatal("expected error on unsupported function in access rule, got nil")
	}
}
