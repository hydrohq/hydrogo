package mailer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type brandedCapture struct{ message Message }

func (c *brandedCapture) Send(_ context.Context, message Message) error {
	c.message = message
	return nil
}

type brandingFake struct{ value EmailBranding }

func (f brandingFake) ReadEmailBranding(context.Context) (EmailBranding, error) { return f.value, nil }

func TestBrandedSenderAppliesCurrentSettingsToPlainAndHTMLMessages(t *testing.T) {
	capture := &brandedCapture{}
	sender := BrandedSender{
		Next: capture,
		Settings: brandingFake{value: EmailBranding{
			LegalName: "Example <Company>", PhysicalAddress: "1 Main Street", PrivacyPolicyURL: "https://example.test/privacy",
			FooterNotice: "Registered in Example", FooterText: "Manage preferences", LogoAssetID: "logo-1", PrimaryColor: "#123456",
		}},
		PublicOrigin: "https://example.test",
	}
	if err := sender.Send(context.Background(), Message{To: "admin@example.test", Subject: "Example", Body: "Message", HTMLBody: `<!doctype html><html><body><p>Message</p></body></html>`}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capture.message.Body, "1 Main Street") || !strings.Contains(capture.message.HTMLBody, "https://example.test/media/logo-1") || !strings.Contains(capture.message.HTMLBody, "#123456") || !strings.Contains(capture.message.HTMLBody, "Privacy policy") {
		t.Fatalf("branding missing from message: %+v", capture.message)
	}
	if strings.Contains(capture.message.HTMLBody, "<Company>") || !strings.Contains(capture.message.HTMLBody, "&lt;Company&gt;") {
		t.Fatalf("branding text not escaped: %s", capture.message.HTMLBody)
	}
}

func TestEmailBrandingRejectsUnsafeURLsLogoIDsAndColor(t *testing.T) {
	for _, value := range []EmailBranding{
		{PrivacyPolicyURL: "javascript:alert(1)"},
		{LogoAssetID: "../../secret"},
		{PrimaryColor: "red"},
	} {
		if _, err := value.Normalize(); err == nil {
			t.Errorf("unsafe branding accepted: %+v", value)
		}
	}
}

func TestBrandedSenderPropagatesSettingsReadFailure(t *testing.T) {
	settings := brandingReadError{}
	err := (BrandedSender{Next: &brandedCapture{}, Settings: settings}).Send(context.Background(), Message{To: "a@example.test", Subject: "test"})
	if !errors.Is(err, errBrandingRead) {
		t.Fatalf("read error = %v", err)
	}
}

type brandingReadError struct{}

func (brandingReadError) ReadEmailBranding(context.Context) (EmailBranding, error) {
	return EmailBranding{}, errBrandingRead
}

var errBrandingRead = errors.New("branding store unavailable")
