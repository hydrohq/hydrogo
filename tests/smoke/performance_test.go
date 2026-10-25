//go:build smoke

package smoke

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const perfType = `id: perf_items
label: Perf items
title_field: title
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, searchable: true}
  - {id: bucket, label: Bucket, kind: select, options: [a, b, c, d, e, f, g, h], filterable: true, list: true}
  - {id: amount, label: Amount, kind: number, filterable: true, list: true}
`

// Budgets are deliberately loose: they are meant to catch a missing index or
// an accidental full-table scan (orders of magnitude), not to benchmark. A
// slow shared CI machine should still pass.
const (
	records      = 10000
	listBudget   = 400 * time.Millisecond // p95, one page of 100
	searchBudget = 600 * time.Millisecond
	aggBudget    = 900 * time.Millisecond
	getBudget    = 150 * time.Millisecond
	writeBudget  = 300 * time.Millisecond
	importBudget = 8 * time.Second // per 1000 rows
	loadBudget   = 2500 * time.Millisecond
)

func TestPerformanceWithRealisticData(t *testing.T) {
	p := start(t, binary, map[string]string{"perf_items": perfType})
	token, _ := p.serviceAccount("perf", "admin.content.read", "admin.content.write")

	t.Run("bulk import", func(t *testing.T) {
		for batch := 0; batch < records/1000; batch++ {
			var b strings.Builder
			b.WriteString("title,bucket,amount\n")
			for i := 0; i < 1000; i++ {
				n := batch*1000 + i
				fmt.Fprintf(&b, "Item %05d %s,%c,%d\n", n, []string{"alpha", "bravo", "charlie", "delta"}[n%4], 'a'+rune(n%8), n%977)
			}
			start := time.Now()
			res := p.must("POST", "/api/v1/admin/content/perf_items/_import", &opts{token: token, ctype: "text/csv", body: b.String()})
			if res.status >= 300 {
				t.Fatalf("import batch %d: %d %s", batch, res.status, res.body)
			}
			if d := time.Since(start); d > importBudget {
				t.Errorf("importing 1000 rows took %s (budget %s)", d, importBudget)
			}
		}
	})

	measure := func(name string, budget time.Duration, n int, path string) {
		t.Run(name, func(t *testing.T) {
			var all []time.Duration
			for i := 0; i < n; i++ {
				started := time.Now()
				res := p.must("GET", path, &opts{token: token})
				all = append(all, time.Since(started))
				if res.status != 200 {
					t.Fatalf("GET %s: %d %s", path, res.status, res.body)
				}
			}
			p95 := percentile(all, 0.95)
			t.Logf("%s: p50 %s, p95 %s over %d requests", name, percentile(all, 0.5), p95, n)
			if p95 > budget {
				t.Errorf("p95 %s is over the %s budget", p95, budget)
			}
		})
	}
	base := "/api/v1/admin/content/perf_items"
	measure("list one page", listBudget, 60, base+"?limit=100&sort=-created_at")
	measure("list one page, sorted by a field", listBudget, 40, base+"?limit=100&sort=-amount")
	measure("filter by a select", listBudget, 60, base+"?limit=100&filter[bucket]=c")
	measure("filter by a number range", listBudget, 40, base+"?limit=100&filter[amount][gte]=900&filter[amount][lt]=950")
	measure("text search", searchBudget, 40, base+"?limit=50&search=charlie")
	measure("aggregate by group", aggBudget, 20, base+"/_aggregate?group_by=bucket&sum=amount")

	t.Run("read one record", func(t *testing.T) {
		first := p.must("GET", base+"?limit=1", &opts{token: token}).list()
		id := first[0].(map[string]any)["id"].(string)
		var all []time.Duration
		for i := 0; i < 80; i++ {
			started := time.Now()
			res := p.must("GET", base+"/"+id, &opts{token: token})
			all = append(all, time.Since(started))
			if res.status != 200 {
				t.Fatalf("%d", res.status)
			}
		}
		if p95 := percentile(all, 0.95); p95 > getBudget {
			t.Errorf("p95 %s over %s", p95, getBudget)
		}
	})

	t.Run("create records", func(t *testing.T) {
		var all []time.Duration
		for i := 0; i < 100; i++ {
			started := time.Now()
			res := p.must("POST", base, &opts{token: token, body: map[string]any{"title": fmt.Sprintf("New %d", i), "bucket": "a", "amount": i}})
			all = append(all, time.Since(started))
			if res.status != 201 {
				t.Fatalf("create: %d %s", res.status, res.body)
			}
		}
		t.Logf("create: p50 %s, p95 %s", percentile(all, 0.5), percentile(all, 0.95))
		if p95 := percentile(all, 0.95); p95 > writeBudget {
			t.Errorf("p95 %s over %s", p95, writeBudget)
		}
	})

	t.Run("eight clients at once: no errors, no throttling of normal use", func(t *testing.T) {
		const workers, each = 8, 25
		var mu sync.Mutex
		var all []time.Duration
		var bad []string
		var wg sync.WaitGroup
		paths := []string{base + "?limit=100", base + "?limit=100&filter[bucket]=b", base + "?limit=50&search=delta", base + "/_aggregate?group_by=bucket&sum=amount"}
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; i < each; i++ {
					started := time.Now()
					res, err := p.do("GET", paths[(w+i)%len(paths)], &opts{token: token})
					d := time.Since(started)
					mu.Lock()
					all = append(all, d)
					if err != nil || res.status != 200 {
						bad = append(bad, fmt.Sprintf("%v/%d", err, res.status))
					}
					mu.Unlock()
				}
			}(w)
		}
		wg.Wait()
		t.Logf("concurrent: p50 %s, p95 %s over %d requests", percentile(all, 0.5), percentile(all, 0.95), len(all))
		if len(bad) > 0 {
			t.Errorf("%d of %d requests failed: %v", len(bad), len(all), bad[:min(len(bad), 5)])
		}
		if p95 := percentile(all, 0.95); p95 > loadBudget {
			t.Errorf("p95 %s over %s", p95, loadBudget)
		}
	})
}
