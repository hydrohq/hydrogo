package forms

import (
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
