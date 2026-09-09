package news

import "testing"

func TestCanonicalizeNormalizesLocaleAndSlug(t *testing.T) {
	item := Item{Locale: "TR-tr", Slug: "/cafe\u0301/"}
	if err := Canonicalize(&item); err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}
	if item.Locale != "tr-tr" || item.Slug != "café" {
		t.Fatalf("canonical item = %#v", item)
	}
}

func TestCanonicalizeRejectsUnsafeSlug(t *testing.T) {
	item := Item{Locale: "en", Slug: "bad\\path"}
	if err := Canonicalize(&item); err == nil {
		t.Fatal("Canonicalize() accepted a backslash path")
	}
}
