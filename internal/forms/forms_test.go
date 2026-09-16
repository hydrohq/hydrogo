package forms

import "testing"

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
