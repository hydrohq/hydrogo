package seo

import (
	"context"
	"strings"
	"testing"
)

func TestRenderLLMSSeparatesIndexAndPlainFullText(t *testing.T) {
	index, full, err := RenderLLMS(Discovery{
		Name: "Example <Company>", Description: "Public site & services", Origin: "https://example.test",
		ExcludePath: []string{"/private"},
		Pages: []DiscoveryPage{
			{Title: "Home", Path: "/en/home", Description: "Welcome"},
			{Title: "Private", Path: "/private/profile", Text: "secret"},
			{Title: "Services", Path: "/en/services", Text: `<p>Trusted &amp; safe</p><script>doNotInclude()</script>`},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "https://example.test/en/home") || strings.Contains(string(index), "private") || strings.Contains(string(index), "Trusted") {
		t.Fatalf("index contains unexpected data: %s", index)
	}
	if !strings.Contains(string(full), "Trusted & safe") || strings.Contains(string(full), "doNotInclude") || strings.Contains(string(full), "secret") {
		t.Fatalf("full text was not safely projected: %s", full)
	}
}

func TestLLMSPathsCannotBreakMarkdownLinks(t *testing.T) {
	if got := cleanPath("/page](https://evil.example)"); strings.Contains(got, "](") {
		t.Fatalf("path can escape markdown link: %q", got)
	}
	if got := cleanPath("/page?next=/private"); got != "" {
		t.Fatalf("query-bearing page path accepted: %q", got)
	}
}

func TestDiscoveryCacheInvalidatesPublishedContent(t *testing.T) {
	cache := &DiscoveryCache{}
	title := "Before"
	provider := func(context.Context) (Discovery, error) {
		return Discovery{Name: "Site", Pages: []DiscoveryPage{{Title: title, Path: "/page"}}}, nil
	}
	first, err := cache.Get(context.Background(), provider, false)
	if err != nil || !strings.Contains(string(first), "Before") {
		t.Fatalf("first result=%q err=%v", first, err)
	}
	title = "After"
	cached, err := cache.Get(context.Background(), provider, false)
	if err != nil || !strings.Contains(string(cached), "Before") {
		t.Fatalf("expected cached result, got %q err=%v", cached, err)
	}
	cache.Invalidate()
	updated, err := cache.Get(context.Background(), provider, false)
	if err != nil || !strings.Contains(string(updated), "After") {
		t.Fatalf("updated result=%q err=%v", updated, err)
	}
}

func TestDiscoveryCacheDoesNotRepublishResultAfterConcurrentInvalidation(t *testing.T) {
	cache := &DiscoveryCache{}
	started, release := make(chan struct{}), make(chan struct{})
	oldProvider := func(context.Context) (Discovery, error) {
		close(started)
		<-release
		return Discovery{Name: "Site", Pages: []DiscoveryPage{{Title: "Old", Path: "/page"}}}, nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cache.Get(context.Background(), oldProvider, false)
	}()
	<-started
	cache.Invalidate()
	close(release)
	<-done
	updated, err := cache.Get(context.Background(), func(context.Context) (Discovery, error) {
		return Discovery{Name: "Site", Pages: []DiscoveryPage{{Title: "New", Path: "/page"}}}, nil
	}, false)
	if err != nil || !strings.Contains(string(updated), "New") || strings.Contains(string(updated), "Old") {
		t.Fatalf("stale generation was cached: %q, err=%v", updated, err)
	}
}
