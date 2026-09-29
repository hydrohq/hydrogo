package downloads

import (
	"bytes"
	"context"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"testing"
)

func TestModelValidationCloneAndTemplRendering(t *testing.T) {
	item := Download{Base: sectoral.Base{ID: "entry-1", Locale: " EN ", Slug: "sample-entry", Title: "Downloads entry", Status: "published", Featured: true}, FileURL: "/files/catalog.pdf", ContentType: "application/pdf", FileSize: 1024, DownloadCount: 3}
	if err := item.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if item.Locale != "en" || !item.Featured {
		t.Fatalf("canonical item = %+v", item.Base)
	}
	var output bytes.Buffer
	if err := Collection([]Download{item}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Dosyayı indir") {
		t.Fatalf("rendered collection omitted %q: %s", "Dosyayı indir", output.String())
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
	item := Download{Base: sectoral.Base{ID: "bad", Locale: "en", Slug: "", Title: "Title"}, FileURL: "/files/catalog.pdf", ContentType: "application/pdf", FileSize: 1024, DownloadCount: 3}
	if err := item.Validate(); err == nil {
		t.Fatal("unsafe slug was accepted")
	}
}

func TestPDFCoverURLOnlyAcceptsLocalMediaPDFs(t *testing.T) {
	cases := []struct {
		fileURL, contentType, want string
	}{
		{"/media/asset_123", "application/pdf", "/media/asset_123?variant=pdf-cover"},
		{"https://elsewhere.test/media/asset", "application/pdf", ""},
		{"/media/asset?download=1", "application/pdf", ""},
		{"/media/asset/extra", "application/pdf", ""},
		{"/media/asset", "application/msword", ""},
	}
	for _, tc := range cases {
		if got := PDFCoverURL(tc.fileURL, tc.contentType); got != tc.want {
			t.Errorf("PDFCoverURL(%q, %q) = %q, want %q", tc.fileURL, tc.contentType, got, tc.want)
		}
	}
}

func TestExtensionManifestDeclaresOnlyOwnedMigration(t *testing.T) {
	manifest := (Extension{}).ModuleManifest()
	if manifest.ID != "downloads" || len(manifest.Migrations) != 1 || manifest.Routes != nil || manifest.Workers != nil {
		t.Fatalf("manifest = %+v", manifest)
	}
}
