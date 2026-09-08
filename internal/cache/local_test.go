package cache

import (
	"testing"
	"time"
)

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
