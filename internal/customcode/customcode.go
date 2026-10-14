// Package customcode validates and prepares operator-managed site integrations.
package customcode

import (
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

const MaxCodeBytes = 16 << 10

var (
	ErrInvalidCode = errors.New("invalid custom site code")
	urlLiteral     = regexp.MustCompile(`https://[A-Za-z0-9.-]+`)
)

type Source struct {
	Directive string
	Origin    string
}

var trustedHosts = map[string][]string{
	"www.googletagmanager.com":     {"script-src", "frame-src", "connect-src", "img-src"},
	"www.google-analytics.com":     {"connect-src", "img-src", "script-src"},
	"region1.google-analytics.com": {"connect-src", "img-src"},
	"www.google.com":               {"connect-src", "img-src"},
	"connect.facebook.net":         {"script-src"},
	"www.facebook.com":             {"connect-src", "img-src"},
	"static.hotjar.com":            {"script-src", "style-src"},
	"script.hotjar.com":            {"script-src"},
	"in.hotjar.com":                {"connect-src"},
	"vars.hotjar.com":              {"frame-src", "connect-src"},
	"static.zdassets.com":          {"script-src", "style-src"},
	"ekr.zdassets.com":             {"script-src"},
	"widget-mediator.zopim.com":    {"frame-src", "connect-src"},
}

// Validate checks both size and resource sources before code is saved.
func Validate(head, body string) ([]Source, error) {
	if len(head) > MaxCodeBytes || len(body) > MaxCodeBytes {
		return nil, ErrInvalidCode
	}
	sources := make(map[string]struct{})
	for _, code := range []string{head, body} {
		if err := validateFragment(code, sources); err != nil {
			return nil, ErrInvalidCode
		}
	}
	result := make([]Source, 0, len(sources))
	for host := range sources {
		for _, directive := range trustedHosts[host] {
			result = append(result, Source{Directive: directive, Origin: "https://" + host})
		}
	}
	return result, nil
}

func validateFragment(code string, sources map[string]struct{}) error {
	if code == "" {
		return nil
	}
	tokenizer := xhtml.NewTokenizer(strings.NewReader(code))
	inScript := false
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case xhtml.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return ErrInvalidCode
		case xhtml.TextToken:
			if inScript {
				for _, literal := range urlLiteral.FindAll(tokenizer.Raw(), -1) {
					parsed, err := url.Parse(string(literal))
					if err != nil || parsed.User != nil {
						return ErrInvalidCode
					}
					host := strings.ToLower(parsed.Hostname())
					if _, ok := trustedHosts[host]; !ok {
						return ErrInvalidCode
					}
					sources[host] = struct{}{}
				}
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			for _, attr := range token.Attr {
				key := strings.ToLower(attr.Key)
				if strings.HasPrefix(key, "on") {
					return ErrInvalidCode
				}
				if key != "src" && !(token.Data == "link" && key == "href") {
					continue
				}
				host, err := resourceHost(token.Data, attr.Val)
				if err != nil {
					return err
				}
				if host != "" {
					sources[host] = struct{}{}
				}
			}
			if token.Data == "object" || token.Data == "embed" || token.Data == "base" {
				return ErrInvalidCode
			}
			if token.Data == "script" {
				inScript = true
			}
		case xhtml.EndTagToken:
			if tokenizer.Token().Data == "script" {
				inScript = false
			}
		}
	}
}

func resourceHost(tag, value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", ErrInvalidCode
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		if strings.HasPrefix(value, "//") {
			return "", ErrInvalidCode
		}
		return "", nil
	}
	if parsed.Scheme != "https" || parsed.User != nil {
		return "", ErrInvalidCode
	}
	host := strings.ToLower(parsed.Hostname())
	allowed, known := trustedHosts[host]
	if !known {
		return "", ErrInvalidCode
	}
	directive := ""
	switch tag {
	case "script":
		directive = "script-src"
	case "iframe":
		directive = "frame-src"
	case "img":
		directive = "img-src"
	case "link":
		directive = "style-src"
	default:
		return "", ErrInvalidCode
	}
	for _, sourceDirective := range allowed {
		if directive == sourceDirective {
			return host, nil
		}
	}
	return "", ErrInvalidCode
}
