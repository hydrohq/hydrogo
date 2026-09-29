package designsystem

import (
	"strings"
	"testing"
)

func TestSectoralHelpersHandleEscapingBoundsAndVariants(t *testing.T) {
	if got := ReadingTime("<p>one &amp; two</p>"); got != 1 {
		t.Fatalf("ReadingTime=%d", got)
	}
	if got := SmartTypography("A...  B -- C"); got != "A… B — C" {
		t.Fatalf("SmartTypography=%q", got)
	}
	if cookieVariant("unknown") != "bar" || cookieVariant("pill") != "pill" {
		t.Fatal("cookie variant was not bounded")
	}
	if safeConsentAction("//evil.example") != "" || safeConsentAction("/privacy/consent") != "/privacy/consent" {
		t.Fatal("consent action was not same-origin")
	}
	got := render(t, ResponsiveDataTable(DataTableProps{Caption: "Records", Columns: []string{"Name", "Status"}, Rows: [][]string{{"A"}}}))
	if !strings.Contains(got, "<td></td>") || !strings.Contains(got, "scope=\"col\"") {
		t.Fatalf("responsive table did not pad or label cells: %s", got)
	}
	got = render(t, CookieNotice(CookieNoticeProps{Heading: "Privacy", Variant: "pill", Action: "/consent", CSRF: "token", ReturnTo: "/about"}))
	if !strings.Contains(got, "ds-cookie-notice--pill") || !strings.Contains(got, `name="decision" value="rejected"`) || !strings.Contains(got, `name="decision" value="granted"`) || !strings.Contains(got, `name="return_to" value="/about"`) || !strings.Contains(got, "/consent") {
		t.Fatalf("cookie notice = %s", got)
	}
	got = render(t, SocialPhotoGrid(SocialPhotoGridProps{Label: "Community", Photos: []SocialPhoto{{Image: "javascript:alert(1)", Alt: "bad"}, {Image: "/safe.webp", Alt: "safe"}}}))
	if strings.Contains(got, "javascript:") || !strings.Contains(got, "safe.webp") {
		t.Fatalf("social grid URL filtering = %s", got)
	}
}
