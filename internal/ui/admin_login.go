package ui

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// AdminLoginCSS contains refined dark and light mode styling for the HydroGo login screen.
const AdminLoginCSS = `
:root {
	--bg-canvas: #090a0f;
	--bg-panel: #0d1017;
	--bg-card: #121620;
	--bg-input: #121620;
	--bg-input-focus: #171d2b;

	--border-subtle: #1b2230;
	--border-light: #263044;
	--border-focus: #0ea5e9;

	--text-primary: #f8fafc;
	--text-secondary: #cbd5e1;
	--text-muted: #94a3b8;

	--brand-primary: #0ea5e9;
	--brand-accent: #38bdf8;
	--brand-glow: rgba(14, 165, 233, 0.22);

	--btn-primary-bg: #0ea5e9;
	--btn-primary-hover: #38bdf8;
	--btn-primary-text: #ffffff;

	--btn-secondary-bg: #121620;
	--btn-secondary-hover: #171d2b;
	--btn-secondary-border: #1b2230;
	--btn-secondary-text: #cbd5e1;

	--color-success: #10b981;
	--color-error: #ef4444;

	--radius-sm: 8px;
	--radius-md: 12px;
	--radius-lg: 16px;
	--radius-xl: 24px;

	--font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
	--font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

html[data-theme="light"] {
	--bg-canvas: #f8fafc;
	--bg-panel: #ffffff;
	--bg-card: #ffffff;
	--bg-input: #f1f5f9;
	--bg-input-focus: #ffffff;

	--border-subtle: #e2e8f0;
	--border-light: #cbd5e1;
	--border-focus: #0284c7;

	--text-primary: #0f172a;
	--text-secondary: #334155;
	--text-muted: #64748b;

	--brand-primary: #0284c7;
	--brand-accent: #0284c7;
	--brand-glow: rgba(2, 132, 199, 0.16);

	--btn-primary-bg: #0284c7;
	--btn-primary-hover: #0369a1;
	--btn-primary-text: #ffffff;

	--btn-secondary-bg: #ffffff;
	--btn-secondary-hover: #f1f5f9;
	--btn-secondary-border: #cbd5e1;
	--btn-secondary-text: #0f172a;
}

html[data-theme="dark"] {
	--bg-canvas: #090a0f;
	--bg-panel: #0d1017;
	--bg-card: #121620;
	--bg-input: #121620;
	--bg-input-focus: #171d2b;

	--border-subtle: #1b2230;
	--border-light: #263044;
	--border-focus: #0ea5e9;

	--text-primary: #f8fafc;
	--text-secondary: #cbd5e1;
	--text-muted: #94a3b8;

	--brand-primary: #0ea5e9;
	--brand-accent: #38bdf8;
	--brand-glow: rgba(14, 165, 233, 0.22);

	--btn-primary-bg: #0ea5e9;
	--btn-primary-hover: #38bdf8;
	--btn-primary-text: #ffffff;

	--btn-secondary-bg: #121620;
	--btn-secondary-hover: #171d2b;
	--btn-secondary-border: #1b2230;
	--btn-secondary-text: #cbd5e1;
}

*, *::before, *::after {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body.login-body {
	min-height: 100vh;
	display: flex;
	background: var(--bg-canvas);
	color: var(--text-primary);
	font-family: var(--font-sans);
	font-size: 14px;
	line-height: 1.5;
	overflow-x: hidden;
	-webkit-font-smoothing: antialiased;
}

.login-layout {
	display: flex;
	width: 100vw;
	min-height: 100vh;
}

/* LEFT BRAND SHOWCASE PANEL */
.login-hero-pane {
	position: relative;
	flex: 1.1;
	background: radial-gradient(circle at 12% 24%, rgba(14, 165, 233, 0.2) 0%, transparent 46%),
	            linear-gradient(145deg, #0c1b2d 0%, #0b111c 68%);
	padding: 56px 64px;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	border-right: 1px solid var(--border-subtle);
	overflow: hidden;
}
.login-hero-pane::after {
	content: "";
	position: absolute;
	right: -145px;
	bottom: -260px;
	width: 610px;
	height: 610px;
	border: 1px solid rgba(56, 189, 248, .13);
	border-radius: 50%;
	box-shadow: 0 0 0 96px rgba(56,189,248,.025), 0 0 0 192px rgba(56,189,248,.018);
	pointer-events: none;
}

html[data-theme="light"] .login-hero-pane {
	background: radial-gradient(circle at 20% 20%, rgba(2, 132, 199, 0.08) 0%, transparent 50%),
	            #f1f5f9;
	border-right-color: var(--border-subtle);
}

.hero-glow-sphere {
	position: absolute;
	top: 15%;
	left: 10%;
	width: 420px;
	height: 420px;
	background: radial-gradient(circle, rgba(14, 165, 233, 0.08) 0%, transparent 70%);
	border-radius: 50%;
	filter: blur(54px);
	pointer-events: none;
}

.hero-header {
	position: relative;
	z-index: 2;
}

.brand-badge {
	display: inline-flex;
	align-items: center;
	gap: 12px;
	text-decoration: none;
	color: var(--text-primary);
}

.brand-icon-box {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 38px;
	height: 38px;
	border-radius: var(--radius-md);
	background: linear-gradient(135deg, #0284c7 0%, #0ea5e9 100%);
	color: #ffffff;
	box-shadow: 0 4px 14px rgba(14, 165, 233, 0.35);
}

.brand-text-wrap {
	display: flex;
	flex-direction: column;
}

.brand-name {
	font-size: 1.15rem;
	font-weight: 800;
	letter-spacing: -0.02em;
	color: var(--text-primary);
}

.brand-tag {
	font-family: var(--font-mono);
	font-size: 0.65rem;
	font-weight: 700;
	letter-spacing: 0.06em;
	text-transform: uppercase;
	color: var(--brand-accent);
}

.hero-content {
	position: relative;
	z-index: 2;
	max-width: 500px;
	margin-top: auto;
	margin-bottom: auto;
	padding: 40px 0;
}

.hero-headline {
	font-size: 2.3rem;
	font-weight: 600;
	letter-spacing: -0.015em;
	line-height: 1.25;
	color: var(--text-primary);
	margin-bottom: 16px;
}

.hero-headline span.gradient-text {
	background: linear-gradient(135deg, #38bdf8 0%, #818cf8 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
}

html[data-theme="light"] .hero-headline span.gradient-text {
	background: linear-gradient(135deg, #0284c7 0%, #4f46e5 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
}

.hero-description {
	font-size: 1rem;
	line-height: 1.6;
	color: var(--text-secondary);
}
.hero-eyebrow {
	margin-bottom: 20px;
	color: #7dd3fc;
	font-family: var(--font-mono);
	font-size: .7rem;
	font-weight: 700;
	letter-spacing: .15em;
	text-transform: uppercase;
}
.hero-links { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 32px; }
.hero-links a { display: inline-flex; align-items: center; min-height: 40px; padding: 0 15px; border: 1px solid rgba(125,211,252,.25); border-radius: 9px; background: rgba(15,30,48,.55); color: #dbeafe; font-size: .82rem; font-weight: 600; text-decoration: none; transition: background .15s, border-color .15s; }
.hero-links a:hover { background: rgba(14,165,233,.18); border-color: #38bdf8; }

/* RIGHT AUTH FORM PANE */
.login-form-pane {
	flex: 0.9;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	padding: 48px 64px;
	background: var(--bg-canvas);
	position: relative;
}

.pane-top-actions {
	display: flex;
	justify-content: flex-end;
	align-items: center;
	gap: 12px;
}
.login-language-switch { display: inline-flex; padding: 3px; border: 1px solid var(--border-light); border-radius: 10px; background: var(--bg-card); }
.login-language-switch a { display: inline-flex; align-items: center; justify-content: center; min-width: 42px; min-height: 34px; border-radius: 7px; color: var(--text-muted); font-size: .75rem; font-weight: 700; text-decoration: none; }
.login-language-switch a[aria-current="page"] { background: var(--brand-primary); color: #06111c; }
.login-language-switch a:hover:not([aria-current="page"]) { color: var(--text-primary); }
.login-language-switch a:focus-visible,.hero-links a:focus-visible { outline: 2px solid #38bdf8; outline-offset: 3px; }

.theme-switch-btn {
	background: var(--bg-card);
	border: 1px solid var(--border-subtle);
	color: var(--text-secondary);
	padding: 6px 14px;
	border-radius: var(--radius-sm);
	font-size: 0.76rem;
	font-family: var(--font-sans);
	font-weight: 600;
	cursor: pointer;
	display: inline-flex;
	align-items: center;
	gap: 7px;
	transition: all 0.15s ease;
}

.theme-switch-btn:hover {
	background: var(--bg-input-focus);
	color: var(--text-primary);
	border-color: var(--border-light);
}

.form-container {
	max-width: 360px;
	width: 100%;
	margin: auto;
}

.form-header {
	margin-bottom: 28px;
}

.form-title {
	font-size: 1.75rem;
	font-weight: 700;
	letter-spacing: -0.025em;
	color: var(--text-primary);
	margin-bottom: 6px;
}

.form-subtitle {
	font-size: 0.88rem;
	color: var(--text-muted);
}

.auth-form {
	display: flex;
	flex-direction: column;
	gap: 18px;
}

.input-group {
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.input-label-row {
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.input-label {
	font-size: 0.82rem;
	font-weight: 600;
	color: var(--text-secondary);
}

.input-wrapper {
	position: relative;
	display: flex;
	align-items: center;
}

.input-icon {
	position: absolute;
	left: 14px;
	color: var(--text-muted);
	pointer-events: none;
	display: grid;
	place-items: center;
}

.auth-input {
	width: 100%;
	height: 44px;
	padding: 0 14px 0 42px;
	background: var(--bg-input);
	border: 1px solid var(--border-subtle);
	border-radius: var(--radius-md);
	color: var(--text-primary);
	font-family: inherit;
	font-size: 0.9rem;
	outline: none;
	transition: border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.auth-input::placeholder {
	color: var(--text-muted);
	opacity: 0.6;
}

.auth-input:hover {
	border-color: var(--border-light);
}

.auth-input:focus {
	background: var(--bg-input-focus);
	border-color: var(--border-focus);
	box-shadow: 0 0 0 3px var(--brand-glow);
}

.form-row-utility {
	display: flex;
	align-items: center;
	justify-content: space-between;
	font-size: 0.82rem;
}

.checkbox-label {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	color: var(--text-secondary);
	cursor: pointer;
	user-select: none;
}

.checkbox-custom {
	width: 16px;
	height: 16px;
	border: 1px solid var(--border-light);
	border-radius: 4px;
	background: var(--bg-input);
	display: grid;
	place-items: center;
	transition: all 0.15s ease;
}

.checkbox-custom svg {
	display: none;
}
.checkbox-label input[type="checkbox"] { position: absolute; width: 1px; height: 1px; opacity: 0; }
.checkbox-label input[type="checkbox"]:focus-visible + .checkbox-custom { outline: 2px solid var(--brand-accent); outline-offset: 3px; }

input[type="checkbox"]:checked + .checkbox-custom {
	background: var(--brand-primary);
	border-color: var(--brand-primary);
	color: #ffffff;
}

input[type="checkbox"]:checked + .checkbox-custom svg {
	display: block;
}

.link-subtle {
	color: var(--brand-accent);
	text-decoration: none;
	font-size: 0.82rem;
	transition: color 0.15s ease;
}

.link-subtle:hover {
	text-decoration: underline;
}

.btn-submit {
	height: 44px;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: 8px;
	background: var(--btn-primary-bg);
	color: var(--btn-primary-text);
	border: none;
	border-radius: var(--radius-md);
	font-family: inherit;
	font-size: 0.9rem;
	font-weight: 600;
	cursor: pointer;
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
	transition: background 0.15s ease, transform 0.1s ease, box-shadow 0.15s ease;
}

.btn-submit:hover {
	background: var(--btn-primary-hover);
	box-shadow: 0 4px 14px var(--brand-glow);
}

.btn-submit:active {
	transform: scale(0.99);
}

.btn-submit:disabled {
	opacity: 0.65;
	cursor: not-allowed;
	transform: none;
}

.form-alert {
	display: none;
	padding: 10px 14px;
	border-radius: var(--radius-sm);
	font-size: 0.82rem;
	line-height: 1.4;
	font-weight: 500;
}

.form-alert.alert-error {
	display: block;
	background: rgba(239, 68, 68, 0.12);
	border: 1px solid rgba(239, 68, 68, 0.3);
	color: var(--color-error);
}

.form-alert.alert-success {
	display: block;
	background: rgba(16, 185, 129, 0.12);
	border: 1px solid rgba(16, 185, 129, 0.3);
	color: var(--color-success);
}
.form-alert.alert-warning { border-color:rgba(217,119,6,.4);background:rgba(245,158,11,.12);color:#92400e }

.form-pane-footer {
	text-align: center;
	font-size: 0.78rem;
	color: var(--text-muted);
}

/* RESPONSIVE ADAPTATIONS */
@media (max-width: 900px) {
	.login-layout {
		flex-direction: column;
	}
	.login-hero-pane {
		display: none;
	}
	.login-form-pane {
		flex: 1;
		padding: 36px 24px;
	}
}
@media (max-width: 480px) {
	.login-form-pane { padding: 24px; }
	.pane-top-actions { justify-content: space-between; }
}
@media (prefers-reduced-motion: reduce) { *,*::before,*::after { transition-duration: .01ms !important; animation-duration: .01ms !important; } }
`

// RenderAdminLogin renders the clean high-performance login screen for HydroGo.
func RenderAdminLogin(w io.Writer, csrfToken, returnTo string) error {
	return RenderAdminLoginWithAppearance(w, csrfToken, returnTo, site.Appearance{})
}

// RenderAdminLoginWithAppearance applies validated site appearance to the public admin login screen.
func RenderAdminLoginWithAppearance(w io.Writer, csrfToken, returnTo string, appearance site.Appearance) error {
	return RenderAdminLoginWithLocaleAndAppearance(w, csrfToken, returnTo, "tr", appearance)
}

// RenderAdminLoginWithLocaleAndAppearance renders the public login in a bounded language.
func RenderAdminLoginWithLocaleAndAppearance(w io.Writer, csrfToken, returnTo, locale string, appearance site.Appearance) error {
	if locale != "tr" {
		locale = "en"
	}
	var mfaChallenge bytes.Buffer
	if err := designsystem.AuthenticatorChallenge().Render(context.Background(), &mfaChallenge); err != nil {
		return err
	}
	if appearance.Normalize() != nil {
		appearance = site.Appearance{LoginOverlayOpacity: .4}
	}
	loginCSS := AdminLoginCSS
	if appearance.LoginBackgroundURL != "" {
		opacity := strconv.FormatFloat(appearance.LoginOverlayOpacity, 'f', 2, 64)
		loginCSS += fmt.Sprintf(`\n.login-hero-pane{background-image:linear-gradient(rgba(9,10,15,%s),rgba(9,10,15,%s)),url("%s");background-position:center;background-size:cover}`, opacity, opacity, appearance.LoginBackgroundURL)
	}
	pageTemplate := `<!doctype html>
<html lang="tr" data-theme="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Yönetici Girişi · HydroGo Studio</title>
<script>
(()=>{try{const saved=localStorage.getItem('hydro-ui-theme-v2');const variant=localStorage.getItem('hydro-theme')==='light'?'light':'dark';if(saved&&saved.endsWith('-light')){document.documentElement.setAttribute('data-theme','light');}else if(saved&&saved.endsWith('-dark')){document.documentElement.setAttribute('data-theme','dark');}else{document.documentElement.setAttribute('data-theme',variant);}}catch(_){}})();
</script>
<style>
%s
</style>
</head>
<body class="login-body">
<div class="login-layout">

	<aside class="login-hero-pane">
		<div class="hero-glow-sphere"></div>

		<header class="hero-header">
			<a class="brand-badge" href="/" aria-label="HydroGo Ana Sayfa">
				<div class="brand-icon-box">
					<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
				</div>
				<div class="brand-text-wrap">
					<span class="brand-name">HYDRO<strong>GO</strong></span>
					<span class="brand-tag">Studio</span>
				</div>
			</a>
		</header>

		<div class="hero-content">
			<p class="hero-eyebrow">The workspace for modern publishing</p>
			<h1 class="hero-headline">Güçlü Yönetim, <br><span class="gradient-text">Yüksek Performans</span></h1>
			<p class="hero-description">Manage your content, memberships, and commerce operations from one fast, secure workspace.</p>
			<nav class="hero-links" aria-label="Resources"><a href="%s">Documentation ↗</a><a href="/">Home ↗</a></nav>
		</div>
	</aside>

	<main class="login-form-pane">
		<div class="pane-top-actions">
			<nav class="login-language-switch" aria-label="Language"><a href="%s"%s>EN</a><a href="%s"%s>TR</a></nav>
			<button id="theme-toggle-btn" class="theme-switch-btn" type="button" aria-label="Temayı Değiştir">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
				<span id="theme-toggle-label">Tema</span>
			</button>
		</div>

		<div class="form-container">
			<div class="form-header">
				<h2 class="form-title">Hoş Geldiniz</h2>
				<p class="form-subtitle">Yönetim merkezine erişmek için giriş yapın.</p>
			</div>

			<form id="login-form" class="auth-form" method="post">
				<input type="hidden" name="_csrf" value="%s">
				%s

				<div class="input-group">
					<label class="input-label" for="login-email">E-posta Adresi</label>
					<div class="input-wrapper">
						<span class="input-icon">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>
						</span>
						<input id="login-email" class="auth-input" name="email" type="email" required autocomplete="username" placeholder="admin@hydrogo.local" autofocus>
					</div>
				</div>

				<div class="input-group">
					<div class="input-label-row">
						<label class="input-label" for="login-password">Şifre</label>
						<a href="mailto:destek@hydrogo.local?subject=Sifre%%20Sifirlama" class="link-subtle">Şifremi unuttum?</a>
					</div>
					<div class="input-wrapper">
						<span class="input-icon">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
						</span>
						<input id="login-password" class="auth-input" name="password" type="password" required autocomplete="current-password" placeholder="••••••••••••">
					</div>
				</div>

				<div class="form-row-utility">
					<label class="checkbox-label">
						<input type="checkbox" name="remember" id="remember-me">
						<span class="checkbox-custom">
							<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5"><polyline points="20 6 9 17 4 12"></polyline></svg>
						</span>
						<span>Beni hatırla</span>
					</label>
				</div>

				<button class="btn-submit" type="submit">
					<span>Giriş Yap</span>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg>
				</button>

				<div id="login-status" class="form-alert" role="alert"></div>
			</form>
		</div>

		<footer class="form-pane-footer">
			<span>HydroGo Studio</span>
		</footer>
	</main>

</div>

<script>
(function(){
	var form = document.getElementById('login-form');
	var status = document.getElementById('login-status');
	var themeBtn = document.getElementById('theme-toggle-btn');
	var themeLabel = document.getElementById('theme-toggle-label');
	var rememberCheckbox = document.getElementById('remember-me');

	if (rememberCheckbox) {
		rememberCheckbox.addEventListener('change', function() {
			var checkCustom = rememberCheckbox.nextElementSibling;
			if (rememberCheckbox.checked) {
				checkCustom.style.background = 'var(--brand-primary)';
				checkCustom.style.borderColor = 'var(--brand-primary)';
				var svg = checkCustom.querySelector('svg');
				if (svg) svg.style.display = 'block';
			} else {
				checkCustom.style.background = 'var(--bg-input)';
				checkCustom.style.borderColor = 'var(--border-light)';
				var svg = checkCustom.querySelector('svg');
				if (svg) svg.style.display = 'none';
			}
		});
	}

	function updateThemeLabel() {
		var isDark = document.documentElement.getAttribute('data-theme') === 'dark';
		if (themeLabel) themeLabel.textContent = isDark ? 'Koyu Mod' : 'Açık Mod';
	}
	updateThemeLabel();

	if (themeBtn) {
		themeBtn.addEventListener('click', function() {
			var current = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
			document.documentElement.setAttribute('data-theme', current);
			try {
				localStorage.setItem('hydro-theme', current);
				localStorage.setItem('hydro-ui-theme-v2', 'workbench-' + current);
			} catch(_) {}
			updateThemeLabel();
		});
	}

	function showMsg(type, text) {
		status.className = 'form-alert alert-' + type;
		status.textContent = text;
		status.style.display = 'block';
	}

	form.addEventListener('submit', async function(e) {
		e.preventDefault();
		var btn = form.querySelector('button[type="submit"]');
		var orig = btn.innerHTML;
		btn.disabled = true;
		btn.innerHTML = '<span>Giriş yapılıyor…</span>';
		status.style.display = 'none';

	try {
			var response = await fetch('/auth/login', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/x-www-form-urlencoded',
					'X-CSRF-Token': form.elements._csrf.value,
					'Accept': 'application/json'
				},
				body: new URLSearchParams(new FormData(form))
			});
			var loginResult = {};
			try { loginResult = await response.json(); } catch (_) {}

			if (!response.ok) {
				if (response.status === 401 && loginResult.error === 'mfa_code_required') {
					var challenge = document.getElementById('mfa-challenge');
					var code = document.getElementById('login-mfa-code');
					var countdown = document.querySelector('[data-mfa-countdown]');
					var nextCode = document.querySelector('[data-mfa-next]');
					challenge.hidden = false;
					code.disabled = false;
					code.required = true;
					var period = Math.max(1, Number(loginResult.totp_period_seconds) || 30);
					var serverOffset = Number(loginResult.server_time_unix) * 1000 - Date.now();
					var initialSlot = Math.floor(Number(loginResult.server_time_unix) / period);
					var tick = function() {
						var serverNow = Date.now() + serverOffset;
						var slot = Math.floor(serverNow / 1000 / period);
						var remaining = period - (Math.floor(serverNow / 1000) %% period);
						countdown.textContent = 'Kodun süresi ' + String(Math.floor(remaining / 60)).padStart(2, '0') + ':' + String(remaining %% 60).padStart(2, '0') + ' sonra dolacak.';
						nextCode.disabled = slot <= initialSlot;
					};
				tick();
					if (!challenge.dataset.timerStarted) {
						challenge.dataset.timerStarted = 'true';
						window.setInterval(tick, 250);
						nextCode.addEventListener('click', function() {
							code.value = '';
							code.focus();
							status.textContent = 'Authenticator uygulamanızdan yeni kodu girin.';
						});
					}
					showMsg('warning', 'Parola doğru. İki adımlı doğrulama kodunuzu girin.');
					code.focus();
				} else if (response.status === 401 && !document.getElementById('mfa-challenge').hidden) {
					showMsg('error', 'Kod doğrulanamadı. Authenticator uygulamanızdaki güncel kodu deneyin.');
				} else if (response.status === 401) {
					showMsg('error', 'E-posta veya şifre hatalı. Lütfen bilgilerinizi kontrol edin.');
				} else if (response.status === 429) {
					showMsg('error', 'Çok fazla başarısız deneme yapıldı. Lütfen biraz bekleyin.');
				} else if (response.status === 403) {
					showMsg('error', 'Güvenlik doğrulaması başarısız. Lütfen sayfayı yenileyin.');
				} else {
					showMsg('error', 'Giriş şu anda tamamlanamadı. Lütfen tekrar deneyin.');
				}
				btn.disabled = false;
				btn.innerHTML = orig;
				return;
			}

			var notice = loginResult.security_notice || {}, attempts = Number(notice.failed_attempts || 0);
			var message = 'Giriş başarılı! Yönetim paneline aktarılıyorsunuz…';
			if (notice.previous_login_at) { message = 'Önceki başarılı giriş: ' + new Date(notice.previous_login_at).toLocaleString(document.documentElement.lang === 'tr' ? 'tr-TR' : 'en-US') + '. ' + attempts + ' başarısız deneme.'; }
			else if (attempts > 0) { message = 'İlk başarılı girişiniz. Önceki oturumdan bu yana ' + attempts + ' başarısız deneme kaydedildi.'; }
			showMsg(attempts > 0 ? 'warning' : 'success', message);
			setTimeout(function() {
				window.location.href = '%s';
			}, (notice.previous_login_at || attempts > 0) ? 1800 : 250);
		} catch (_) {
			showMsg('error', 'Sunucuya ulaşılamadı. Lütfen bağlantınızı kontrol edip tekrar deneyin.');
			btn.disabled = false;
			btn.innerHTML = orig;
		}
	});
})();
</script>
</body>
</html>`
	loginURL := func(language string) string {
		query := url.Values{"lang": {language}}
		if returnTo != "/admin" {
			query.Set("return_to", returnTo)
		}
		return template.HTMLEscapeString("/login?" + query.Encode())
	}
	current := func(language string) string {
		if locale == language {
			return ` aria-current="page"`
		}
		return ""
	}
	docsURL := "/docs"
	if locale == "tr" {
		docsURL = "/docs?lang=tr"
	}
	_, err := fmt.Fprintf(w, localizeAdminLoginPage(locale, pageTemplate),
		template.CSS(loginCSS),
		docsURL,
		loginURL("en"), current("en"), loginURL("tr"), current("tr"),
		template.HTMLEscapeString(csrfToken),
		localizeAdminLoginMFA(locale, mfaChallenge.String()),
		template.HTMLEscapeString(returnTo),
	)
	return err
}

// AdminLoginCSP returns hash sources for the exact inline blocks rendered on the login page.
func AdminLoginCSP(document []byte) string {
	return security.InlineCSPDirectives(document)
}

var adminLoginEnglish = strings.NewReplacer(
	`<html lang="tr"`, `<html lang="en"`,
	`Yönetici Girişi · HydroGo Studio`, `Sign in · HydroGo Studio`,
	`HydroGo Ana Sayfa`, `HydroGo home`,
	`Güçlü Yönetim,`, `One workspace,`,
	`Yüksek Performans`, `built for speed`,
	`Temayı Değiştir`, `Change theme`,
	`<span id="theme-toggle-label">Tema</span>`, `<span id="theme-toggle-label">Theme</span>`,
	`Hoş Geldiniz`, `Welcome back`,
	`Yönetim merkezine erişmek için giriş yapın.`, `Sign in to your Admin Studio workspace.`,
	`E-posta Adresi`, `Email address`,
	`Şifremi unuttum?`, `Forgot password?`,
	`>Şifre<`, `>Password<`,
	`Beni hatırla`, `Remember me`,
	`>Giriş Yap<`, `>Sign in<`,
	`'Koyu Mod'`, `'Dark mode'`,
	`'Açık Mod'`, `'Light mode'`,
	`Giriş yapılıyor…`, `Signing in…`,
	`Kodun süresi `, `Code expires in `,
	` sonra dolacak.`, `.`,
	`Authenticator uygulamanızdan yeni kodu girin.`, `Enter the next code from your authenticator app.`,
	`Parola doğru. İki adımlı doğrulama kodunuzu girin.`, `Password accepted. Enter your two-step verification code.`,
	`Kod doğrulanamadı. Authenticator uygulamanızdaki güncel kodu deneyin.`, `Code could not be verified. Try the current code in your authenticator app.`,
	`E-posta veya şifre hatalı. Lütfen bilgilerinizi kontrol edin.`, `Email or password is incorrect. Check your details and try again.`,
	`Çok fazla başarısız deneme yapıldı. Lütfen biraz bekleyin.`, `Too many attempts. Please wait and try again.`,
	`Güvenlik doğrulaması başarısız. Lütfen sayfayı yenileyin.`, `Security check failed. Refresh the page and try again.`,
	`Giriş şu anda tamamlanamadı. Lütfen tekrar deneyin.`, `Sign in is temporarily unavailable. Please try again.`,
	`Giriş başarılı! Yönetim paneline aktarılıyorsunuz…`, `Signed in. Opening Admin Studio…`,
	`Önceki başarılı giriş: `, `Previous successful sign in: `,
	` başarısız deneme.`, ` failed attempts.`,
	`İlk başarılı girişiniz. Önceki oturumdan bu yana `, `This is your first successful sign in. There have been `,
	` başarısız deneme kaydedildi.`, ` failed attempts since the previous session.`,
	`Sunucuya ulaşılamadı. Lütfen bağlantınızı kontrol edip tekrar deneyin.`, `Could not reach the server. Check your connection and try again.`,
)

var adminLoginTurkish = strings.NewReplacer(
	`The workspace for modern publishing`, `Modern yayıncılık için çalışma alanı`,
	`Manage your content, memberships, and commerce operations from one fast, secure workspace.`, `İçerik, üyelik ve ticaret süreçlerinizi hızlı ve güvenli tek bir çalışma alanından yönetin.`,
	`aria-label="Resources"`, `aria-label="Kaynaklar"`,
	`>Documentation ↗<`, `>Dokümantasyon ↗<`,
	`>Home ↗<`, `>Ana sayfa ↗<`,
	`aria-label="Language"`, `aria-label="Dil"`,
)

func localizeAdminLoginPage(locale, page string) string {
	if locale == "tr" {
		return adminLoginTurkish.Replace(page)
	}
	return adminLoginEnglish.Replace(page)
}

var adminLoginEnglishMFA = strings.NewReplacer(
	`İki adımlı doğrulama`, `Two-step verification`,
	`Authenticator uygulamanızdaki güncel kodu veya kurtarma kodlarından birini girin.`, `Enter the current code from your authenticator app or one of your recovery codes.`,
	`Doğrulama kodu`, `Verification code`,
	`Sonraki kodu kullan`, `Use next code`,
)

func localizeAdminLoginMFA(locale, markup string) string {
	if locale == "tr" {
		return markup
	}
	return adminLoginEnglishMFA.Replace(markup)
}
