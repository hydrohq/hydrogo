package v1

import (
	"errors"
	"testing"
)

func validAdminManifest() AdminExtensionManifest {
	return AdminExtensionManifest{
		ID: "acme-reports", Name: "Acme Reports", Version: "1.0.0", EntrypointURL: "https://reports.acme.example/admin",
		Surfaces:      []AdminSurface{{ID: "sales", Label: "Sales report", Path: "/sales", Placement: "reports", Permissions: []string{"reports.read"}}},
		Subscriptions: []AdminSubscription{{Event: "order.created"}},
	}
}

func TestAdminExtensionManifestValidatesAndResolvesSameOriginSurface(t *testing.T) {
	manifest := validAdminManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, ok := manifest.SurfaceURL("sales"); !ok || got != "https://reports.acme.example/admin/sales" {
		t.Fatalf("surface URL = %q/%v", got, ok)
	}
}

func TestAdminExtensionManifestRejectsUnsafeExternalOrDuplicateSurface(t *testing.T) {
	cases := []AdminExtensionManifest{
		func() AdminExtensionManifest {
			m := validAdminManifest()
			m.EntrypointURL = "http://reports.acme.example"
			return m
		}(),
		func() AdminExtensionManifest {
			m := validAdminManifest()
			m.Surfaces[0].Path = "//evil.example"
			return m
		}(),
		func() AdminExtensionManifest {
			m := validAdminManifest()
			m.Surfaces = append(m.Surfaces, m.Surfaces[0])
			return m
		}(),
		func() AdminExtensionManifest {
			m := validAdminManifest()
			m.Subscriptions = append(m.Subscriptions, m.Subscriptions[0])
			return m
		}(),
	}
	for i, manifest := range cases {
		if err := manifest.Validate(); !errors.Is(err, ErrInvalidAdminExtension) {
			t.Fatalf("case %d error = %v, want ErrInvalidAdminExtension", i, err)
		}
	}
}

func TestSortAdminExtensionsIsDeterministic(t *testing.T) {
	manifests := []AdminExtensionManifest{validAdminManifest(), validAdminManifest()}
	manifests[0].ID, manifests[1].ID = "zeta", "alpha"
	sorted := SortAdminExtensions(manifests)
	if sorted[0].ID != "alpha" || sorted[1].ID != "zeta" {
		t.Fatalf("sorted IDs = %q, %q", sorted[0].ID, sorted[1].ID)
	}
}

func TestParseAdminExtensionManifestValidatesBeforeRegistration(t *testing.T) {
	manifest, err := ParseAdminExtensionManifest([]byte(`{"id":"acme-reports","name":"Acme Reports","version":"1.0.0","entrypoint_url":"https://reports.acme.example/admin","surfaces":[{"id":"sales","label":"Sales","path":"/sales","placement":"reports"}]}`))
	if err != nil || manifest.ID != "acme-reports" {
		t.Fatalf("parsed manifest = %#v, err = %v", manifest, err)
	}
	if _, err := ParseAdminExtensionManifest([]byte(`{"id":"bad","name":"Bad","version":"1","entrypoint_url":"http://evil.example","surfaces":[{"id":"x","label":"X","path":"/","placement":"x"}]}`)); !errors.Is(err, ErrInvalidAdminExtension) {
		t.Fatalf("invalid parsed manifest error = %v", err)
	}
}
