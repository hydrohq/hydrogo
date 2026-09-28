package media

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSanitizeSVGKeepsVectorDataAndStripsEventAttributes(t *testing.T) {
	input := `<svg viewBox="0 0 10 10"><title>Logo</title><path d="M0 0L10 10" fill="#123456" onload="alert(1)"/></svg>`
	var output bytes.Buffer
	if err := SanitizeSVG(strings.NewReader(input), &output, 4096); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{`<svg`, `<title>Logo</title>`, `<path`, `d="M0 0L10 10"`, `fill="#123456"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("sanitized SVG missing %q: %s", want, got)
		}
	}
	for _, forbidden := range []string{"onload", "alert", "script", "foreignObject"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(forbidden)) {
			t.Fatalf("unsafe SVG content retained %q: %s", forbidden, got)
		}
	}
}

func TestSanitizeSVGRejectsActiveContentAndExternalReferences(t *testing.T) {
	for _, input := range []string{
		`<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg xmlns="http://www.w3.org/2000/svg"><text>&x;</text></svg>`,
		`<svg><script>alert(1)</script></svg>`,
		`<svg><use href="https://evil.example/payload.svg#x"/></svg>`,
		`<svg><path fill="url(https://evil.example/x)"/></svg>`,
	} {
		if err := SanitizeSVG(strings.NewReader(input), &bytes.Buffer{}, 4096); !errors.Is(err, ErrInvalidSVG) {
			t.Fatalf("SanitizeSVG(%q) error = %v", input, err)
		}
	}
}

func TestSanitizeSVGEnforcesInputLimitAndRoot(t *testing.T) {
	if err := SanitizeSVG(strings.NewReader(`<svg/>`), &bytes.Buffer{}, 4); !errors.Is(err, ErrInvalidSVG) {
		t.Fatalf("oversize error = %v", err)
	}
	if err := SanitizeSVG(strings.NewReader(`<g/>`), &bytes.Buffer{}, 4096); !errors.Is(err, ErrInvalidSVG) {
		t.Fatalf("non-SVG root error = %v", err)
	}
}

func TestUploaderSanitizesSVGBeforeImmutableStorage(t *testing.T) {
	dir := t.TempDir()
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	result, err := uploader.Save(strings.NewReader(`<svg viewBox="0 0 2 2"><path d="M0 0" onclick="alert(1)"/></svg>`), "logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	if result.MimeType != "image/svg+xml" || !strings.HasSuffix(result.StoragePath, ".svg") {
		t.Fatalf("SVG result = %#v", result)
	}
	data, err := os.ReadFile(result.StoragePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`<path d="M0 0"`)) || bytes.Contains(data, []byte("onclick")) {
		t.Fatalf("stored SVG not sanitized: %s", data)
	}
}

func TestUploaderRejectsMaliciousSVGBeforeStorage(t *testing.T) {
	dir := t.TempDir()
	uploader, err := NewUploader(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uploader.Save(strings.NewReader(`<svg><script>alert(1)</script></svg>`), "logo.svg"); !errors.Is(err, ErrInvalidSVG) {
		t.Fatalf("malicious SVG error = %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected SVG left files behind: %v", entries)
	}
}
