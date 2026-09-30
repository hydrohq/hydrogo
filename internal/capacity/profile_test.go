package capacity

import "testing"

func TestProfileDefaultsAreBounded(t *testing.T) {
	for _, profile := range []Profile{Compact, Standard, Distributed} {
		budget, err := Defaults(profile)
		if err != nil {
			t.Fatal(err)
		}
		if err := budget.Validate(); err != nil {
			t.Fatalf("%s budget: %v", profile, err)
		}
	}
	if _, err := Defaults(Profile("unknown")); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestPublicHTMLCacheBudgetValidation(t *testing.T) {
	base, err := Defaults(Standard)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*Budget)
	}{
		{"zero entries", func(b *Budget) { b.PublicHTMLCacheEntries = 0 }},
		{"zero max bytes", func(b *Budget) { b.PublicHTMLCacheMaxBytes = 0 }},
		{"zero max body bytes", func(b *Budget) { b.PublicHTMLCacheMaxBodyBytes = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			budget := base
			tc.mutate(&budget)
			if err := budget.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestPublicHTMLCacheBudgetScalesWithProfile(t *testing.T) {
	compact, err := Defaults(Compact)
	if err != nil {
		t.Fatal(err)
	}
	standard, err := Defaults(Standard)
	if err != nil {
		t.Fatal(err)
	}
	distributed, err := Defaults(Distributed)
	if err != nil {
		t.Fatal(err)
	}
	if !(compact.PublicHTMLCacheEntries < standard.PublicHTMLCacheEntries && standard.PublicHTMLCacheEntries < distributed.PublicHTMLCacheEntries) {
		t.Fatalf("expected increasing PublicHTMLCacheEntries by profile: compact=%d standard=%d distributed=%d",
			compact.PublicHTMLCacheEntries, standard.PublicHTMLCacheEntries, distributed.PublicHTMLCacheEntries)
	}
	if !(compact.PublicHTMLCacheMaxBytes < standard.PublicHTMLCacheMaxBytes && standard.PublicHTMLCacheMaxBytes < distributed.PublicHTMLCacheMaxBytes) {
		t.Fatalf("expected increasing PublicHTMLCacheMaxBytes by profile: compact=%d standard=%d distributed=%d",
			compact.PublicHTMLCacheMaxBytes, standard.PublicHTMLCacheMaxBytes, distributed.PublicHTMLCacheMaxBytes)
	}
}
