package ui

import (
	"fmt"
	"html/template"
	"io"
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
	background: radial-gradient(circle at 20% 20%, rgba(14, 165, 233, 0.12) 0%, transparent 50%),
	            #0b111c;
	padding: 56px 64px;
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	border-right: 1px solid var(--border-subtle);
	overflow: hidden;
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
		flex: none;
		padding: 36px 24px;
	}
	.hero-headline {
		font-size: 1.85rem;
	}
	.hero-content {
		padding: 16px 0;
	}
	.login-form-pane {
		flex: 1;
		padding: 36px 24px;
	}
}
`

// RenderAdminLogin renders the clean high-performance login screen for HydroGo.
func RenderAdminLogin(w io.Writer, csrfToken, nonce, returnTo string) error {
	_, err := fmt.Fprintf(w, `<!doctype html>
<html lang="tr" data-theme="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Yönetici Girişi · HydroGo Studio</title>
<script nonce="%s">
(()=>{try{const saved=localStorage.getItem('hydro-ui-theme-v2');const variant=localStorage.getItem('hydro-theme')==='light'?'light':'dark';if(saved&&saved.endsWith('-light')){document.documentElement.setAttribute('data-theme','light');}else if(saved&&saved.endsWith('-dark')){document.documentElement.setAttribute('data-theme','dark');}else{document.documentElement.setAttribute('data-theme',variant);}}catch(_){}})();
</script>
<style nonce="%s">
%s
</style>
</head>
<body class="login-body">
<div class="login-layout">

	<!-- Sol Bölüm: Platform Kimlik Paneli -->
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
			<h1 class="hero-headline">Güçlü Yönetim, <br><span class="gradient-text">Yüksek Performans</span></h1>
			<p class="hero-description">İçerik, üyelik ve e-ticaret operasyonlarınızı tek bir merkezden hızlı ve güvenli şekilde yönetin.</p>
		</div>
	</aside>

	<!-- Sağ Bölüm: Giriş Formu -->
	<main class="login-form-pane">
		<div class="pane-top-actions">
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
						<input type="checkbox" name="remember" style="display:none;" id="remember-me">
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

<script nonce="%s">
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

			if (!response.ok) {
				if (response.status === 401) {
					showMsg('error', 'E-posta veya şifre hatalı. Lütfen bilgilerinizi kontrol edin.');
				} else if (response.status === 429) {
					showMsg('error', 'Çok fazla başarısız deneme yapıldı. Lütfen biraz bekleyin.');
				} else if (response.status === 403) {
					showMsg('error', 'Güvenlik doğrulaması (CSRF) başarısız. Lütfen sayfayı yenileyin.');
				} else {
					showMsg('error', 'Giriş şu anda tamamlanamadı. Lütfen tekrar deneyin.');
				}
				btn.disabled = false;
				btn.innerHTML = orig;
				return;
			}

			showMsg('success', 'Giriş başarılı! Yönetim paneline aktarılıyorsunuz…');
			setTimeout(function() {
				window.location.href = '%s';
			}, 250);
		} catch (_) {
			showMsg('error', 'Sunucuya ulaşılamadı. Lütfen bağlantınızı kontrol edip tekrar deneyin.');
			btn.disabled = false;
			btn.innerHTML = orig;
		}
	});
})();
</script>
</body>
</html>`,
		template.HTMLEscapeString(nonce),
		template.HTMLEscapeString(nonce),
		template.CSS(AdminLoginCSS),
		template.HTMLEscapeString(csrfToken),
		template.HTMLEscapeString(nonce),
		template.HTMLEscapeString(returnTo),
	)
	return err
}
