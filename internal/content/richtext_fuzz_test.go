package content

import "testing"

func FuzzValidateRichText(f *testing.F) {
	for _, seed := range []string{
		"plain text",
		`{"ops":[{"insert":"hello"}]}`,
		`<script>alert(1)</script><p>safe</p>`,
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_ = ValidateRichText(value)
	})
}
