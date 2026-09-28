package announcements

import (
	"bytes"
	"context"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"testing"
	"time"
)

func TestModelValidationCloneAndTemplRendering(t *testing.T) {
	item := Announcement{Base: sectoral.Base{ID: "entry-1", Locale: " EN ", Slug: "sample-entry", Title: "Announcements entry", Body: "Duyuru", Status: "published", Featured: true}, StartsAt: time.Now().Add(-time.Hour), EndsAt: time.Now().Add(time.Hour), Dismissible: true}
	if err := item.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if item.Locale != "en" || !item.Featured {
		t.Fatalf("canonical item = %+v", item.Base)
	}
	var output bytes.Buffer
	if err := Collection([]Announcement{item}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Duyuru") {
		t.Fatalf("rendered collection omitted %q: %s", "Duyuru", output.String())
	}
	copy := item.Clone("copy-1", "sample-entry-copy")
	if copy.ID != "copy-1" || copy.Status != "draft" || copy.Featured || copy.Slug != "sample-entry-copy" {
		t.Fatalf("clone = %+v", copy.Base)
	}
	if err := copy.Validate(); err != nil {
		t.Fatalf("cloned draft invalid: %v", err)
	}
}

func TestExitIntentLeadCaptureKeepsProtectedPublicFormContract(t *testing.T) {
	props := ExitIntentProps{
		Form:   forms.Form{ID: "exit-lead", Title: "Contact us", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}},
		Action: "/forms/submit", CSRFToken: "csrf-token", SubmissionToken: "submission-token",
		Headline: "Wait a moment", Description: "Leave your email", SubmitLabel: "Send",
	}
	var output bytes.Buffer
	if err := ExitIntentLeadCapture(props).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	rendered := output.String()
	for _, want := range []string{"role=\"dialog\"", "aria-modal=\"true\"", "data-exit-intent-cooldown=\"604800000\"", "name=\"_csrf\" value=\"csrf-token\"", "name=\"_submission_token\" value=\"submission-token\"", "name=\"website_url\"", "data-public-form", "Wait a moment"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered exit-intent prompt omitted %q: %s", want, rendered)
		}
	}
}

func TestExitIntentLeadCaptureOmitsInvalidFormAndHeadIsOptIn(t *testing.T) {
	var output bytes.Buffer
	if err := ExitIntentLeadCapture(ExitIntentProps{}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid form rendered prompt: %s", output.String())
	}
	output.Reset()
	if err := ExitIntentHead().Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, asset := range []string{"/forms/static/forms.css", "/forms/static/forms.js"} {
		if !strings.Contains(output.String(), asset) {
			t.Errorf("head omitted opt-in asset %q: %s", asset, output.String())
		}
	}
}

func TestModelRejectsInvalidIdentity(t *testing.T) {
	item := Announcement{Base: sectoral.Base{ID: "bad", Locale: "en", Slug: "", Title: "Title", Body: "Duyuru"}, StartsAt: time.Now().Add(-time.Hour), EndsAt: time.Now().Add(time.Hour), Dismissible: true}
	if err := item.Validate(); err == nil {
		t.Fatal("unsafe slug was accepted")
	}
}

func TestExtensionManifestDeclaresOnlyOwnedMigration(t *testing.T) {
	manifest := (Extension{}).ModuleManifest()
	if manifest.ID != "announcements" || len(manifest.Migrations) != 1 || manifest.Routes != nil || manifest.Workers != nil {
		t.Fatalf("manifest = %+v", manifest)
	}
}
