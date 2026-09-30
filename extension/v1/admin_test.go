package v1

import (
	"crypto/ed25519"
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

func TestSignedAdminExtensionManifestVerifiesAgainstConfiguredKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignAdminExtensionManifest(validAdminManifest(), "vendor-2026", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := signed.Verify(map[string]ed25519.PublicKey{"vendor-2026": publicKey})
	if err != nil || verified.ID != "acme-reports" {
		t.Fatalf("verified manifest = %#v, err = %v", verified, err)
	}
	signed.Manifest.Name = "tampered"
	if _, err := signed.Verify(map[string]ed25519.PublicKey{"vendor-2026": publicKey}); !errors.Is(err, ErrInvalidAdminExtensionSignature) {
		t.Fatalf("tampered manifest error = %v", err)
	}
	if _, err := signed.Verify(map[string]ed25519.PublicKey{"vendor-2027": publicKey}); !errors.Is(err, ErrInvalidAdminExtensionSignature) {
		t.Fatalf("unknown key error = %v", err)
	}
}

func TestAdminManifestUnboundedStrings(t *testing.T) {
	hugeString := string(make([]byte, 10*1024*1024)) // 10MB string
	manifest := AdminExtensionManifest{
		ID:            "valid-id",
		Name:          hugeString, // unbounded!
		Version:       hugeString, // unbounded!
		EntrypointURL: "https://example.com",
		Surfaces: []AdminSurface{
			{
				ID:        "surf-1",
				Label:     hugeString, // unbounded!
				Placement: hugeString, // unbounded!
				Path:      "/valid",
			},
		},
	}

	if err := manifest.Validate(); err == nil {
		t.Fatalf("VULNERABILITY: AdminExtensionManifest allows unbounded string sizes in Name, Version, Label, and Placement! This causes memory DoS.")
	}
}
