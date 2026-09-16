package v1

import "testing"

func TestFormContractNormalizesAndValidates(t *testing.T) {
	f := FormDefinition{ID: "contact", Title: "Contact", Fields: []FormField{{ID: "email", Type: FormEmail, Label: "Email", Required: true}, {ID: "topic", Type: FormSelect, Label: "Topic", Options: []string{"sales", "support"}}}}
	clean, problems, err := f.NormalizeFormSubmission(map[string]string{"email": " user@example.com ", "topic": "support"})
	if err != nil || problems != nil || clean["email"] != "user@example.com" {
		t.Fatalf("unexpected result: %#v %#v %v", clean, problems, err)
	}
	_, problems, err = f.NormalizeFormSubmission(map[string]string{"topic": "invalid"})
	if err != nil || problems["email"] != "required" || problems["topic"] != "invalid-option" {
		t.Fatalf("unexpected validation: %#v %v", problems, err)
	}
}

func TestStableFormFieldsCopiesInput(t *testing.T) {
	input := []FormField{{ID: "z"}, {ID: "a"}}
	got := StableFormFields(input)
	if got[0].ID != "a" || input[0].ID != "z" {
		t.Fatalf("unexpected sort/input: %#v %#v", got, input)
	}
}
