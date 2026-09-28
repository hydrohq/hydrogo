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
