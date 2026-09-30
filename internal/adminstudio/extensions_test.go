package adminstudio

import (
	"errors"
	"testing"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestExtensionRegistryValidatesAndSortsDeclarativeManifests(t *testing.T) {
	base := extensionv1.AdminExtensionManifest{ID: "reports", Name: "Reports", Version: "1.0.0", EntrypointURL: "https://reports.example/admin", Surfaces: []extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "/", Placement: "reports"}}}
	other := base
	other.ID, other.Name = "billing", "Billing"
	other.Surfaces = []extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "/billing", Placement: "billing"}}
	registry, err := NewExtensionRegistry([]extensionv1.AdminExtensionManifest{base, other})
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.Manifests(); len(got) != 2 || got[0].ID != "billing" || got[1].ID != "reports" {
		t.Fatalf("registry order = %#v", got)
	}
}

func TestExtensionRegistryRejectsDuplicateOrInvalidManifest(t *testing.T) {
	manifest := extensionv1.AdminExtensionManifest{ID: "reports", Name: "Reports", Version: "1", EntrypointURL: "https://reports.example", Surfaces: []extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "/", Placement: "reports"}}}
	if _, err := NewExtensionRegistry([]extensionv1.AdminExtensionManifest{manifest, manifest}); !errors.Is(err, ErrInvalidAdminExtensionRegistry) {
		t.Fatalf("duplicate error = %v", err)
	}
	manifest.EntrypointURL = "http://reports.example"
	if _, err := NewExtensionRegistry([]extensionv1.AdminExtensionManifest{manifest}); !errors.Is(err, ErrInvalidAdminExtensionRegistry) {
		t.Fatalf("invalid error = %v", err)
	}
}
