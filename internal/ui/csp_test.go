package ui

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestAdminScriptHashesAuthorizeGeneratedBlockEditorLoader(t *testing.T) {
	// The loader is a same-origin file now, allowed by script-src 'self'.
	var out bytes.Buffer
	if err := AdminStart(&out, "Editor", "", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<script defer src="/admin-assets/js/block-editor.js"></script>`) {
		t.Fatal("admin shell does not load the generated block editor loader")
	}
	loader := adminAsset(t, "js/block-editor.js")
	for _, contract := range []string{"HydroInitGeneratedBlockEditors", "mount.dataset.editorjsScript", "mount.dataset.editorjsPreview", "data.blocks.length===0", "MutationObserver(labelBlocks)", "aria-label", "role','textbox", "aria-multiline", "/vendor/htmx/htmx.min.js", "window.htmx?.process(document.body)"} {
		if !strings.Contains(loader, contract) {
			t.Fatalf("block editor loader is missing %q", contract)
		}
	}
}

func TestAdminScriptHashesAuthorizeTheThemeBootstrap(t *testing.T) {
	hashes, err := AdminScriptHashes()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := AdminStart(&out, "CSP", "", ""); err != nil {
		t.Fatal(err)
	}
	inline, _ := security.InlineCSPHashes(out.Bytes())
	if len(inline) == 0 {
		t.Fatal("admin shell has no inline theme bootstrap")
	}
	for _, want := range inline {
		if !slices.Contains(hashes, want) {
			t.Fatalf("admin CSP catalog misses inline shell script hash %s", want)
		}
	}
}
