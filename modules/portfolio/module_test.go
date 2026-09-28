package portfolio

import (
	"bytes"
	"context"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"testing"
)

func TestModelValidationCloneAndTemplRendering(t *testing.T) {
	item := Project{Base: sectoral.Base{ID: "entry-1", Locale: " EN ", Slug: "sample-entry", Title: "Portfolio entry", Status: "published", Featured: true}, Client: "Örnek Müşteri", Industry: "Üretim", CoverURL: "/media/project.webp", Gallery: []string{"/media/one.webp"}}
	if err := item.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if item.Locale != "en" || !item.Featured {
		t.Fatalf("canonical item = %+v", item.Base)
	}
	var output bytes.Buffer
	if err := Collection([]Project{item}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Üretim") {
		t.Fatalf("rendered collection omitted %q: %s", "Üretim", output.String())
	}
	output.Reset()
	if err := Related(item.ID, []Project{item, {Base: sectoral.Base{ID: "related-1", Locale: "en", Slug: "related", Title: "Related project", Status: "published", Tags: []string{"go"}}, Industry: "Technology"}}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `aria-label="İlgili projeler"`) || !strings.Contains(output.String(), "Related project") || strings.Contains(output.String(), "Portfolio entry") {
		t.Fatalf("related project rendering = %s", output.String())
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
	item := Project{Base: sectoral.Base{ID: "bad", Locale: "en", Slug: "", Title: "Title"}, Client: "Örnek Müşteri", Industry: "Üretim", CoverURL: "/media/project.webp", Gallery: []string{"/media/one.webp"}}
	if err := item.Validate(); err == nil {
		t.Fatal("unsafe slug was accepted")
	}
}

func TestExtensionManifestDeclaresOnlyOwnedMigration(t *testing.T) {
	manifest := (Extension{}).ModuleManifest()
	if manifest.ID != "portfolio" || len(manifest.Migrations) != 2 || manifest.Migrations[1] != "202609280021_portfolio_tag_relations" || manifest.Routes != nil || manifest.Workers != nil {
		t.Fatalf("manifest = %+v", manifest)
	}
}
