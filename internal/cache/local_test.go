package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheCoalescesConcurrentLoads(t *testing.T) {
	c := New[string](2, time.Minute)
	var mu sync.Mutex
	loads := 0
	loader := func() (string, error) {
		mu.Lock()
		loads++
		mu.Unlock()
		return "value", nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := c.GetOrLoad("key", loader)
			if err != nil || value != "value" {
				t.Errorf("load = %q/%v", value, err)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if loads != 1 {
		t.Fatalf("loader calls = %d, want 1", loads)
	}
}

func BenchmarkCacheCoalesces64ConcurrentLookups(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := New[string](64, time.Minute)
		var loads atomic.Int32
		var wg sync.WaitGroup
		wg.Add(64)
		for j := 0; j < 64; j++ {
			go func() {
				defer wg.Done()
				_, err := c.GetOrLoad("public:home", func() (string, error) {
					loads.Add(1)
					return "projection", nil
				})
				if err != nil {
					b.Errorf("cache load: %v", err)
				}
			}()
		}
		wg.Wait()
		if got := loads.Load(); got != 1 {
			b.Fatalf("loader calls = %d, want 1", got)
		}
	}
}

func TestCacheRejectsNilLoader(t *testing.T) {
	c := New[string](1, time.Minute)
	if _, err := c.GetOrLoad("key", nil); !errors.Is(err, errors.New("cache loader is required")) {
		if err == nil || err.Error() != "cache loader is required" {
			t.Fatalf("nil loader error = %v", err)
		}
	}
}

func TestCacheIsBoundedAndExpires(t *testing.T) {
	c := New[string](1, 10*time.Millisecond)
	c.Set("one", "1")
	c.Set("two", "2")
	if _, ok := c.Get("one"); ok {
		t.Fatal("cache exceeded configured bound")
	}
	if value, ok := c.Get("two"); !ok || value != "2" {
		t.Fatalf("cached value = %q/%v", value, ok)
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := c.Get("two"); ok {
		t.Fatal("expired value was returned")
	}
}

func TestCacheRejectsRefillFromPreviousGeneration(t *testing.T) {
	c := New[string](2, time.Minute)
	generation := c.Generation()
	c.Clear()
	if c.SetIfGeneration("stale", "old", generation) {
		t.Fatal("stale generation refill was accepted")
	}
	if _, ok := c.Get("stale"); ok {
		t.Fatal("stale value entered cache")
	}
	if !c.SetIfGeneration("fresh", "new", c.Generation()) {
		t.Fatal("current generation refill was rejected")
	}
}

func TestGetOrLoadDoesNotRefillAfterClear(t *testing.T) {
	c := New[string](2, time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	result := make(chan string, 1)
	go func() {
		value, err := c.GetOrLoad("slow", func() (string, error) {
			close(started)
			<-release
			return "stale", nil
		})
		if err != nil {
			t.Errorf("slow load error: %v", err)
		}
		result <- value
	}()
	<-started
	c.Clear()
	close(release)
	if got := <-result; got != "stale" {
		t.Fatalf("loader result = %q", got)
	}
	loads := 0
	value, err := c.GetOrLoad("slow", func() (string, error) {
		loads++
		return "fresh", nil
	})
	if err != nil || value != "fresh" || loads != 1 {
		t.Fatalf("fresh load = %q/%v, calls=%d", value, err, loads)
	}
}

func TestGetOrLoadRevalidatesAfterClearWhileOldLoadIsInFlight(t *testing.T) {
	c := New[string](2, time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	oldResult := make(chan string, 1)
	go func() {
		value, err := c.GetOrLoad("route", func() (string, error) {
			close(started)
			<-release
			return "old", nil
		})
		if err != nil {
			t.Errorf("old load error: %v", err)
		}
		oldResult <- value
	}()
	<-started
	c.Clear()
	freshResult := make(chan string, 1)
	go func() {
		value, err := c.GetOrLoad("route", func() (string, error) { return "fresh", nil })
		if err != nil {
			t.Errorf("fresh load error: %v", err)
		}
		freshResult <- value
	}()
	if got := <-freshResult; got != "fresh" {
		t.Fatalf("fresh result = %q", got)
	}
	close(release)
	if got := <-oldResult; got != "old" {
		t.Fatalf("old result = %q", got)
	}
	if got, ok := c.Get("route"); !ok || got != "fresh" {
		t.Fatalf("cached revalidated value = %q/%v", got, ok)
	}
}

func TestGetOrLoadAtGenerationRejectsLateOldSlotRefill(t *testing.T) {
	c := New[string](2, time.Minute)
	oldGeneration := c.Generation()
	started := make(chan struct{})
	release := make(chan struct{})
	oldResult := make(chan string, 1)
	go func() {
		value, err := c.GetOrLoadAtGeneration("route", oldGeneration, func() (string, error) {
			close(started)
			<-release
			return "old-slot", nil
		})
		if err != nil {
			t.Errorf("old generation load error: %v", err)
		}
		oldResult <- value
	}()
	<-started
	c.Clear()

	fresh, err := c.GetOrLoadAtGeneration("route", c.Generation(), func() (string, error) {
		return "active-slot", nil
	})
	if err != nil || fresh != "active-slot" {
		t.Fatalf("fresh generation load = %q/%v", fresh, err)
	}
	close(release)
	if got := <-oldResult; got != "old-slot" {
		t.Fatalf("old generation result = %q", got)
	}
	if got, ok := c.Get("route"); !ok || got != "active-slot" {
		t.Fatalf("old generation refilled cache: %q/%v", got, ok)
	}
}

func TestCacheStatsAreBoundedAndAccurate(t *testing.T) {
	c := New[string](1, time.Millisecond)
	if _, err := c.GetOrLoad("key", func() (string, error) { return "value", nil }); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("key"); !ok {
		t.Fatal("loaded value was not cached")
	}
	time.Sleep(2 * time.Millisecond)
	if _, ok := c.Get("key"); ok {
		t.Fatal("expired value was returned")
	}
	c.Set("one", "1")
	c.Set("two", "2")
	stats := c.Stats()
	if stats.Loads != 1 || stats.Hits != 1 || stats.Expirations < 1 || stats.Evictions != 1 {
		t.Fatalf("unexpected cache stats: %+v", stats)
	}
}

func TestCacheFailsClosedWhenNilOrZeroValue(t *testing.T) {
	var nilCache *Cache[string]
	if _, err := nilCache.GetOrLoad("key", func() (string, error) { return "value", nil }); !errors.Is(err, ErrCacheUnavailable) {
		t.Fatalf("nil cache load = %v, want %v", err, ErrCacheUnavailable)
	}
	if _, ok := nilCache.Get("key"); ok {
		t.Fatal("nil cache returned a value")
	}
	zeroCache := Cache[string]{}
	if _, err := zeroCache.GetOrLoad("key", func() (string, error) { return "value", nil }); !errors.Is(err, ErrCacheUnavailable) {
		t.Fatalf("zero-value cache load = %v, want %v", err, ErrCacheUnavailable)
	}
	zeroCache.Set("key", "value")
	if zeroCache.Stats() != (Stats{}) {
		t.Fatal("zero-value cache reported non-zero stats")
	}
}

func TestCacheLoaderPanicDoesNotBlockCoalescedWaiters(t *testing.T) {
	c := New[string](2, time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := c.GetOrLoad("panic", func() (string, error) {
			close(started)
			<-release
			panic("private loader detail")
		})
		first <- err
	}()
	<-started
	second := make(chan error, 1)
	go func() {
		_, err := c.GetOrLoad("panic", func() (string, error) { panic("private waiter detail") })
		second <- err
	}()
	close(release)
	select {
	case err := <-first:
		if !errors.Is(err, ErrCacheLoaderPanic) {
			t.Fatalf("first loader error = %v, want %v", err, ErrCacheLoaderPanic)
		}
	case <-time.After(time.Second):
		t.Fatal("panicking loader did not complete")
	}
	select {
	case err := <-second:
		if !errors.Is(err, ErrCacheLoaderPanic) {
			t.Fatalf("coalesced waiter error = %v, want %v", err, ErrCacheLoaderPanic)
		}
	case <-time.After(time.Second):
		t.Fatal("coalesced waiter remained blocked")
	}
}
