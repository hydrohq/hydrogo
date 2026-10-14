package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func TestAdminUXTemplComponentsRenderAccessibleBoundedFlows(t *testing.T) {
	tests := []struct {
		name   string
		render func(*bytes.Buffer) error
		want   []string
	}{
		{"import", func(out *bytes.Buffer) error {
			return ImportWizard([]string{"Başlık", "Metin"}, "csrf", true).Render(context.Background(), out)
		}, []string{".xlsx", "Sütunları eşleştirin", "map_title", "5.000 satır"}},
		{"import preview", func(out *bytes.Buffer) error {
			return ImportPreview("csrf", map[string]string{"slug": "0", "title": "1", "body": "2"}, 3, 1, 2).Render(context.Background(), out)
		}, []string{"data-import-preview", "3 yeni taslak", "1 yinelenen", `name="action" value="import"`, `name="map_slug" value="0"`}},
		{"notifications", func(out *bytes.Buffer) error {
			return NotificationCenter([]NotificationItem{{ID: "internal-id", Title: "İçerik hazır", Message: "Sayfa yayında", TargetPath: "/admin/pages"}}, "csrf", false, "UTC", "monthly", true).Render(context.Background(), out)
		}, []string{"Bildirimler", "İçerik hazır", "/admin/pages", "_csrf"}},
		{"translation", func(out *bytes.Buffer) error {
			return TranslationEditor(TranslationEditorData{SourceTitle: "Kaynak", SourceBody: "Metin", Locale: "en", Action: "/admin/pages/p/translate", CSRF: "csrf"}).Render(context.Background(), out)
		}, []string{"data-translation-source", "data-translation-target", "Kaynak içerik", "Çeviriyi kaydet"}},
		{"visual revision", func(out *bytes.Buffer) error {
			return VisualRevisionCompare("<h1>Eski</h1>", "<h1>Yeni</h1>", "Sürüm 1", "Geçerli taslak").Render(context.Background(), out)
		}, []string{"data-visual-revision", `sandbox=""`, "Sürüm 1", "Geçerli taslak"}},
		{"guide tabs", func(out *bytes.Buffer) error {
			return GuideCenter("https://video.example.test/guide").Render(context.Background(), out)
		}, []string{`role="tablist"`, `data-guide-tab="video"`, `https://video.example.test/guide`, `rel="noopener noreferrer"`}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := test.render(&out); err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

func TestAdminPasswordStrengthUsesDelegatedAccessibleFeedback(t *testing.T) {
	var out bytes.Buffer
	if err := designsystem.PasswordStrengthMeter().Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-password-meter`, `data-password-feedback`, `aria-live="polite"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("admin password form missing %q", want)
		}
	}
	for _, want := range []string{"document.addEventListener('input'", "document.addEventListener('submit'", "[data-password-feedback]", "yeni ve güçlü bir parola seçin"} {
		if !strings.Contains(adminAsset(t, "js/password-strength.js")+adminAsset(t, "js/password-submit.js"), want) {
			t.Fatalf("password strength behavior missing %q", want)
		}
	}
}

func TestAdminPasswordResetCopyUsesProtectedNoStoreEndpoint(t *testing.T) {
	for _, want := range []string{"[data-password-reset-link]", "X-CSRF-Token", "credentials:'same-origin'", "navigator.clipboard.writeText", "expires in one hour"} {
		if !strings.Contains(adminAsset(t, "js/reset-link.js"), want) {
			t.Fatalf("reset link behavior missing %q", want)
		}
	}
}

func TestAdminDraftAutosaveIsUserScopedAndRecoversOnlyEditableFields(t *testing.T) {
	for _, want := range []string{"studioUserScope", "hydro-draft:", "data-autosave-resource", "localStorage.setItem", "localStorage.removeItem", "Taslağı geri yükle", "Sil", "block_csrf_", "3*1024*1024", "30*24*60*60*1000"} {
		if !strings.Contains(adminAsset(t, "js/draft-autosave.js"), want) {
			t.Fatalf("autosave behavior missing %q", want)
		}
	}
	if !strings.Contains(adminAsset(t, "js/draft-autosave.js"), "'password','file'") {
		t.Fatal("autosave must exclude password controls")
	}
}

func TestRevisionDiffEscapesUntrustedFieldValues(t *testing.T) {
	var out bytes.Buffer
	if err := RevisionDiffPanel([]adminstudio.FieldDiff{{Field: "title", Before: "<script>bad</script>", After: "Yeni"}}, "Editor", "bugün", "/admin/pages/p/revisions/1/restore", "csrf").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>bad") || !strings.Contains(out.String(), "&lt;script&gt;") {
		t.Fatal("revision diff did not escape saved content")
	}
}
