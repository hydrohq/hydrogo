package site

import "testing"

func TestValidateRedirectUsesCanonicalPaths(t *testing.T) {
	value := Redirect{SourcePath: " /Old/%C3%9Cr%C3%BCn/ ", TargetPath: "/new", Status: 301}
	if err := ValidateRedirect(value); err != nil {
		t.Fatalf("ValidateRedirect() error = %v", err)
	}
	if got := normalizePath(value.SourcePath); got != "/old/ürün" {
		t.Fatalf("normalizePath() = %q, want %q", got, "/old/ürün")
	}
}

func TestValidateRedirectRejectsUnsafeEncoding(t *testing.T) {
	value := Redirect{SourcePath: "/old", TargetPath: "/bad\\path", Status: 301}
	if err := ValidateRedirect(value); err == nil {
		t.Fatal("ValidateRedirect() accepted an unsafe path")
	}
}
