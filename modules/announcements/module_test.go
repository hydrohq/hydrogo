package announcements

import (
	"bytes"
	"context"
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
