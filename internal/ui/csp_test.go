package ui

import (
	"strings"
	"testing"
)

func TestAdminScriptHashesAuthorizeGeneratedBlockEditorLoader(t *testing.T) {
	hashes, err := AdminScriptHashes()
	if err != nil {
		t.Fatal(err)
	}
	want := hash(AdminBlockEditorJS)
	for _, source := range hashes {
		if source == want {
			for _, contract := range []string{"HydroInitGeneratedBlockEditors", "mount.dataset.editorjsScript", "mount.dataset.editorjsPreview", "data.blocks.length===0", "MutationObserver(labelBlocks)", "aria-label", "role','textbox", "aria-multiline", "/vendor/htmx/htmx.min.js", "window.htmx?.process(document.body)"} {
				if !strings.Contains(AdminBlockEditorJS, contract) {
					t.Fatalf("hash-authorized block editor loader is missing %q", contract)
				}
			}
			return
		}
	}
	t.Fatal("Admin CSP catalog does not authorize generated block editor loader")
}
