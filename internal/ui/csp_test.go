package ui

import (
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
