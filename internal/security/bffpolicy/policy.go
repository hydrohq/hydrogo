// Package bffpolicy contains the narrow request boundary a separate web BFF
// must apply before calling HydroGo. It intentionally does not implement a
// session store or token exchange; those belong to the deploying frontend.
package bffpolicy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

var (
	ErrInvalidCoreOrigin = errors.New("invalid core origin")
	ErrDisallowedPath    = errors.New("bff path is not a versioned api route")
	ErrBrowserCredential = errors.New("browser authorization cannot cross the bff boundary")
	ErrMissingCoreCookie = errors.New("server-side core session is required")
)

// BuildCoreRequest maps a BFF request to the configured Core origin. The
// caller must resolve coreCookie from a server-side BFF session; a browser
// Cookie header is never trusted or forwarded. Only /api/v1 routes may cross
// this boundary.
func BuildCoreRequest(in *http.Request, coreOrigin *url.URL, coreCookie string) (*http.Request, error) {
	if in == nil || coreOrigin == nil || (coreOrigin.Scheme != "http" && coreOrigin.Scheme != "https") || coreOrigin.Host == "" {
		return nil, ErrInvalidCoreOrigin
	}
	if !allowedPath(in.URL.Path) {
		return nil, ErrDisallowedPath
	}
	if strings.TrimSpace(in.Header.Get("Authorization")) != "" || strings.TrimSpace(in.Header.Get("Proxy-Authorization")) != "" {
		return nil, ErrBrowserCredential
	}
	if strings.TrimSpace(coreCookie) == "" {
		return nil, ErrMissingCoreCookie
	}

	out := in.Clone(in.Context())
	out.URL = &url.URL{Scheme: coreOrigin.Scheme, Host: coreOrigin.Host, Path: in.URL.Path, RawPath: in.URL.RawPath, RawQuery: in.URL.RawQuery}
	out.Host = coreOrigin.Host
	out.RequestURI = ""
	out.Header = in.Header.Clone()
	for _, name := range []string{
		"Authorization", "Proxy-Authorization", "Cookie", "Origin", "Referer",
		"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto",
		"Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-Dest",
	} {
		out.Header.Del(name)
	}
	out.Header.Set("Cookie", coreCookie)
	return out, nil
}

func allowedPath(value string) bool {
	if value != "/api/v1" && !strings.HasPrefix(value, "/api/v1/") {
		return false
	}
	if value == "" || strings.Contains(value, "\\") {
		return false
	}
	clean := path.Clean(value)
	return clean == value && !strings.HasPrefix(clean, "/api/v1/../")
}

// ValidateCoreOrigin is useful at BFF startup so a deployment fails before
// serving traffic rather than accepting an attacker-controlled upstream URL.
func ValidateCoreOrigin(origin string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: origin must be an absolute HTTP(S) URL without path or credentials", ErrInvalidCoreOrigin)
	}
	parsed.Path = ""
	return parsed, nil
}
