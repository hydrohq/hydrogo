package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
)

type AdminAreaData struct {
	Request                  *http.Request
	Email                    string
	Nonce                    string
	CSRF                     string
	Area                     string
	Locale                   string
	TenantID                 string
	LocaleSaved              bool
	SoundAlertsEnabled       bool
	FontToolsEnabled         bool
	Users                    []*auth.User
	EmailVerificationEnabled bool
	PasswordResetLinkEnabled bool
	MFAEnabled               bool
	MFAAvailable             bool
	MFAStatusLoaded          bool
	Permissions              []string
	IPBans                   []SecurityBanRow
}

type SecurityBanRow struct{ Address, Expires string }

type AdminResourceUsageData struct {
	Email    string
	Nonce    string
	Snapshot observability.RuntimeSnapshot
	Locale   string
}

type adminResourceUsageView struct {
	AdminResourceUsageData
	HeapAlloc string
	HeapInUse string
	HeapSys   string
	Uptime    string
	StartedAt string
}

// RenderDeveloperTabs returns the developer tab bar as markup for pages that
// still assemble their body as a string (modules/site).
func RenderDeveloperTabs(activeTab string) string {
	var out bytes.Buffer
	if err := DeveloperTabs(activeTab).Render(context.Background(), &out); err != nil {
		return ""
	}
	return out.String()
}

func RenderAdminResourceUsage(w io.Writer, data AdminResourceUsageData) error {
	s := data.Snapshot
	view := adminResourceUsageView{
		AdminResourceUsageData: data,
		HeapAlloc:              formatBytes(s.HeapAllocBytes),
		HeapInUse:              formatBytes(s.HeapInUseBytes),
		HeapSys:                formatBytes(s.HeapSysBytes),
		Uptime:                 formatDuration(s.Uptime),
		StartedAt:              s.StartedAt.Local().Format("02.01.2006 15:04:05"),
	}
	if err := AdminStartLocale(w, "Kaynak tüketimi", "GELİŞTİRİCİ", "resources", data.Locale); err != nil {
		return err
	}
	if err := resourceUsagePage(view).Render(context.Background(), w); err != nil {
		return fmt.Errorf("render resource usage: %w", err)
	}
	return AdminEnd(w)
}

func formatBytes(value uint64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB"}
	v := float64(value)
	for _, unit := range units {
		v /= 1024
		if v < 1024 || unit == "GB" {
			return fmt.Sprintf("%.1f %s", v, unit)
		}
	}
	return fmt.Sprintf("%d B", value)
}

func formatDuration(value time.Duration) string {
	if value < time.Minute {
		return fmt.Sprintf("%ds", int(value.Seconds()))
	}
	return fmt.Sprintf("%dd %02dh %02dm", int(value/(24*time.Hour)), int(value/time.Hour)%24, int(value/time.Minute)%60)
}

// itoa formats a number for the views.
func itoa(value any) string { return fmt.Sprint(value) }

// areaText picks the English copy for the "en" admin locale and the Turkish
// copy otherwise.
func areaText(locale, tr, en string) string {
	if locale == "en" {
		return en
	}
	return tr
}

func adminAreaTitles(area, locale string) (string, string) {
	if locale == "en" {
		return adminAreaEnglishLabels(area)
	}
	switch area {
	case "developer":
		return "API & Entegrasyon Konsolu", "GELİŞTİRİCİ MERKEZİ"
	case "operations":
		return "Sistem Sağlığı & Operasyon", "OPERASYON MERKEZİ"
	case "security":
		return "Güvenlik & Erişim Kontrolü", "GÜVENLİK MERKEZİ"
	case "permissions":
		return "Rol & Yetki Matrisi (RBAC)", "YETKİ YÖNETİMİ"
	case "users":
		return "Admin Kullanıcıları", "KULLANICI YÖNETİMİ"
	case "account":
		return "Yönetici Hesap Ayarları", "HESAP BİLGİLERİ"
	case "media":
		return "Medya & Asset Pipeline", "MEDYA KÜTÜPHANESİ"
	case "site":
		return "Site & Yayın Yüzeyi", "SİTE YAPILANDIRMASI"
	default:
		return "Yönetim Alanı", "STUDIO"
	}
}

func adminAreaEnglishLabels(area string) (string, string) {
	switch area {
	case "account":
		return "Administrator Account Settings", "ACCOUNT"
	case "developer":
		return "API & Integration Console", "DEVELOPER CENTER"
	case "security":
		return "Security & Access Control", "SECURITY CENTER"
	case "permissions":
		return "Role & Permission Matrix (RBAC)", "PERMISSION MANAGEMENT"
	case "users":
		return "Admin Users", "USER MANAGEMENT"
	case "media":
		return "Media & Asset Pipeline", "MEDIA LIBRARY"
	case "site":
		return "Site & Publishing Surface", "SITE CONFIGURATION"
	default:
		return "Administration Area", "STUDIO"
	}
}

// adminAreaBody picks the view for an admin area.
func adminAreaBody(data AdminAreaData) templ.Component {
	switch data.Area {
	case "site":
		return siteArea(data.FontToolsEnabled)
	case "developer":
		return developerConsole()
	case "operations":
		return operationsArea()
	case "security":
		return securityArea(data)
	case "permissions":
		return permissionsArea(data.Permissions)
	case "users":
		return usersArea(data)
	case "account":
		return accountArea(data)
	default:
		return areaLoading()
	}
}

func RenderAdminArea(w io.Writer, data AdminAreaData) error {
	title, areaLabel := adminAreaTitles(data.Area, data.Locale)
	activeItem := data.Area
	if activeItem == "security" || activeItem == "control" || activeItem == "operations" {
		activeItem = "developer"
	}
	var startErr error
	if data.Request != nil {
		startErr = AdminStartForRequest(w, data.Request, title, areaLabel, activeItem)
	} else {
		startErr = AdminStartLocale(w, title, areaLabel, activeItem, data.Locale)
	}
	if startErr != nil {
		return startErr
	}
	if err := adminAreaBody(data).Render(context.Background(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}

func securityBanRows(bans []SecurityBanRow) [][]string {
	rows := make([][]string, 0, len(bans))
	for _, item := range bans {
		rows = append(rows, []string{item.Address, item.Expires})
	}
	return rows
}

func userStatus(locale string, active bool) string {
	switch {
	case locale == "tr" && active:
		return "Aktif"
	case locale == "tr":
		return "Pasif"
	case active:
		return "Active"
	default:
		return "Inactive"
	}
}

// userPermissionCatalog lists the permissions the user editor offers: the
// active Edition's catalog, or every known permission without one.
func userPermissionCatalog(permissions []string) []string {
	if len(permissions) == 0 {
		return auth.KnownPermissions
	}
	return permissions
}

// permissionOverride reports a user's override for one permission as
// "inherit", "allow" or "deny".
func permissionOverride(user *auth.User, permission string) string {
	allowed, ok := user.PermissionOverrides[permission]
	switch {
	case !ok:
		return "inherit"
	case allowed:
		return "allow"
	default:
		return "deny"
	}
}
