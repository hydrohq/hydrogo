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
