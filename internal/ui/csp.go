package ui

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/ardaozyildirim/hydrogo/internal/ui/aurora"
	"github.com/ardaozyildirim/hydrogo/internal/ui/dashboardtw"
	"github.com/ardaozyildirim/hydrogo/internal/ui/hydrogoui"
	"github.com/ardaozyildirim/hydrogo/internal/ui/shadcnui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func hash(data string) string {
	sum := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func AdminScriptHashes() []string {
	return []string{
		hash(`(()=>{try{const key='hydro-ui-theme-v2';const saved=localStorage.getItem(key);const variant=localStorage.getItem('hydro-theme')||(window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light');if(saved){const parts=saved.split('-');document.documentElement.setAttribute('data-ui-theme',['hydrogo','shadcn','nova','aurora'].includes(parts[0])?parts[0]:'hydrogo');document.documentElement.setAttribute('data-theme',parts[1]==='dark'?'dark':'light');}else{document.documentElement.setAttribute('data-ui-theme','hydrogo');document.documentElement.setAttribute('data-theme',variant);}}catch(_){document.documentElement.setAttribute('data-ui-theme','hydrogo');document.documentElement.setAttribute('data-theme','dark');}})();`),
		hash(AdminBlockEditorJS),
		hash(AdminIconJS),
		hash(AdminThemeMigrationJS + ";\n" + AdminSPARouterJS + ";\n" + AdminSidebarUsersJS + ";\n" + AdminResponsiveJS + ";\n" + adminUXScript() + ";\n" + AdminResetLinkJS + ";\n" + AdminDraftAutosaveJS + ";\n" + AdminPasswordStrengthJS + ";\n" + AdminPasswordSubmitJS + ";\n" + AdminEditorStatsJS + ";\n" + AdminPageWeightJS + ";\n" + AdminStatusFilterJS + ";\n" + AdminListStateJS + ";\n" + AdminListViewJS + ";\n" + AdminCopyTextJS + ";\n" + AdminUXHotkeysJS + ";\n" + AdminNotificationJS + ";\n" + AdminStaffNoticeJS + ";\n" + AdminRealtimeChimeJS + ";\n" + AdminDevicePreviewJS + ";\n" + AdminTourJS + ";\n" + AdminGuideJS + ";\n" + AdminSessionWarningJS + ";\n"),
		hash(AdminEnglishUIJS),
		hash(`document.querySelectorAll('a[target="_blank"]').forEach(function(link){link.setAttribute('rel','noopener noreferrer')});`),
	}
}

func AdminStyleHashes() []string {
	return []string{
		hash(AdminInterFontCSS + AdminCSS + AdminCSSOverrides + AdminDashboardCSS + AdminPolishCSS + AdminNavTreeCSS + AdminRichTextCSS + AdminRichTextFixCSS + AdminItemListCSS + AdminFormLayoutCSS + AdminIconCSS + AdminSearchCSS + AdminToastCSS + AdminInfoTipCSS + AdminResourceCSS + AdminAreaCSS + designsystem.PageLayoutPickerCSS + AdminPermissionsCSS + AdminHydroGoCSS + hydrogoui.CSSText + AdminShadcnCSS + AdminShadcnDashboardLayoutCSS + shadcnui.CSSText + AdminNovaCSS + AdminNovaDensityCSS + AdminNovaTypeCSS + AdminNovaDashboardCSS + AdminNovaComponentsCSS + AdminNovaDashboardLayoutCSS + AdminAuroraCSS + AdminAuroraDensityCSS + AdminAuroraTypeCSS + AdminAuroraDashboardLayoutCSS + aurora.CSSText + dashboardtw.CSSText + AdminFamilySelectCSS + AdminShowcaseCSS + AdminKitCSS + AdminListViewCSS + AdminMediaCSS + AdminMediaFormsCSS + AdminSplitEditorCSS + AdminTabsCSS + AdminSpotlightCSS + AdminResponsiveCSS + AdminUXCSS + AdminDraftAutosaveCSS + AdminPasswordStrengthCSS + AdminEditorStatsCSS + AdminBusinessCSS + AdminRevisionCSS + AdminStatusFilterCSS),
		hash(AdminInterFontCSS + AdminCSS),
		hash(AdminCSSOverrides),
		hash(AdminDashboardCSS),
		hash(AdminPolishCSS),
		hash(AdminNavTreeCSS),
		hash(AdminRichTextCSS + AdminRichTextFixCSS),
		hash(AdminItemListCSS),
		hash(AdminFormLayoutCSS),
		hash(AdminIconCSS),
		hash(AdminSearchCSS),
		hash(AdminToastCSS),
		hash(AdminInfoTipCSS + AdminResourceCSS + AdminAreaCSS + designsystem.PageLayoutPickerCSS + AdminPermissionsCSS + AdminHydroGoCSS + hydrogoui.CSSText + AdminShadcnCSS + AdminShadcnDashboardLayoutCSS + shadcnui.CSSText + AdminNovaCSS + AdminNovaDensityCSS + AdminNovaTypeCSS + AdminNovaDashboardCSS + AdminNovaComponentsCSS + AdminNovaDashboardLayoutCSS + AdminAuroraCSS + AdminAuroraDensityCSS + AdminAuroraTypeCSS + AdminAuroraDashboardLayoutCSS + aurora.CSSText + dashboardtw.CSSText + AdminFamilySelectCSS + AdminShowcaseCSS + AdminKitCSS + AdminListViewCSS + AdminMediaCSS + AdminMediaFormsCSS + AdminSplitEditorCSS + AdminTabsCSS + AdminSpotlightCSS + AdminResponsiveCSS + AdminUXCSS + AdminDraftAutosaveCSS + AdminPasswordStrengthCSS + AdminEditorStatsCSS + AdminBusinessCSS + AdminRevisionCSS + AdminStatusFilterCSS),
	}
}
