package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension"
	"github.com/ardaozyildirim/hydrogo/internal/modules"
)

type sidebarModule struct {
	name     string
	manifest modules.Manifest
}

func (m sidebarModule) Name() string                     { return m.name }
func (m sidebarModule) Routes() []extension.ModuleRoute  { return nil }
func (m sidebarModule) ModuleManifest() modules.Manifest { return m.manifest }

func TestRenderModuleSidebarUsesActiveRegistryOnly(t *testing.T) {
	registry := modules.NewRegistry()
	if err := registry.Register(sidebarModule{name: "news", manifest: modules.Manifest{ID: "news", Admin: modules.AdminManifest{Route: "/admin/news", Label: "Haberler", Group: "Content", Order: 1}}}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RenderModuleSidebar(&out, registry, "/admin/news"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `href="/admin/news"`) || !strings.Contains(out.String(), "is-active") || !strings.Contains(out.String(), "CONTENT") {
		t.Fatalf("unexpected sidebar: %s", out.String())
	}
}

func TestModuleSidebarRejectsExternalRoute(t *testing.T) {
	registry := modules.NewRegistry()
	if err := registry.Register(sidebarModule{name: "unsafe", manifest: modules.Manifest{ID: "unsafe", Admin: modules.AdminManifest{Route: "https://evil.example", Label: "Unsafe"}}}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RenderModuleSidebar(&out, registry, ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "evil.example") {
		t.Fatalf("external route leaked into sidebar: %s", out.String())
	}
}
