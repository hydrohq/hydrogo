package ui

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/ui/aurora"
	"github.com/ardaozyildirim/hydrogo/internal/ui/dashboardtw"
	"github.com/ardaozyildirim/hydrogo/internal/ui/hydrogoui"
	"github.com/ardaozyildirim/hydrogo/internal/ui/shadcnui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func TestAdminStyleHashesIncludeCompleteShellStyleBlock(t *testing.T) {
	hashes := AdminStyleHashes()
	if len(hashes) == 0 {
		t.Fatal("Admin style hash catalog is empty")
	}
	for _, source := range hashes {
		if source == hash(AdminInterFontCSS+AdminCSS+AdminCSSOverrides+AdminCollectionDescriptionCSS+AdminDashboardCSS+AdminPolishCSS+AdminNavTreeCSS+AdminRichTextCSS+AdminRichTextFixCSS+AdminItemListCSS+AdminFormLayoutCSS+AdminCollectionRecordFormCSS+AdminIconCSS+AdminSearchCSS+AdminToastCSS+AdminInfoTipCSS+AdminResourceCSS+AdminAreaCSS+designsystem.PageLayoutPickerCSS+AdminPermissionsCSS+AdminHydroGoCSS+hydrogoui.CSSText+AdminShadcnCSS+AdminShadcnDashboardLayoutCSS+shadcnui.CSSText+AdminNovaCSS+AdminNovaDensityCSS+AdminNovaTypeCSS+AdminNovaDashboardCSS+AdminNovaComponentsCSS+AdminNovaDashboardLayoutCSS+AdminAuroraCSS+AdminAuroraDensityCSS+AdminAuroraTypeCSS+AdminAuroraDashboardLayoutCSS+aurora.CSSText+dashboardtw.CSSText+AdminFamilySelectCSS+AdminShowcaseCSS+AdminKitCSS+AdminListViewCSS+AdminMediaCSS+AdminMediaFormsCSS+AdminSplitEditorCSS+AdminTabsCSS+AdminSpotlightCSS+AdminResponsiveCSS+AdminUXCSS+AdminDraftAutosaveCSS+AdminPasswordStrengthCSS+AdminEditorStatsCSS+AdminBusinessCSS+AdminRevisionCSS+AdminStatusFilterCSS+AdminAvatarContrastCSS) {
			return
		}
	}
	t.Fatal("Admin CSP catalog does not authorize the complete rendered shell style block")
}

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
