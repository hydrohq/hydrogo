package cache

import (
	"errors"
	"sync"
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
