package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
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
		{"notifications", func(out *bytes.Buffer) error {
			return NotificationCenter([]NotificationItem{{ID: "internal-id", Title: "İçerik hazır", Message: "Sayfa yayında", TargetPath: "/admin/pages"}}, "csrf").Render(context.Background(), out)
		}, []string{"Bildirimler", "İçerik hazır", "/admin/pages", "_csrf"}},
		{"translation", func(out *bytes.Buffer) error {
			return TranslationEditor(TranslationEditorData{SourceTitle: "Kaynak", SourceBody: "Metin", Locale: "en", Action: "/admin/pages/p/translate", CSRF: "csrf"}).Render(context.Background(), out)
		}, []string{"data-translation-source", "data-translation-target", "Kaynak içerik", "Çeviriyi kaydet"}},
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

func TestRevisionDiffEscapesUntrustedFieldValues(t *testing.T) {
	var out bytes.Buffer
	if err := RevisionDiffPanel([]adminstudio.FieldDiff{{Field: "title", Before: "<script>bad</script>", After: "Yeni"}}, "Editor", "bugün", "/admin/pages/p/revisions/1/restore", "csrf").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>bad") || !strings.Contains(out.String(), "&lt;script&gt;") {
		t.Fatal("revision diff did not escape saved content")
	}
}
