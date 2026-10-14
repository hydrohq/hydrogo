package transport

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) appearance(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site appearance unavailable", http.StatusServiceUnavailable)
		return
	}
	value := site.AppearanceFromSettings(settings)
	if ui.AdminStartForRequest(w, r, "Marka ve görünüm", "Renkleri ve yazı tipini sitenizin görünümüne uygulayın", "site") != nil {
		return
	}
	view := appearanceView{Font: value.FontFamily, CSRF: security.CSRFTokenFromContext(r.Context()), Fields: []uicomponents.InputProps{
		{ID: "appearance-primary", Name: "primary_color", Type: "color", Label: "Ana renk (açık tema)", Value: value.PrimaryColor, Required: true},
		{ID: "appearance-accent", Name: "accent_color", Type: "color", Label: "Vurgu rengi (açık tema)", Value: value.AccentColor, Required: true},
		{ID: "appearance-dark-primary", Name: "dark_primary_color", Type: "color", Label: "Ana renk (koyu tema)", Value: value.DarkPrimaryColor, Required: true},
		{ID: "appearance-dark-accent", Name: "dark_accent_color", Type: "color", Label: "Vurgu rengi (koyu tema)", Value: value.DarkAccentColor, Required: true},
		{ID: "appearance-logo", Name: "logo_url", Type: "text", Label: "Logo medya yolu (ör. /media/logo.svg)", Value: value.LogoURL},
		{ID: "appearance-dark-logo", Name: "dark_logo_url", Type: "text", Label: "Koyu tema logo medya yolu", Value: value.DarkLogoURL},
		{ID: "appearance-favicon", Name: "favicon_url", Type: "text", Label: "Favicon medya yolu", Value: value.FaviconURL},
		{ID: "appearance-social-image", Name: "social_image_url", Type: "text", Label: "Sosyal paylaşım görseli medya yolu", Value: value.SocialImageURL},
		{ID: "appearance-login-background", Name: "login_background_url", Type: "text", Label: "Giriş ekranı arka plan görseli", Value: value.LoginBackgroundURL},
		{ID: "appearance-login-overlay", Name: "login_overlay_opacity", Type: "range", Label: "Giriş görseli karartma oranı (0–80%)", Value: strconv.Itoa(int(math.Round(value.LoginOverlayOpacity * 100)))},
	}}
	if err := appearanceCard(view).Render(r.Context(), w); err != nil {
		return
	}
	_ = ui.AdminEnd(w)
}

// isMediaPathField marks appearance inputs that hold a media path and get a
// "choose from media" button.
func isMediaPathField(name string) bool { return strings.HasSuffix(name, "_url") }

func (h Handler) saveAppearance(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Görünüm ayarları geçersiz.", http.StatusBadRequest)
		return
	}
	opacityPercent, opacityErr := strconv.Atoi(r.FormValue("login_overlay_opacity"))
	if opacityErr != nil || opacityPercent < 0 || opacityPercent > 80 {
		http.Error(w, "Karartma oranı 0 ile 0,8 arasında olmalıdır.", http.StatusBadRequest)
		return
	}
	opacity := float64(opacityPercent) / 100
	value := site.Appearance{PrimaryColor: r.FormValue("primary_color"), AccentColor: r.FormValue("accent_color"), DarkPrimaryColor: r.FormValue("dark_primary_color"), DarkAccentColor: r.FormValue("dark_accent_color"), FontFamily: r.FormValue("font_family"), LogoURL: r.FormValue("logo_url"), DarkLogoURL: r.FormValue("dark_logo_url"), FaviconURL: r.FormValue("favicon_url"), SocialImageURL: r.FormValue("social_image_url"), LoginBackgroundURL: r.FormValue("login_background_url"), LoginOverlayOpacity: opacity}
	if err := value.Normalize(); err != nil {
		http.Error(w, "Renkler beyaz metinle en az 4,5:1 kontrast sağlamalı ve desteklenen yazı tiplerinden seçilmelidir.", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveAppearance(r.Context(), value); err != nil {
		http.Error(w, "Görünüm kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/appearance", http.StatusSeeOther)
}

func (h Handler) seoSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	var excluded string
	for _, item := range settings {
		if item.Key == "seo.llms_excluded_paths" {
			excluded = item.Value
			break
		}
	}
	if ui.AdminStartForRequest(w, r, "SEO ayarları", "Yayınlanmış içeriklerin arama ve keşif görünümü", "site") != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Textarea(uicomponents.TextareaProps{ID: "llms-excluded-paths", Name: "paths", Label: "LLM metin dosyalarından çıkarılacak yollar (satır başına bir yol)", Value: excluded, Rows: 8, MaxLength: 4096, HasMaxLength: true}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "Ayarları kaydet", Type: "submit", Variant: uicomponents.Primary}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	_ = uicomponents.Card(uicomponents.CardProps{Title: "Yayınlanan içerik", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/seo", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: body})}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveSEOSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid SEO settings", http.StatusBadRequest)
		return
	}
	lines := strings.Split(r.FormValue("paths"), "\n")
	paths := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(path)
		if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(path, "\\\r\n\x00") || len(path) > 512 {
			http.Error(w, "invalid SEO settings", http.StatusBadRequest)
			return
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	if len(paths) > 100 {
		http.Error(w, "invalid SEO settings", http.StatusBadRequest)
		return
	}
	value := site.Setting{Key: "seo.llms_excluded_paths", Value: strings.Join(paths, "\n"), Public: true, UpdatedAt: time.Now().UTC()}
	if err := h.Service.SaveSetting(r.Context(), value); err != nil {
		http.Error(w, "SEO settings could not be saved", siteStatus(err))
		return
	}
	if h.InvalidateDiscovery != nil {
		h.InvalidateDiscovery()
	}
	http.Redirect(w, r, "/admin/site/seo", http.StatusSeeOther)
}

var localFontFamilies = []string{"Inter", "Lato", "Montserrat", "Open Sans", "Roboto", "Source Sans 3"}

func (h Handler) fontPicker(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	current := ""
	for _, item := range settings {
		if item.Key == "design.font_css" {
			current = item.Value
			break
		}
	}
	if ui.AdminStartForRequest(w, r, "Yerel yazı tipi", "Kendi siteniz için yerel bir yazı tipi seçin", "site") != nil {
		return
	}
	body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
		for _, component := range []templ.Component{
			uicomponents.Select(uicomponents.SelectProps{ID: "font-family", Name: "family", Label: "Yazı tipi", Options: localFontFamilies, Required: true}),
			uicomponents.Button(uicomponents.ButtonProps{Label: "Yazı tipini indir ve uygula", Type: "submit", Variant: uicomponents.Primary}),
		} {
			if err := component.Render(ctx, output); err != nil {
				return err
			}
		}
		return nil
	})
	activeFont := ""
	if current != "" {
		activeFont = filepath.Base(current)
	}
	_ = fontPickerStatus(activeFont, r.URL.Query().Get("saved") == "1").Render(r.Context(), w)
	_ = uicomponents.Card(uicomponents.CardProps{Title: "Yazı tipi seç", Body: uicomponents.Form(uicomponents.FormProps{Action: "/admin/site/fonts", Method: "post", CSRF: security.CSRFTokenFromContext(r.Context()), Body: body})}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) localizeFont(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.FontsDir == "" {
		http.Error(w, "Bu özellik için yerel yazı tipi klasörü yapılandırılmalı.", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Yazı tipi seçimi geçersiz.", http.StatusBadRequest)
		return
	}
	family := r.FormValue("family")
	allowed := false
	for _, option := range localFontFamilies {
		if option == family {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "Yazı tipi seçimi geçersiz.", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	cssPath, err := designsystem.LocalizeGoogleFont(ctx, family, h.FontsDir, h.fontHTTPClient)
	if err != nil {
		http.Error(w, "Yazı tipi indirilemedi. Lütfen daha sonra tekrar deneyin.", http.StatusBadGateway)
		return
	}
	publicPath := "/fonts/" + filepath.Base(cssPath)
	if err := h.Service.SaveSetting(r.Context(), site.Setting{Key: "design.font_css", Value: publicPath, Public: true, UpdatedAt: time.Now().UTC()}); err != nil {
		http.Error(w, "Yazı tipi ayarı kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/fonts?saved=1", http.StatusSeeOther)
}
