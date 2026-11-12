package main

import (
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/modules/commerce"
)

// Never-checked orders go first, then the least recently checked: a pile of old unpaid orders that was
// already looked at cannot hold a fresh (possibly paid) one back.
func TestPendingCheckRoundRotates(t *testing.T) {
	base := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	orders := []commerce.Order{
		{ID: "old-unpaid-1", CreatedAt: base},
		{ID: "old-unpaid-2", CreatedAt: base.Add(time.Second)},
		{ID: "fresh-paid", CreatedAt: base.Add(time.Minute)},
	}
	checked := map[string]time.Time{"old-unpaid-1": base.Add(2 * time.Minute), "old-unpaid-2": base.Add(2 * time.Minute)}
	got := pendingCheckRound(orders, checked)
	if got[0].ID != "fresh-paid" {
		t.Fatalf("a never-checked order must come first, got %v", []string{got[0].ID, got[1].ID, got[2].ID})
	}
	if got[1].ID != "old-unpaid-1" || got[2].ID != "old-unpaid-2" {
		t.Fatalf("ties break by age, got %v", []string{got[0].ID, got[1].ID, got[2].ID})
	}
	if orders[0].ID != "old-unpaid-1" || orders[2].ID != "fresh-paid" {
		t.Fatal("the input slice must not be reordered")
	}
}
