package sdkdist

import (
	"strings"
	"testing"
	"time"
)

func TestValidateGrantBindsProjectVersionsAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	grant := ArtifactGrant{AccessToken: "token-123456", ProjectID: "proj-1", CoreVersion: "1.2.3", SDKVersion: "1.0.0", ArtifactURL: "https://artifacts.example.test/hydro", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	if err := ValidateGrant(grant, "proj-1", "1.2.3", "1.0.0", now, false); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*ArtifactGrant){
		func(g *ArtifactGrant) { g.ProjectID = "other" },
		func(g *ArtifactGrant) { g.CoreVersion = "other" },
		func(g *ArtifactGrant) { g.ExpiresAt = now.Add(-time.Second) },
		func(g *ArtifactGrant) { g.ExpiresAt = g.IssuedAt.Add(MaxGrantLifetime + time.Hour) },
	} {
		copy := grant
		mutate(&copy)
		if err := ValidateGrant(copy, "proj-1", "1.2.3", "1.0.0", now, false); err == nil {
			t.Fatal("invalid grant was accepted")
		}
	}
}

func TestRedactTokenDoesNotExposeShortTokens(t *testing.T) {
	if got := RedactToken("short"); got != "[redacted]" {
		t.Fatalf("short token redaction = %q", got)
	}
	if got := RedactToken("abcdefgh1234"); strings.Contains(got, "abcdefgh1234") {
		t.Fatal("full token was exposed")
	}
}
