package forms

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func testForm() Form {
	return Form{ID: "contact", Title: "Contact", Fields: []Field{
		{ID: "email", Type: Email, Label: "Email", Required: true},
		{ID: "topic", Type: Select, Label: "Topic", Options: []string{"sales", "support"}},
	}}
}

func TestFormValidationAndSubmission(t *testing.T) {
	form := testForm()
	clean, validation, err := form.NormalizeSubmission(map[string]string{"email": " user@example.com ", "topic": "support"})
	if err != nil || validation != nil || clean["email"] != "user@example.com" {
		t.Fatalf("unexpected submission result: %#v %#v %v", clean, validation, err)
	}
	_, validation, err = form.NormalizeSubmission(map[string]string{"topic": "invalid"})
	if err != nil || validation["email"] != "required" || validation["topic"] != "invalid-option" {
		t.Fatalf("unexpected validation: %#v %v", validation, err)
	}
}

func TestFormRejectsDuplicateAndUnboundedFields(t *testing.T) {
	form := testForm()
	form.Fields = append(form.Fields, Field{ID: "email", Type: Text, Label: "Duplicate"})
	if err := form.Validate(); err == nil {
		t.Fatal("expected duplicate rejection")
	}
	form = testForm()
	form.Fields[1].Options = []string{"sales", "sales"}
	if err := form.Validate(); err == nil {
		t.Fatal("expected duplicate option rejection")
	}
}

func TestStableFields(t *testing.T) {
	got := StableFields([]Field{{ID: "z"}, {ID: "a"}})
	if got[0].ID != "a" || got[1].ID != "z" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestEmailAndAutoresponderValidation(t *testing.T) {
	form := testForm()
	if _, problems, err := form.NormalizeSubmission(map[string]string{"email": "person@gmai.com"}); err != nil || problems["email"] != "" {
		t.Fatalf("valid email was rejected: %#v/%v", problems, err)
	}
	if _, problems, err := form.NormalizeSubmission(map[string]string{"email": "not-an-email"}); err != nil || problems["email"] != "invalid-email" {
		t.Fatalf("invalid email was accepted: %#v/%v", problems, err)
	}
	form.Autoresponder = true
	form.Fields = []Field{{ID: "message", Type: Textarea, Label: "Message"}}
	if err := form.Validate(); err == nil {
		t.Fatal("autoresponder form without an email field was accepted")
	}
	form = testForm()
	form.NotificationRecipients = []string{"valid@example.test", "javascript:alert(1)"}
	if err := form.Validate(); err == nil {
		t.Fatal("invalid notification recipient was accepted")
	}
}

func TestPublicFormUsesNativeAccessibleControlsAndSpamFields(t *testing.T) {
	form := Form{ID: "contact", Title: "Contact", TurnstileEnabled: true, Fields: []Field{
		{ID: "email", Type: Email, Label: "Email", Required: true},
		{ID: "topic", Type: Select, Label: "Topic", Options: []string{"sales", "support"}},
	}}
	var output bytes.Buffer
	component := PublicForm(PublicFormProps{Form: form, Action: "javascript:alert(1)", CSRFToken: "csrf-token", SubmissionToken: "0123456789abcdef", TurnstileSiteKey: "site-key"})
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{`action="/forms/contact/submit"`, `data-form-view-endpoint="/forms/contact/view"`, `name="_csrf" value="csrf-token"`, `name="_submission_token" value="0123456789abcdef"`, `name="website_url"`, `type="email"`, `data-email-suggest`, `aria-live="polite"`, `data-sitekey="site-key"`, `challenges.cloudflare.com/turnstile/v0/api.js`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q from form: %s", want, got)
		}
	}
	if strings.Contains(got, "javascript:") {
		t.Fatalf("unsafe form action rendered: %s", got)
	}
}

func TestPolicyConsentRendersEscapedModalAndValidatesFieldContract(t *testing.T) {
	form := Form{ID: "contact", Title: "Contact", Fields: []Field{{ID: "terms", Type: Checkbox, Label: "I agree", Required: true, PolicyText: "Terms & conditions"}}}
	var output bytes.Buffer
	if err := PublicForm(PublicFormProps{Form: form}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-policy-open="hydro-policy-contact-terms"`, `<dialog`, `Terms &amp; conditions`, `data-policy-close`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("policy consent omitted %q: %s", want, output.String())
		}
	}
	form.Fields[0].Type = Text
	if err := form.Validate(); err == nil {
		t.Fatal("policy text on a non-consent field was accepted")
	}
}

func TestPublicFormOptInConversionMetadataAndRejectsMalformedSettings(t *testing.T) {
	form := Form{ID: "contact", Title: "Contact", ConversionTracking: true, GoogleAdsConversionID: "AW-12345", GoogleAdsConversionLabel: "contact-lead", Fields: []Field{{ID: "email", Type: Email, Label: "Email"}}}
	var output bytes.Buffer
	if err := PublicForm(PublicFormProps{Form: form}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-conversion-enabled`, `data-google-ads-conversion-id="AW-12345"`, `data-google-ads-conversion-label="contact-lead"`, `data-meta-pixel-event="Lead"`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("rendered form omitted %q: %s", want, output.String())
		}
	}
	form.GoogleAdsConversionID = "AW-123\" onmouseover=\"alert(1)"
	if err := form.Validate(); err == nil {
		t.Fatal("malformed conversion ID accepted")
	}
	form.GoogleAdsConversionID = ""
	form.ConversionTracking = false
	if err := form.Validate(); err == nil {
		t.Fatal("conversion destination accepted while tracking is disabled")
	}
}

func TestConversionEventsAreOptInAndOnlySentAfterSuccessfulHTMXResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/forms/static/forms.js", nil)
	request.SetPathValue("path", "forms.js")
	ServeAsset(recorder, request)
	script := recorder.Body.String()
	for _, want := range []string{"htmx:afterRequest", "event.detail?.successful === true", "statusCode < 300", "data-conversion-enabled", "window.gtag", "window.fbq", "data-conversion-sent"} {
		if want == "data-conversion-sent" {
			want = "conversionSent"
		}
		if !strings.Contains(script, want) {
			t.Errorf("forms script omitted conversion guard %q", want)
		}
	}
}

func TestFormsAssetsAreAllowlisted(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.SetPathValue("path", "forms.js")
	response := httptest.NewRecorder()
	ServeAsset(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "gmail.com") {
		t.Fatalf("forms script response %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest("GET", "/", nil)
	request.SetPathValue("path", "../../secret")
	response = httptest.NewRecorder()
	ServeAsset(response, request)
	if response.Code != 404 {
		t.Fatalf("unexpected asset path status %d", response.Code)
	}
}
