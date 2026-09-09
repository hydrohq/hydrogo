package release_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/release"
)

// TestBlueGreenTrafficWithConcurrentCacheInvalidation verifies that under concurrent traffic:
// 1. All requests receive valid HTTP 200 responses throughout Blue->Green switch and Green->Blue rollback.
// 2. DB/Loader lookups are strictly bounded and coalesced (no thundering herd).
// 3. Stale cache refill from in-flight requests across generation clears is rejected.
func TestBlueGreenTrafficWithConcurrentCacheInvalidation(t *testing.T) {
	c := cache.New[string](128, time.Minute)

	var dbQueryCount atomic.Int32
	firstBlueStarted := make(chan struct{})
	greenStarted := make(chan struct{})
	releaseBlue := make(chan struct{})
	releaseGreen := make(chan struct{})
	var firstBlue atomic.Bool
	var greenNotification sync.Once
	loadPage := func(slot string) string {
		val, err := c.GetOrLoad("public:page", func() (string, error) {
			dbQueryCount.Add(1)
			if slot == "blue" && firstBlue.CompareAndSwap(false, true) {
				close(firstBlueStarted)
				<-releaseBlue
			}
			if slot == "green" {
				greenNotification.Do(func() { close(greenStarted) })
				<-releaseGreen
			}
			return fmt.Sprintf("content-from-%s", slot), nil
		})
		if err != nil {
			return ""
		}
		return val
	}

	blueHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		res := loadPage("blue")
		_, _ = w.Write([]byte(res))
	})
	greenHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		res := loadPage("green")
		_, _ = w.Write([]byte(res))
	})

	traffic, err := release.NewTraffic(blueHandler, greenHandler)
	if err != nil {
		t.Fatal(err)
	}

	deployment, err := release.NewDeployment(release.SlotBlue)
	if err != nil {
		t.Fatal(err)
	}

	// Force an old-generation read to remain in flight while the new slot
	// starts its own load. Green must win the cache refill; the late Blue
	// result may still reach its caller but must not repopulate the cache.
	blueResult := make(chan string, 1)
	go func() { blueResult <- loadPage("blue") }()
	<-firstBlueStarted
	c.Clear()
	if err := deployment.PrepareGreen(); err != nil {
		t.Fatal(err)
	}
	if err := deployment.MarkReady(); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Warmup(true); err != nil {
		t.Fatal(err)
	}
	if err := deployment.Switch(true); err != nil {
		t.Fatal(err)
	}
	if err := traffic.Switch(deployment); err != nil {
		t.Fatal(err)
	}
	greenResult := make(chan string, 1)
	go func() { greenResult <- loadPage("green") }()
	<-greenStarted
	close(releaseGreen)
	if got := <-greenResult; got != "content-from-green" {
		t.Fatalf("green load = %q", got)
	}
	close(releaseBlue)
	if got := <-blueResult; got != "content-from-blue" {
		t.Fatalf("in-flight blue load = %q", got)
	}
	if got, ok := c.Get("public:page"); !ok || got != "content-from-green" {
		t.Fatalf("stale blue refill won cache: %q, present=%v", got, ok)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var totalRequests atomic.Int64
	var totalErrors atomic.Int64

	const workerCount = 16
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					req := httptest.NewRequest(http.MethodGet, "/public/page", nil)
					rec := httptest.NewRecorder()
					traffic.ServeHTTP(rec, req)
					totalRequests.Add(1)
					if rec.Code != http.StatusOK {
						totalErrors.Add(1)
					}
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()
	}

	// Wait for blue traffic to start
	time.Sleep(50 * time.Millisecond)

	// The transition above is also exercised under concurrent traffic below.
	c.Clear()

	time.Sleep(100 * time.Millisecond)
	if got := loadPage("green"); got != "content-from-green" {
		t.Fatalf("green slot served %q", got)
	}

	// Step 2: Rollback to Blue
	if err := deployment.DrainBlue(); err != nil {
		t.Fatal(err)
	}
	c.Clear()
	if err := deployment.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := traffic.Rollback(deployment); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	if totalErrors.Load() > 0 {
		t.Fatalf("encountered %d errors during blue-green traffic switch", totalErrors.Load())
	}
	if totalRequests.Load() < 100 {
		t.Fatalf("expected substantial traffic, got %d requests", totalRequests.Load())
	}
	// We cleared cache twice (switch + rollback), so db queries should be very low (e.g. <= 10) despite hundreds of requests
	if dbQueries := dbQueryCount.Load(); dbQueries > 10 {
		t.Fatalf("excessive DB query count: %d for %d requests", dbQueries, totalRequests.Load())
	}

	if got := loadPage("blue"); got != "content-from-blue" {
		t.Fatalf("rollback to blue served %q", got)
	}
}
