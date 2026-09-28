package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPictureAndVideoEmbedRendering(t *testing.T) {
	var out bytes.Buffer
	if err := Picture(PictureProps{Src: "/media/asset", Alt: "Photo", BlurHash: "LEHV6nWB2yk8pyo0adR*.7kCMdnj", FocalX: .2, FocalY: .8}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `object-position: 20.00% 80.00%`) || !strings.Contains(out.String(), `data-blurhash="LEHV6nWB2yk8pyo0adR*.7kCMdnj"`) {
		t.Fatalf("picture metadata missing: %s", out.String())
	}
	out.Reset()
	if err := PictureVideoEmbed("https://youtu.be/abc123", "Video").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "youtube-nocookie.com/embed/abc123") || strings.Contains(out.String(), "<script") {
		t.Fatalf("unsafe video embed: %s", out.String())
	}
	out.Reset()
	if err := PictureVideoEmbed("https://example.test/video", "Video").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("untrusted video rendered: %s", out.String())
	}
}

func TestMediaAdminComponentsEscapeAndExposeControls(t *testing.T) {
	var out bytes.Buffer
	items := []MediaFolderItem{{ID: "f1", Name: "News", Href: "/admin/media?folder=f1", Depth: 1}}
	if err := MediaFolderTree(items, "f1").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "aria-current=\"page\"") || !strings.Contains(out.String(), "padding-inline-start: 2em") {
		t.Fatalf("folder tree lacks current nested item: %s", out.String())
	}
	out.Reset()
	form := MediaAssetMetadataForm(MediaMetadataFormProps{Action: "/admin/media/id/metadata", CSRF: "token", Folders: items, FocalX: "0.5", FocalY: "0.5"})
	if err := form.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `name="focal_x"`) || !strings.Contains(out.String(), `name="tags"`) || !strings.Contains(out.String(), `name="_csrf"`) {
		t.Fatalf("metadata controls missing: %s", out.String())
	}
	out.Reset()
	settings := MediaSettingsForm(MediaSettingsFormProps{CSRF: "token", HotlinkProtection: true, WatermarkEnabled: true, WatermarkText: "© Brand", WatermarkPosition: "bottom-right", WatermarkOpacity: "60", WatermarkMargin: "12", WatermarkFolders: items, WatermarkFoldersSelected: []string{"f1"}, WatermarkLogos: []MediaLogoItem{{ID: "logo1", Name: "brand.png"}}, WatermarkLogoID: "logo1"})
	if err := settings.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hotlink_protection", "allowed_referer_hosts", "watermark_enabled", "watermark_logo_id", "watermark_folders", "watermark_opacity", "watermark_margin"} {
		if !strings.Contains(out.String(), `name="`+name+`"`) {
			t.Fatalf("settings control %q missing: %s", name, out.String())
		}
	}
	if !strings.Contains(out.String(), `value="logo1" selected>`) || !strings.Contains(out.String(), `value="f1" selected>`) {
		t.Fatalf("saved logo or folder selection was not retained: %s", out.String())
	}
}
