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

func TestParseGrantIsBoundedAndRejectsTrailingData(t *testing.T) {
	data := `{"access_token":"token-123456","project_id":"proj-1","core_version":"1.2.3","sdk_version":"1.0.0","artifact_url":"https://artifacts.example.test/hydro","issued_at":"2026-09-16T11:59:00Z","expires_at":"2026-09-16T13:00:00Z"}`
	grant, err := ParseGrant([]byte(data))
	if err != nil || grant.ProjectID != "proj-1" {
		t.Fatalf("parsed grant = %#v/%v", grant, err)
	}
	if _, err := ParseGrant(append([]byte(data), []byte("{}")...)); err == nil {
		t.Fatal("trailing grant data was accepted")
	}
	if _, err := ParseGrant(make([]byte, MaxGrantResponseBytes+1)); err == nil {
		t.Fatal("oversized grant response was accepted")
	}
}
