package designsystem

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, component templ.Component) string {
	t.Helper()
	var out bytes.Buffer
	if err := component.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestCoreBlocksRenderEscapedContentAndRejectUnsafeLinks(t *testing.T) {
	hero := render(t, HeroBlock(HeroProps{Title: `<Hello>`, MediaURL: `javascript:alert(1)`, ButtonText: "Go", ButtonURL: "/contact"}))
	if !strings.Contains(hero, "&lt;Hello&gt;") || !strings.Contains(hero, `href="/contact"`) || strings.Contains(hero, "javascript:") {
		t.Fatalf("hero content was not safely rendered: %s", hero)
	}
	cta := render(t, CTABlock(CTAProps{Title: "Contact", ButtonText: "Open", ButtonURL: "//evil.test"}))
	if !strings.Contains(cta, "Contact") || strings.Contains(cta, "evil.test") || strings.Contains(cta, "<a ") {
		t.Fatalf("CTA rendered unsafe link: %s", cta)
	}
	pricing := render(t, PricingTable(PricingTableProps{Items: []PricingItem{{Name: "Starter", PriceCents: 1299, Currency: "USD", Features: []string{"One site"}}}}))
	if !strings.Contains(pricing, "Starter") || !strings.Contains(pricing, "1299 USD") || !strings.Contains(pricing, "One site") {
		t.Fatalf("pricing table missed data: %s", pricing)
	}
	testimonials := render(t, Testimonials(TestimonialsProps{Items: []Testimonial{{Name: "Ari", Quote: "Works <well>"}}}))
	if !strings.Contains(testimonials, "Ari") || !strings.Contains(testimonials, "Works &lt;well&gt;") {
		t.Fatalf("testimonial content was not escaped: %s", testimonials)
	}
	if head := render(t, Head()); !strings.Contains(head, "designsystem.css") || !strings.Contains(head, `defer`) {
		t.Fatalf("design system assets missing: %s", head)
	}
}

func TestExportLinksRenderSafeTypedDownloads(t *testing.T) {
	got := render(t, ExportLinks([]ExportLink{{Label: "Excel", URL: "/admin/export.xlsx?q=ş"}, {Label: "Unsafe", URL: "javascript:alert(1)"}}))
	if !strings.Contains(got, `href="/admin/export.xlsx?q=ş"`) || !strings.Contains(got, `download`) || strings.Contains(got, "javascript:") || strings.Contains(got, "Unsafe") {
		t.Fatalf("export links were not safely rendered: %s", got)
	}
}

func TestContentViewSortOffersAccessibleSortLinks(t *testing.T) {
	got := render(t, ContentViewSort(ContentViewSortProps{Path: "/admin/pages", Status: "published", Selected: "-views"}))
	for _, want := range []string{`aria-label="Sort content by views"`, `href="/admin/pages?sort=-views&amp;status=published"`, `href="/admin/pages?sort=views&amp;status=published"`, `aria-current="page"`, "Most viewed", "Least viewed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sort control missing %q: %s", want, got)
		}
	}
	unsafe := render(t, ContentViewSort(ContentViewSortProps{Path: "//evil.test"}))
	if strings.Contains(unsafe, "evil.test") {
		t.Fatalf("unsafe sort target rendered: %s", unsafe)
	}
}

func TestDeviceViewportPreviewIsAccessibleAndSandboxed(t *testing.T) {
	got := render(t, DeviceViewportPreview(DeviceViewportPreviewProps{Endpoint: "/admin/pages/one%2Ftwo/preview-link", CSRF: "token<&"}))
	for _, want := range []string{`data-device-preview-open`, `data-device-preview-dialog`, `aria-labelledby=`, `data-device-size="mobile"`, `aria-pressed="true"`, `375 px`, `Masaüstü`, `sandbox=""`, `referrerpolicy="no-referrer"`, `data-device-preview-frame`} {
		if !strings.Contains(got, want) {
			t.Errorf("device preview missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "allow-same-origin") || strings.Contains(got, `csrf="token<&"`) {
		t.Fatalf("device preview must keep the iframe sandboxed and escape attributes: %s", got)
	}
}

func TestAnalyticsDateRangeUsesBoundedTypedGETForm(t *testing.T) {
	got := render(t, AnalyticsDateRange(AnalyticsDateRangeProps{From: "2026-09-01", To: "2026-09-29"}))
	for _, want := range []string{`method="get"`, `action="/admin/reports"`, `name="range" value="custom"`, `name="from" value="2026-09-01"`, `name="to" value="2026-09-29"`, `type="date"`} {
		if !strings.Contains(got, want) {
			t.Errorf("analytics date range missing %q: %s", want, got)
		}
	}
}

func TestViewCountBadgeFormatsAndLabelsCount(t *testing.T) {
	got := render(t, ViewCountBadge(ViewCountBadgeProps{Count: 1249}))
	if !strings.Contains(got, `aria-label="1249 views"`) || !strings.Contains(got, "1.2k") || !strings.Contains(got, "👁") {
		t.Fatalf("view count badge = %s", got)
	}
}

func TestSoundAlertPreferenceIsOptInAndAccessible(t *testing.T) {
	got := render(t, SoundAlertPreference(SoundAlertPreferenceProps{Title: "Notifications", Label: "Play a sound for new inquiries", Description: "This setting is stored in this browser."}))
	for _, want := range []string{`data-sound-alert-settings`, `data-sound-alert-preference`, `type="checkbox"`, "Play a sound for new inquiries"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sound preference missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, " checked") {
		t.Fatalf("sound alerts must default to off: %s", got)
	}
}

func TestInitialsAvatarIsEscapedAndPaletteIsStable(t *testing.T) {
	avatar := render(t, InitialsAvatar(InitialsAvatarProps{Initials: Initials("arda.ozyildirim"), Class: InitialsAvatarClass("arda@example.test")}))
	if !strings.Contains(avatar, `aria-label="AO"`) || !strings.Contains(avatar, `>AO</span>`) {
		t.Fatalf("avatar initials missing: %s", avatar)
	}
	if Initials("arda.ozyildirim") != "AO" || Initials("single") != "SI" || Initials(" ") != "?" {
		t.Fatal("initials extraction did not handle multi-part, single-part, or empty values")
	}
	if InitialsAvatarClass("arda@example.test") != InitialsAvatarClass("ARDA@example.test") {
		t.Fatal("avatar palette should be stable and case insensitive")
	}
}

func TestHeadInlinesCriticalTokensAndPreloadsRemainingStyles(t *testing.T) {
	html := render(t, Head())
	for _, want := range []string{`--background:#fff`, `rel="preload" as="style"`, `data-hydro-late-style`, `<noscript>`, `rel="stylesheet"`, `designsystem.css`} {
		if !strings.Contains(html, want) {
			t.Errorf("Head missing %q: %s", want, html)
		}
	}
}

func TestBackToTopIsAccessibleAndProgressivelyEnhanced(t *testing.T) {
	got := render(t, BackToTop("Page top"))
	for _, want := range []string{`type="button"`, `aria-label="Page top"`, `data-back-to-top`, `hidden`} {
		if !strings.Contains(got, want) {
			t.Errorf("back-to-top missing %q: %s", want, got)
		}
	}
}

func TestLightboxAudioAndShareComponentsValidateURLs(t *testing.T) {
	gallery := render(t, LightboxGallery(LightboxProps{ID: "gallery", Label: "Photos", Images: []GalleryImage{{Src: "/one.webp", Alt: "One", Caption: "First"}, {Src: "javascript:alert(1)", Alt: "Unsafe"}}}))
	for _, want := range []string{`data-lightbox-gallery`, `data-lightbox`, `data-lightbox-next`, `alt="One"`, `First`} {
		if !strings.Contains(gallery, want) {
			t.Errorf("gallery missing %q: %s", want, gallery)
		}
	}
	if strings.Contains(gallery, "javascript:") {
		t.Fatal("unsafe gallery image rendered")
	}
	audio := render(t, AudioPlayer(AudioProps{ID: "episode", Src: "/episode.mp3", Title: "Episode", TranscriptURL: "https://example.test/transcript"}))
	for _, want := range []string{`<audio controls`, `data-audio-speed`, `Read transcript`, `Episode`} {
		if !strings.Contains(audio, want) {
			t.Errorf("audio player missing %q: %s", want, audio)
		}
	}
	if got := render(t, AudioPlayer(AudioProps{Src: "javascript:alert(1)"})); got != "" {
		t.Fatalf("unsafe audio source rendered: %s", got)
	}
	share := render(t, ShareButtons(ShareProps{URL: "https://example.test/a?x=1&y=2", Title: "A title"}))
	for _, want := range []string{`aria-label="Share this page"`, `noopener noreferrer`, `https://www.linkedin.com/sharing/share-offsite/`, `data-copy-text=`} {
		if !strings.Contains(share, want) {
			t.Errorf("share controls missing %q: %s", want, share)
		}
	}
	if got := render(t, ShareButtons(ShareProps{URL: "javascript:alert(1)"})); got != "" {
		t.Fatalf("unsafe share URL rendered: %s", got)
	}
}

func TestBeforeAfterRender(t *testing.T) {
	got := render(t, BeforeAfter(BeforeAfterProps{BeforeSrc: "/before.webp", AfterSrc: "/after.webp", BeforeLabel: "Old", AfterLabel: "New", Position: 150}))
	for _, want := range []string{`type="range"`, `value="100"`, `aria-label="Old / New"`, `data-before-after`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(render(t, BeforeAfter(BeforeAfterProps{BeforeSrc: "javascript:alert(1)"})), "javascript:") {
		t.Fatal("unsafe image URL rendered")
	}
}

func TestAnimatedCounterRenderHasStaticFallbackAndTokenClasses(t *testing.T) {
	got := render(t, AnimatedCounter(CounterProps{Stats: []CounterStat{{Value: 2500, Suffix: "+", Label: "Projects"}}}))
	for _, want := range []string{`data-counter-value="2500"`, `>2500</span>`, `Projects`, `bg-background`, `text-foreground`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestFormSuccessConfettiIsOptIn(t *testing.T) {
	if got := render(t, FormSuccessConfetti(ConfettiProps{})); got != "" {
		t.Fatalf("disabled confetti rendered: %s", got)
	}
	if got := render(t, FormSuccessConfetti(ConfettiProps{Enabled: true})); !strings.Contains(got, `data-confetti-success`) || !strings.Contains(got, `aria-hidden="true"`) {
		t.Fatalf("confetti is not opt-in/hidden from AT: %s", got)
	}
}

func TestContactCTARendersSafePhoneAndWhatsAppLinks(t *testing.T) {
	got := render(t, ContactCTA(ContactCTAProps{Phone: "+1 (415) 555-0132", WhatsApp: "14155550132", Message: "Hello & welcome"}))
	if !strings.Contains(got, `href="tel:+14155550132"`) || !strings.Contains(got, `https://wa.me/14155550132?text=Hello+%26+welcome`) || !strings.Contains(got, `rel="noopener noreferrer"`) {
		t.Fatalf("contact links missing or unsafe: %s", got)
	}
	if got := render(t, ContactCTA(ContactCTAProps{Phone: "javascript:12345678"})); strings.Contains(got, "tel:") {
		t.Fatalf("invalid phone rendered: %s", got)
	}
}

func TestFloatingContactUsesOnlySafeConfiguredDestinations(t *testing.T) {
	got := render(t, FloatingContact(FloatingContactProps{Phone: "+1 (415) 555-0132", WhatsAppURL: "https://wa.me/14155550132"}))
	for _, want := range []string{`data-floating-contact`, `href="tel:+14155550132"`, `href="https://wa.me/14155550132"`, `rel="noopener noreferrer"`} {
		if !strings.Contains(got, want) {
			t.Errorf("floating contact missing %q: %s", want, got)
		}
	}
	unsafe := render(t, FloatingContact(FloatingContactProps{WhatsAppURL: "https://evil.example/"}))
	if strings.Contains(unsafe, `data-floating-contact`) || strings.Contains(unsafe, `evil.example`) {
		t.Fatalf("unsafe WhatsApp destination rendered: %s", unsafe)
	}
}

func TestFAQAccordionRendersAccessibleDetailsAndSchema(t *testing.T) {
	got := render(t, FAQAccordion(FAQProps{ID: "help", Heading: "Help", Items: []FAQItem{{Question: "Where?", Answer: `At <home> </script>`}}}))
	for _, want := range []string{`<details`, `<summary>Where?</summary>`, `application/ld+json`, `"@type":"FAQPage"`, `&lt;home&gt;`, `\u003c/script\u003e`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestCopyButtonEscapesAndIsKeyboardNative(t *testing.T) {
	got := render(t, CopyButton(CopyButtonProps{Text: `<secret>"`, Label: "Copy value", CopiedLabel: "Done"}))
	for _, want := range []string{`<button`, `type="button"`, `aria-label="Copy value"`, `data-copy-text="&lt;secret&gt;&#34;"`, `Done`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestStickyHeaderRendersTypedThresholdAndChildren(t *testing.T) {
	component := StickyHeader(StickyHeaderProps{ID: "site-header", Threshold: 80})
	ctx := templ.WithChildren(context.Background(), templ.Raw("Navigation"))
	var out bytes.Buffer
	if err := component.Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `data-threshold="80"`) || !strings.Contains(out.String(), "Navigation") {
		t.Fatalf("header missing threshold/slot: %s", out.String())
	}
}

func TestVideoPlayerHasNativeFallbackControlsAndCaptionTrack(t *testing.T) {
	got := render(t, VideoPlayer(VideoProps{ID: "video-1", Src: "/video.mp4", Tracks: []CaptionTrack{{Src: "/captions.vtt", Language: "en", Label: "English"}}}))
	for _, want := range []string{`controls`, `data-video-speed`, `data-video-pip`, `kind="captions"`, `srclang="en"`, `ArrowLeft`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestMasonryGridRejectsUnsafeLinkAndRendersLazyImages(t *testing.T) {
	got := render(t, MasonryGrid(MasonryProps{Columns: 3, Items: []MasonryItem{{Image: "/a.webp", Alt: "Room", Link: "javascript:alert(1)"}}}))
	if strings.Contains(got, "javascript:") || strings.Contains(got, "<a ") || !strings.Contains(got, `loading="lazy"`) || !strings.Contains(got, `--ds-columns: 3`) {
		t.Fatalf("unsafe or incomplete masonry: %s", got)
	}
}

func TestImageHotspotsKeepListFallbackAndClampInlineCoordinates(t *testing.T) {
	got := render(t, ImageHotspots(HotspotsProps{Image: "/room.webp", Points: []Hotspot{{X: 150, Y: -1, Title: "Window", Description: "View"}}}))
	for _, want := range []string{`style="left:100.00%;top:0.00%;"`, `<ol`, `Window`, `Show details: Window`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestAssetsExposeOnlyAllowlistedFiles(t *testing.T) {
	for name, want := range map[string]string{"designsystem.css": "--background", "widgets.js": "hydro:form-success"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/", nil)
		request.SetPathValue("path", name)
		ServeAsset(recorder, request)
		if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("asset %s: status=%d", name, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	request.SetPathValue("path", "../go.mod")
	ServeAsset(recorder, request)
	if recorder.Code != 404 {
		t.Fatalf("unexpected asset path status %d", recorder.Code)
	}
}
