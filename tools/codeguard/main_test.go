package main

import (
	"strings"
	"testing"
)

func TestHTMLStringsCountsMarkupLiteralsOnly(t *testing.T) {
	src := "package x\n" +
		"var a = `<div class=\"x\">hi</div>`\n" +
		"var b = \"<section>\"\n" +
		"var c = \"a < b && c > d\"\n" +
		"var d = \"<-chan int\"\n"
	count, err := htmlStrings("x.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestCompareRatchets(t *testing.T) {
	known := baseline{"legacy.go": {HTMLStrings: 3, Lines: 900}}
	tests := []struct {
		name    string
		current baseline
		want    string
	}{
		{"unchanged legacy file passes", baseline{"legacy.go": {HTMLStrings: 3, Lines: 900}}, ""},
		{"shrinking legacy file passes", baseline{"legacy.go": {HTMLStrings: 1, Lines: 850}}, ""},
		{"more HTML in legacy file fails", baseline{"legacy.go": {HTMLStrings: 4, Lines: 900}}, "grew from 3 to 4"},
		{"growing oversized file fails", baseline{"legacy.go": {HTMLStrings: 3, Lines: 901}}, "grew from 900 to 901"},
		{"new HTML file fails", baseline{"new.go": {HTMLStrings: 1}}, "contain HTML"},
		{"new oversized file fails", baseline{"new.go": {Lines: 801}}, "exceeds the 800-line limit"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problems := compare(known, test.current)
			got := strings.Join(problems, "\n")
			if test.want == "" && got != "" {
				t.Fatalf("unexpected problems: %s", got)
			}
			if test.want != "" && !strings.Contains(got, test.want) {
				t.Fatalf("problems %q do not mention %q", got, test.want)
			}
		})
	}
}
