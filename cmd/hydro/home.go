package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

func publicHomePath(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	return "/public/pages/" + url.PathEscape(locale) + "/demo/html"
}

func configuredLocale(locale string, supported []string) string {
	locale = strings.TrimSpace(locale)
	for _, candidate := range supported {
		if strings.EqualFold(locale, candidate) {
			return candidate
		}
	}
	return ""
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func requestedLocale(r *http.Request, supported []string, defaultLocale string) (string, bool) {
	if selected := configuredLocale(r.URL.Query().Get("locale"), supported); selected != "" {
		return selected, true
	}
	if selected := configuredLocale(cookieValue(r, "hydro_locale"), supported); selected != "" {
		return selected, false
	}
	return i18n.ResolveAcceptLanguage(r.Header.Get("Accept-Language"), supported, defaultLocale), false
}

func localeRedirect(w http.ResponseWriter, r *http.Request, locale string) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Vary", "Cookie")
	http.Redirect(w, r, "/"+url.PathEscape(locale)+"/", http.StatusFound)
}

func technicalHomeWithLocales(locale string, locales []string, demoPath string, demoEnabled bool) string {
	markup := technicalHomeHTML(locale, demoPath, demoEnabled)
	if len(locales) < 2 {
		return markup
	}
	var picker bytes.Buffer
	if err := designsystem.LocalePicker(designsystem.LocalePickerProps{Locales: locales, Selected: configuredLocale(locale, locales)}).Render(context.Background(), &picker); err != nil {
		return markup
	}
	return strings.Replace(markup, "</body>", picker.String()+"</body>", 1)
}

// technicalHomeHTML is intentionally small: the default root is a runtime
// status surface, not a marketing page. Product-specific frontends can still
// be configured through site.home_path or opened from the optional demo seed.
func technicalHomeHTML(locale, demoPath string, demoEnabled bool, nonce ...string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	styleNonceAttr := ""
	if len(nonce) > 0 && strings.TrimSpace(nonce[0]) != "" {
		styleNonceAttr = fmt.Sprintf(` nonce="%s"`, template.HTMLEscapeString(nonce[0]))
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Welcome to HydroGo</title>
  <style%s>
    :root {
      --bg: #090a0f;
      --card: #0d1017;
      --card-raised: #121620;
      --border: #1e2533;
      --border-focus: #0ea5e9;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --primary: #0ea5e9;
      --primary-hover: #38bdf8;
      --btn-secondary: #121620;
      --btn-secondary-hover: #181f2c;
      --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: var(--font-sans);
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px;
      line-height: 1.5;
      -webkit-font-smoothing: antialiased;
      text-align: center;
      position: relative;
      overflow: hidden;
    }
    .hero-glow-sphere {
      position: absolute;
      top: 20%%;
      left: 50%%;
      transform: translate(-50%%, -50%%);
      width: 500px;
      height: 500px;
      background: radial-gradient(circle, rgba(14, 165, 233, 0.12) 0%%, transparent 65%%);
      border-radius: 50%%;
      filter: blur(60px);
      pointer-events: none;
      z-index: 0;
    }
    .landing-wrapper {
      max-width: 640px;
      width: 100%%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      position: relative;
      z-index: 1;
      animation: fadeIn 0.4s ease;
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .brand-icon-box {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 72px;
      height: 72px;
      border-radius: 20px;
      background: linear-gradient(135deg, #0284c7 0%%, #0ea5e9 100%%);
      color: #ffffff;
      box-shadow: 0 8px 32px rgba(14, 165, 233, 0.35);
      margin-bottom: 28px;
    }
    .hero-title {
      font-size: clamp(2rem, 4.5vw, 2.75rem);
      font-weight: 600;
      letter-spacing: -0.015em;
      color: #ffffff;
      margin-bottom: 14px;
      line-height: 1.2;
    }
    .hero-subtitle {
      font-size: clamp(1rem, 2.5vw, 1.1rem);
      color: var(--text-muted);
      margin-bottom: 36px;
      font-weight: 400;
      max-width: 500px;
      line-height: 1.6;
    }
    .actions-group {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 14px;
      flex-wrap: wrap;
    }
    .btn-hero {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 12px 28px;
      font-size: 0.95rem;
      font-weight: 600;
      border-radius: 10px;
      text-decoration: none;
      transition: all 0.2s ease;
      cursor: pointer;
      min-width: 140px;
    }
    .btn-start {
      background: var(--primary);
      color: #ffffff;
      border: 1px solid var(--primary);
      box-shadow: 0 4px 16px rgba(14, 165, 233, 0.35);
    }
    .btn-start:hover {
      background: var(--primary-hover);
      border-color: var(--primary-hover);
      box-shadow: 0 6px 24px rgba(56, 189, 248, 0.5);
      transform: translateY(-1px);
    }
    .btn-docs {
      background: var(--btn-secondary);
      color: #ffffff;
      border: 1px solid var(--border);
    }
    .btn-docs:hover {
      background: var(--btn-secondary-hover);
      border-color: var(--border-focus);
      color: #ffffff;
      transform: translateY(-1px);
    }
  </style>
</head>
<body>
<div class="hero-glow-sphere"></div>
<main class="landing-wrapper">
  <div class="brand-icon-box">
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
  </div>

  <h1 class="hero-title">Welcome to HydroGo</h1>
  <p class="hero-subtitle">Manage your content, memberships, and publishing workflows from one fast, secure workspace.</p>

  <div class="actions-group">
    <a href="/admin" class="btn-hero btn-start">Get Started</a>
    <a href="/docs" class="btn-hero btn-docs">Documentation</a>
  </div>
</main>
</body>
</html>`, template.HTMLEscapeString(locale), styleNonceAttr)
}
