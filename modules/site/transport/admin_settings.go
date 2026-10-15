package transport

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// plainSettingKeys lists the string settings the generic endpoint edits; every
// one of them is public. Structured settings (maintenance, staff notice,
// social links, business profile, translations) have their own endpoints.
var plainSettingKeys = map[string]func(string) bool{
	"site.name":                   maxLen(200),
	"site.title":                  maxLen(200),
	"site.description":            maxLen(1000),
	"site.home_path":              func(string) bool { return true }, // ValidateSetting checks the path.
	"seo.llms_excluded_paths":     validExcludedPaths,
	"appearance.favicon_url":      validAssetURL,
	"appearance.social_image_url": validAssetURL,
}

func maxLen(limit int) func(string) bool {
	return func(value string) bool { return len(value) <= limit && !strings.ContainsAny(value, "\x00") }
}

func validAssetURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if len(value) > 2048 || strings.ContainsAny(value, "\\\r\n\x00") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//") && parsed.Host == ""
	}
	return parsed.Scheme == "https" && parsed.Host != ""
}

func validExcludedPaths(value string) bool {
	lines := strings.Split(value, "\n")
	count := 0
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(path)
		if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(path, "\\\r\n\x00") || len(path) > 512 {
			return false
		}
		count++
	}
	return count <= 100
}

type settingView struct {
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	Public    bool       `json:"public"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func (h Handler) adminSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	stored, err := h.Service.ListSettings(r.Context())
	if err != nil {
		writeSiteError(w, r, err, "site settings unavailable")
		return
	}
	byKey := make(map[string]site.Setting, len(stored))
	for _, setting := range stored {
		byKey[setting.Key] = setting
	}
	keys := make([]string, 0, len(plainSettingKeys))
	for key := range plainSettingKeys {
		keys = append(keys, key)
	}
	sortStrings(keys)
	result := make([]settingView, 0, len(keys))
	for _, key := range keys {
		view := settingView{Key: key, Public: true}
		if setting, ok := byKey[key]; ok {
			view.Value = setting.Value
			if !setting.UpdatedAt.IsZero() {
				updated := setting.UpdatedAt
				view.UpdatedAt = &updated
			}
		}
		result = append(result, view)
	}
	api.WriteData(w, http.StatusOK, result, nil)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func (h Handler) adminSaveSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		Settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"settings"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if len(payload.Settings) == 0 || len(payload.Settings) > len(plainSettingKeys) {
		api.WriteError(w, r, http.StatusBadRequest, "settings list is empty or too long", nil)
		return
	}
	fields := map[string]string{}
	values := make([]site.Setting, 0, len(payload.Settings))
	for _, entry := range payload.Settings {
		key := strings.TrimSpace(entry.Key)
		validate, known := plainSettingKeys[key]
		switch {
		case !known:
			fields[key] = "unknown"
		case !validate(entry.Value):
			fields[key] = "invalid"
		default:
			value := site.Setting{Key: key, Value: strings.TrimSpace(entry.Value), Public: true, UpdatedAt: time.Now().UTC()}
			if site.ValidateSetting(value) != nil {
				fields[key] = "invalid"
				continue
			}
			values = append(values, value)
		}
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	for _, value := range values {
		if err := h.Service.SaveSetting(r.Context(), value); err != nil {
			writeSiteError(w, r, err, "setting could not be saved")
			return
		}
	}
	if h.InvalidateDiscovery != nil {
		h.InvalidateDiscovery()
	}
	h.adminSettings(w, r)
}

func (h Handler) privateSetting(w http.ResponseWriter, r *http.Request, key string) (site.Setting, bool) {
	setting, _, err := h.Service.GetSetting(r.Context(), key)
	if err != nil {
		writeSiteError(w, r, err, "setting unavailable")
		return site.Setting{}, false
	}
	return setting, true
}

type maintenanceView struct {
	Enabled       bool     `json:"enabled"`
	Allowlist     []string `json:"allowlist"`
	StartsAt      string   `json:"starts_at,omitempty"`
	EndsAt        string   `json:"ends_at,omitempty"`
	Headline      string   `json:"headline,omitempty"`
	Message       string   `json:"message,omitempty"`
	HasPreviewKey bool     `json:"has_preview_key"`
}

func maintenanceViewOf(value site.MaintenanceSettings) maintenanceView {
	allowlist := value.Allowlist
	if allowlist == nil {
		allowlist = []string{}
	}
	return maintenanceView{Enabled: value.Enabled, Allowlist: allowlist, StartsAt: value.StartsAt, EndsAt: value.EndsAt, Headline: value.Headline, Message: value.Message, HasPreviewKey: value.PreviewKeyHash != ""}
}

func (h Handler) adminMaintenance(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	setting, ok := h.privateSetting(w, r, site.MaintenanceSettingKey)
	if !ok {
		return
	}
	api.WriteData(w, http.StatusOK, maintenanceViewOf(site.MaintenanceFromSetting(setting)), nil)
}

func (h Handler) adminSaveMaintenance(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		maintenanceView
		PreviewKey      string `json:"preview_key"`
		ClearPreviewKey bool   `json:"clear_preview_key"`
	}
	if !decode(w, r, &payload) {
		return
	}
	current, ok := h.privateSetting(w, r, site.MaintenanceSettingKey)
	if !ok {
		return
	}
	settings := site.MaintenanceSettings{Enabled: payload.Enabled, Allowlist: trimAll(payload.Allowlist), StartsAt: strings.TrimSpace(payload.StartsAt), EndsAt: strings.TrimSpace(payload.EndsAt), Headline: strings.TrimSpace(payload.Headline), Message: strings.TrimSpace(payload.Message), PreviewKeyHash: site.MaintenanceFromSetting(current).PreviewKeyHash}
	if payload.ClearPreviewKey {
		settings.PreviewKeyHash = ""
	}
	if payload.PreviewKey != "" {
		settings.PreviewKeyHash = site.MaintenancePreviewKeyHash(payload.PreviewKey)
		if settings.PreviewKeyHash == "" {
			api.WriteValidation(w, r, map[string]string{"preview_key": "invalid"})
			return
		}
	}
	setting, err := settings.Setting()
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"maintenance": "invalid"})
		return
	}
	if err := h.Service.SaveSetting(r.Context(), setting); err != nil {
		writeSiteError(w, r, err, "maintenance settings could not be saved")
		return
	}
	api.WriteData(w, http.StatusOK, maintenanceViewOf(settings), nil)
}

func trimAll(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (h Handler) adminStaffNotice(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	setting, ok := h.privateSetting(w, r, site.StaffNoticeSettingKey)
	if !ok {
		return
	}
	api.WriteData(w, http.StatusOK, site.StaffBroadcastNoticeFromSetting(setting), nil)
}

func (h Handler) adminSaveStaffNotice(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var notice site.StaffBroadcastNotice
	if !decode(w, r, &notice) {
		return
	}
	notice.Message, notice.Severity = strings.TrimSpace(notice.Message), strings.TrimSpace(notice.Severity)
	setting, err := notice.Setting()
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"notice": "invalid"})
		return
	}
	if err := h.Service.SaveSetting(r.Context(), setting); err != nil {
		writeSiteError(w, r, err, "staff notice could not be saved")
		return
	}
	api.WriteData(w, http.StatusOK, notice, nil)
}

type socialProfileView struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

type socialLinksView struct {
	Platforms []string            `json:"platforms"`
	Profiles  []socialProfileView `json:"profiles"`
}

func socialLinksViewOf(profiles []site.SocialProfile) socialLinksView {
	view := socialLinksView{Platforms: site.SocialPlatforms(), Profiles: make([]socialProfileView, 0, len(profiles))}
	for _, profile := range profiles {
		view.Profiles = append(view.Profiles, socialProfileView{Platform: profile.Platform, URL: profile.URL})
	}
	return view
}

func (h Handler) adminSocialLinks(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	settings, err := h.Service.ListSettings(r.Context())
	if err != nil {
		writeSiteError(w, r, err, "social links unavailable")
		return
	}
	api.WriteData(w, http.StatusOK, socialLinksViewOf(site.SocialProfilesFromSettings(settings)), nil)
}

func (h Handler) adminSaveSocialLinks(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var payload struct {
		Profiles []socialProfileView `json:"profiles"`
	}
	if !decode(w, r, &payload) {
		return
	}
	profiles := make([]site.SocialProfile, 0, len(payload.Profiles))
	for _, profile := range payload.Profiles {
		profiles = append(profiles, site.SocialProfile{Platform: profile.Platform, URL: profile.URL})
	}
	normalized, err := site.NormalizeSocialProfiles(profiles)
	if err != nil {
		api.WriteValidation(w, r, map[string]string{"profiles": "invalid"})
		return
	}
	if err := h.Service.SaveSocialProfiles(r.Context(), normalized); err != nil {
		writeSiteError(w, r, err, "social links could not be saved")
		return
	}
	api.WriteData(w, http.StatusOK, socialLinksViewOf(normalized), nil)
}

func (h Handler) adminBusinessProfile(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	settings, err := h.Service.ListSettings(r.Context())
	if err != nil {
		writeSiteError(w, r, err, "business profile unavailable")
		return
	}
	profile, ok := site.BusinessProfileFromSettings(settings)
	if !ok {
		api.WriteData(w, http.StatusOK, nil, nil)
		return
	}
	api.WriteData(w, http.StatusOK, profile, nil)
}

func (h Handler) adminSaveBusinessProfile(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	var profile site.BusinessProfile
	if !decode(w, r, &profile) {
		return
	}
	if err := profile.Validate(); err != nil {
		api.WriteValidation(w, r, map[string]string{"profile": "invalid"})
		return
	}
	if err := h.Service.SaveBusinessProfile(r.Context(), profile); err != nil {
		writeSiteError(w, r, err, "business profile could not be saved")
		return
	}
	api.WriteData(w, http.StatusOK, profile, nil)
}

type translationRow struct {
	Key         string `json:"key"`
	Default     string `json:"default"`
	Translation string `json:"translation"`
	Missing     bool   `json:"missing"`
}

type translationsView struct {
	Locale  string           `json:"locale"`
	Locales []string         `json:"locales"`
	Rows    []translationRow `json:"rows"`
}

func translationsViewOf(tenantID, locale string) translationsView {
	rows := i18n.CatalogRows(tenantID, locale)
	view := translationsView{Locale: locale, Locales: i18n.SupportedCatalogLocales(), Rows: make([]translationRow, 0, len(rows))}
	for _, row := range rows {
		view.Rows = append(view.Rows, translationRow{Key: row.Key, Default: row.Default, Translation: row.Translation, Missing: row.Missing})
	}
	return view
}

func catalogLocale(w http.ResponseWriter, r *http.Request) (string, bool) {
	requested := r.PathValue("locale")
	if !i18n.IsSupportedCatalogLocale(requested) {
		api.WriteError(w, r, http.StatusNotFound, "unsupported translation locale", nil)
		return "", false
	}
	return i18n.ResolveCatalogLocale(requested), true
}

func (h Handler) adminTranslations(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	locale, ok := catalogLocale(w, r)
	if !ok {
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	api.WriteData(w, http.StatusOK, translationsViewOf(scope.TenantID, locale), nil)
}

// adminSaveTranslations stores overrides for the given keys; an empty value
// removes the override and falls back to the catalog.
func (h Handler) adminSaveTranslations(w http.ResponseWriter, r *http.Request) {
	if !h.requireSiteManage(w, r) {
		return
	}
	locale, ok := catalogLocale(w, r)
	if !ok {
		return
	}
	var payload struct {
		Values map[string]string `json:"values"`
	}
	if !decode(w, r, &payload) {
		return
	}
	if len(payload.Values) == 0 || len(payload.Values) > 500 {
		api.WriteError(w, r, http.StatusBadRequest, "values map is empty or too long", nil)
		return
	}
	fields := map[string]string{}
	for key, value := range payload.Values {
		if !i18n.IsCatalogKey(key) {
			fields[key] = "unknown"
		} else if len(value) > 1000 || strings.ContainsAny(value, "\x00\r\n") {
			fields[key] = "invalid"
		}
	}
	if len(fields) > 0 {
		api.WriteValidation(w, r, fields)
		return
	}
	scope, _ := tenant.FromContext(r.Context())
	for key, value := range payload.Values {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: i18n.OverrideSettingKey(locale, key), Value: value, Public: true}); err != nil {
			writeSiteError(w, r, err, "translation could not be saved")
			return
		}
		i18n.SetOverride(scope.TenantID, locale, key, value)
	}
	api.WriteData(w, http.StatusOK, translationsViewOf(scope.TenantID, locale), nil)
}
