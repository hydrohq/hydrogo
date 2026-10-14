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
	// The login styles live in static/admin/css/login.css. A configured
	// background image is passed as a custom property on the hero pane, so the
	// page needs no inline <style> block.
	heroStyle := ""
	if appearance.LoginBackgroundURL != "" {
		opacity := strconv.FormatFloat(appearance.LoginOverlayOpacity, 'f', 2, 64)
		heroStyle = fmt.Sprintf(` style='--login-background:linear-gradient(rgba(9,10,15,%s),rgba(9,10,15,%s)),url("%s")'`, opacity, opacity, template.HTMLEscapeString(appearance.LoginBackgroundURL))
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
<link rel="stylesheet" href="/admin-assets/css/login.css">
</head>
<body class="login-body">
<div class="login-layout">

	<aside class="login-hero-pane"%s>
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
	var formHeader = form.querySelector('.form-header');

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
		status.setAttribute('role', type === 'error' ? 'alert' : 'status');
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
					if (formHeader) formHeader.hidden = true;
					form.querySelectorAll('.input-group, .form-row-utility').forEach(function(group) { group.hidden = true; });
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
		heroStyle,
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
