package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui/hydrogoui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func hash(data string) string {
	sum := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func AdminScriptHashes() ([]string, error) {
	hashes := []string{
		hash(AdminThemeBootJS),
		hash(AdminBlockEditorJS),
		hash(AdminIconJS),
		hash(AdminSPARouterJS + ";\n" + AdminSidebarUsersJS + ";\n" + AdminResponsiveJS + ";\n" + adminUXScript() + ";\n" + AdminResetLinkJS + ";\n" + AdminDraftAutosaveJS + ";\n" + AdminPasswordStrengthJS + ";\n" + AdminPasswordSubmitJS + ";\n" + AdminEditorStatsJS + ";\n" + AdminPageWeightJS + ";\n" + AdminStatusFilterJS + ";\n" + AdminListStateJS + ";\n" + AdminListViewJS + ";\n" + AdminCopyTextJS + ";\n" + AdminUXHotkeysJS + ";\n" + AdminNotificationJS + ";\n" + AdminStaffNoticeJS + ";\n" + AdminRealtimeChimeJS + ";\n" + AdminDevicePreviewJS + ";\n" + AdminTourJS + ";\n" + AdminGuideJS + ";\n" + AdminSessionWarningJS + ";\n"),
		hash(AdminEnglishUIJS),
		hash(`document.querySelectorAll('a[target="_blank"]').forEach(function(link){link.setAttribute('rel','noopener noreferrer')});`),
	}
	for _, render := range []func(*bytes.Buffer) error{
		func(out *bytes.Buffer) error { return adminControlTemplate.Execute(out, AdminControlData{}) },
		func(out *bytes.Buffer) error { return adminReportsPathTemplate.Execute(out, AdminDashboardData{}) },
		func(out *bytes.Buffer) error {
			return RenderAdminArea(out, AdminAreaData{Area: "developer", Locale: "tr"})
		},
	} {
		var document bytes.Buffer
		if err := render(&document); err != nil {
			return nil, fmt.Errorf("render Admin script catalog: %w", err)
		}
		scripts, _ := security.InlineCSPHashes(document.Bytes())
		hashes = append(hashes, scripts...)
	}
	return uniqueScriptHashes(hashes), nil
}

func uniqueScriptHashes(hashes []string) []string {
	seen := make(map[string]struct{}, len(hashes))
	unique := make([]string, 0, len(hashes))
	for _, source := range hashes {
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		unique = append(unique, source)
	}
	return unique
}

func AdminStyleHashes() []string {
	return []string{
		hash(AdminInterFontCSS + AdminCSS + AdminCSSOverrides + AdminCollectionDescriptionCSS + AdminDashboardCSS + AdminPolishCSS + AdminNavTreeCSS + AdminRichTextCSS + AdminRichTextFixCSS + AdminBlockEditorCSS + AdminItemListCSS + AdminFormLayoutCSS + AdminCollectionRecordFormCSS + AdminIconCSS + AdminSearchCSS + AdminToastCSS + AdminInfoTipCSS + AdminResourceCSS + AdminAreaCSS + designsystem.PageLayoutPickerCSS + AdminPermissionsCSS + AdminHydroGoCSS + hydrogoui.CSSText + AdminFormControlBaseCSS + AdminShowcaseCSS + AdminKitCSS + AdminListViewCSS + AdminMediaCSS + AdminMediaFormsCSS + AdminSplitEditorCSS + AdminTabsCSS + AdminSpotlightCSS + AdminResponsiveCSS + AdminUXCSS + AdminDraftAutosaveCSS + AdminPasswordStrengthCSS + AdminEditorStatsCSS + AdminBusinessCSS + AdminRevisionCSS + AdminStatusFilterCSS + AdminAvatarContrastCSS),
		hash(AdminInterFontCSS + AdminCSS),
		hash(AdminCSSOverrides + AdminCollectionDescriptionCSS),
		hash(AdminDashboardCSS),
		hash(AdminPolishCSS),
		hash(AdminNavTreeCSS),
		hash(AdminRichTextCSS + AdminRichTextFixCSS + AdminBlockEditorCSS),
		hash(AdminItemListCSS),
		hash(AdminFormLayoutCSS),
		hash(AdminCollectionRecordFormCSS),
		hash(AdminIconCSS),
		hash(AdminSearchCSS),
		hash(AdminToastCSS),
		hash(AdminInfoTipCSS + AdminResourceCSS + AdminAreaCSS + designsystem.PageLayoutPickerCSS + AdminPermissionsCSS + AdminHydroGoCSS + hydrogoui.CSSText + AdminFormControlBaseCSS + AdminShowcaseCSS + AdminKitCSS + AdminListViewCSS + AdminMediaCSS + AdminMediaFormsCSS + AdminSplitEditorCSS + AdminTabsCSS + AdminSpotlightCSS + AdminResponsiveCSS + AdminUXCSS + AdminDraftAutosaveCSS + AdminPasswordStrengthCSS + AdminEditorStatsCSS + AdminBusinessCSS + AdminRevisionCSS + AdminStatusFilterCSS + AdminAvatarContrastCSS),
	}
}
