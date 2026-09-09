package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRemote struct {
	value  string
	getErr error
	setErr error
}

func (r *fakeRemote) Get(string) (string, bool, error) { return r.value, r.value != "", r.getErr }
func (r *fakeRemote) Set(string, string) error         { return r.setErr }

func TestTwoTierUsesL2AndWarmsL1(t *testing.T) {
	l2 := &fakeRemote{value: "remote"}
	cache, err := NewTwoTier(New[string](2, time.Minute), l2)
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	value, err := cache.GetOrLoad("key", func() (string, error) { loads++; return "loader", nil })
	if err != nil || value != "remote" || loads != 0 {
		t.Fatalf("L2 result = %q/%v, loads = %d", value, err, loads)
	}
	value, _ = cache.GetOrLoad("key", func() (string, error) { loads++; return "loader", nil })
	if value != "remote" || loads != 0 {
		t.Fatalf("L1 result = %q, loads = %d", value, loads)
	}
}

func TestTwoTierFallsBackWhenL2Fails(t *testing.T) {
	l2 := &fakeRemote{getErr: errors.New("L2 unavailable"), setErr: errors.New("L2 unavailable")}
	cache, _ := NewTwoTier(New[string](2, time.Minute), l2)
	value, err := cache.GetOrLoad("key", func() (string, error) { return "local", nil })
	if err != nil || value != "local" {
		t.Fatalf("fallback = %q/%v", value, err)
	}
}

func TestNewTwoTierRequiresLocal(t *testing.T) {
	if _, err := NewTwoTier[string](nil, nil); err == nil {
		t.Fatal("expected error when local cache is nil")
	}
}

func TestTwoTierWithNilRemote(t *testing.T) {
	cache, err := NewTwoTier(New[string](2, time.Minute), nil)
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	val, err := cache.GetOrLoad("key", func() (string, error) {
		loads++
		return "val", nil
	})
	if err != nil || val != "val" || loads != 1 {
		t.Fatalf("unexpected value %q, loads %d, err %v", val, loads, err)
	}
	val, err = cache.GetOrLoad("key", func() (string, error) {
		loads++
		return "val2", nil
	})
	if err != nil || val != "val" || loads != 1 {
		t.Fatalf("unexpected cached value %q, loads %d, err %v", val, loads, err)
	}
}

func TestTwoTierCoalescesConcurrentMissesWhenL2Misses(t *testing.T) {
	l2 := &fakeRemote{}
	cache, err := NewTwoTier(New[string](2, time.Minute), l2)
	if err != nil {
		t.Fatal(err)
	}
	const callers = 32
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	var wg sync.WaitGroup
	values := make(chan string, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := cache.GetOrLoad("shared", func() (string, error) {
				if loads.Add(1) == 1 {
					close(started)
					<-release
				}
				return "loaded", nil
			})
			if err != nil {
				t.Errorf("concurrent load: %v", err)
				return
			}
			values <- value
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(values)
	if got := loads.Load(); got != 1 {
		t.Fatalf("loader calls = %d, want 1", got)
	}
	for value := range values {
		if value != "loaded" {
			t.Fatalf("loaded value = %q", value)
		}
	}
}
