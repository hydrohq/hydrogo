package sdkdist

import "testing"

func TestValidateDownloadURL(t *testing.T) {
	if _, err := ValidateDownloadURL("https://artifacts.example.test/runtime", false); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateDownloadURL("http://127.0.0.1:8080/runtime", true); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"http://artifacts.example.test/runtime",
		"file:///tmp/runtime",
		"https://user:pass@artifacts.example.test/runtime",
		"https://artifacts.example.test/runtime#fragment",
		"https:///runtime",
	} {
		if _, err := ValidateDownloadURL(raw, false); err == nil {
			t.Fatalf("unsafe URL accepted: %q", raw)
		}
	}
}
