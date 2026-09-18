package ui

import (
	"fmt"
	"html/template"
	"io"
)

// AdminLoginCSS contains the flat, low-radius, dense Linear + Carbon styling for the admin login screen.
const AdminLoginCSS = `
:root {
	--login-bg: #070b12;
	--login-surface: #111927;
	--login-surface-hover: #182235;
	--login-line: #1e293b;
	--login-line-focus: #0ea5e9;
	--login-ink: #f8fafc;
	--login-ink-secondary: #94a3b8;
	--login-muted: #64748b;
	--login-brand: #0ea5e9;
	--login-brand-hover: #38bdf8;
	--login-brand-subtle: rgba(14,165,233,0.12);
	--login-green: #10b981;
	--login-green-subtle: rgba(16,185,129,0.1);
	--login-danger: #ef4444;
	--login-danger-subtle: rgba(239,68,68,0.1);
	--login-radius-xs: 6px;
	--login-radius-sm: 8px;
	--login-radius-md: 12px;
	--login-radius-lg: 16px;
	--login-radius-xl: 24px;
	--login-font-sans: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI Variable", "Segoe UI", sans-serif;
	--login-font-mono: "JetBrains Mono", "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
}

@media (prefers-color-scheme: light) {
	:root {
		--login-bg: #f8fafc;
		--login-surface: #ffffff;
		--login-surface-hover: #f4f7fa;
		--login-line: #e2e8f0;
		--login-line-focus: #0284c7;
		--login-ink: #0f172a;
		--login-ink-secondary: #334155;
		--login-muted: #64748b;
		--login-brand: #0284c7;
		--login-brand-hover: #0369a1;
		--login-brand-subtle: rgba(2,132,199,0.08);
		--login-green: #059669;
		--login-green-subtle: rgba(5,150,105,0.08);
		--login-danger: #dc2626;
		--login-danger-subtle: rgba(220,38,38,0.08);
	}
}

html[data-theme="dark"] {
	--login-bg: #070b12;
	--login-surface: #111927;
	--login-surface-hover: #182235;
	--login-line: #1e293b;
	--login-line-focus: #0ea5e9;
	--login-ink: #f8fafc;
	--login-ink-secondary: #94a3b8;
	--login-muted: #64748b;
	--login-brand: #0ea5e9;
	--login-brand-hover: #38bdf8;
	--login-brand-subtle: rgba(14,165,233,0.12);
	--login-green: #10b981;
	--login-green-subtle: rgba(16,185,129,0.1);
	--login-danger: #ef4444;
	--login-danger-subtle: rgba(239,68,68,0.1);
}

html[data-theme="light"] {
	--login-bg: #f8fafc;
	--login-surface: #ffffff;
	--login-surface-hover: #f4f7fa;
	--login-line: #e2e8f0;
	--login-line-focus: #0284c7;
	--login-ink: #0f172a;
	--login-ink-secondary: #334155;
	--login-muted: #64748b;
	--login-brand: #0284c7;
	--login-brand-hover: #0369a1;
	--login-brand-subtle: rgba(2,132,199,0.08);
	--login-green: #059669;
	--login-green-subtle: rgba(5,150,105,0.08);
	--login-danger: #dc2626;
	--login-danger-subtle: rgba(220,38,38,0.08);
}

*, *::before, *::after {
	box-sizing: border-box;
}

body.admin-login-body {
	margin: 0;
	padding: 0;
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	background: var(--login-bg);
	color: var(--login-ink);
	font-family: var(--login-font-sans);
	font-size: 14px;
	line-height: 1.5;
	-webkit-font-smoothing: antialiased;
	-moz-osx-font-smoothing: grayscale;
}

.admin-login-shell {
	width: 100%;
	max-width: 380px;
	padding: 24px 16px;
	display: flex;
	flex-direction: column;
	gap: 16px;
}

.admin-login-header {
	display: flex;
	flex-direction: column;
	align-items: center;
	text-align: center;
	gap: 8px;
}

.admin-login-brand {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	color: var(--login-ink);
	text-decoration: none;
	font-size: 1.15rem;
	font-weight: 800;
	letter-spacing: -0.02em;
	transition: opacity 0.12s cubic-bezier(0.16, 1, 0.3, 1);
}

.admin-login-brand:hover {
	opacity: 0.85;
}

.admin-login-brand-icon {
	display: grid;
	place-items: center;
	width: 28px;
	height: 28px;
	border-radius: var(--login-radius-sm);
	background: var(--login-brand-subtle);
	color: var(--login-brand);
	border: 1px solid var(--login-line);
}

.admin-login-badge {
	display: inline-flex;
	align-items: center;
	gap: 5px;
	font-family: var(--login-font-mono);
	font-size: 0.68rem;
	font-weight: 700;
	letter-spacing: 0.06em;
	text-transform: uppercase;
	color: var(--login-brand);
	background: var(--login-brand-subtle);
	border: 1px solid rgba(94, 106, 210, 0.25);
	padding: 2px 8px;
	border-radius: var(--login-radius-xs);
}

.admin-login-badge-dot {
	width: 5px;
	height: 5px;
	border-radius: 1px;
	background: var(--login-brand);
}

.admin-login-card {
	background: var(--login-surface);
	border: 1px solid var(--login-line);
	border-radius: var(--login-radius-lg);
	padding: 22px 20px 20px;
	box-shadow: none;
}

.admin-login-title-row {
	margin-bottom: 16px;
	padding-bottom: 10px;
	border-bottom: 1px solid var(--login-line);
}

.admin-login-card h1 {
	margin: 0 0 3px;
	font-size: 1.05rem;
	font-weight: 700;
	letter-spacing: -0.02em;
	color: var(--login-ink);
}

.admin-login-lead {
	margin: 0;
	font-size: 0.78rem;
	color: var(--login-muted);
	line-height: 1.4;
}

.admin-login-form {
	display: flex;
	flex-direction: column;
	gap: 12px;
}

.admin-login-field {
	display: flex;
	flex-direction: column;
	gap: 5px;
}

.admin-login-label {
	font-size: 0.76rem;
	font-weight: 600;
	color: var(--login-ink-secondary);
}

.admin-login-input-wrap {
	position: relative;
	display: flex;
	align-items: center;
}

.admin-login-input-icon {
	position: absolute;
	left: 10px;
	color: var(--login-muted);
	pointer-events: none;
	width: 14px;
	height: 14px;
	display: flex;
	align-items: center;
	justify-content: center;
}

.admin-login-input {
	width: 100%;
	height: 32px;
	min-height: 32px;
	padding: 0 10px 0 32px;
	background: var(--login-surface);
	border: 1px solid var(--login-line);
	border-radius: var(--login-radius-md);
	color: var(--login-ink);
	font-family: inherit;
	font-size: 0.84rem;
	outline: none;
	transition: border-color 0.12s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.12s cubic-bezier(0.16, 1, 0.3, 1);
}

.admin-login-input::placeholder {
	color: var(--login-muted);
	opacity: 0.7;
}

.admin-login-input:hover {
	border-color: var(--login-ink-secondary);
}

.admin-login-input:focus,
.admin-login-input:focus-visible {
	border-color: var(--login-brand);
	outline: 2px solid var(--login-brand);
	outline-offset: 1px;
}

.admin-login-submit {
	height: 32px;
	min-height: 32px;
	margin-top: 4px;
	padding: 0 14px;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: 6px;
	background: var(--login-brand);
	color: #ffffff;
	border: 1px solid transparent;
	border-radius: var(--login-radius-md);
	font-family: inherit;
	font-size: 0.82rem;
	font-weight: 600;
	cursor: pointer;
	text-decoration: none;
	transition: background-color 0.12s cubic-bezier(0.16, 1, 0.3, 1), transform 0.08s cubic-bezier(0.16, 1, 0.3, 1);
}

.admin-login-submit:hover {
	background: var(--login-brand-hover);
}

.admin-login-submit:active {
	transform: scale(0.985);
}

.admin-login-submit:focus-visible {
	outline: 2px solid var(--login-brand);
	outline-offset: 2px;
}

.admin-login-submit:disabled {
	opacity: 0.6;
	cursor: not-allowed;
	transform: none;
}

.admin-login-status {
	display: none;
	padding: 8px 10px;
	border-radius: var(--login-radius-sm);
	font-size: 0.76rem;
	line-height: 1.4;
	font-weight: 500;
}

.admin-login-status.status-error {
	display: block;
	background: var(--login-danger-subtle);
	border: 1px solid rgba(239, 68, 68, 0.3);
	color: var(--login-danger);
}

.admin-login-status.status-success {
	display: block;
	background: var(--login-green-subtle);
	border: 1px solid rgba(16, 185, 129, 0.3);
	color: var(--login-green);
}

.admin-login-footer {
	margin-top: 14px;
	padding-top: 12px;
	border-top: 1px solid var(--login-line);
	display: flex;
	align-items: center;
	justify-content: space-between;
	font-size: 0.72rem;
	color: var(--login-muted);
}

.admin-login-trust {
	display: inline-flex;
	align-items: center;
	gap: 5px;
	font-family: var(--login-font-mono);
}

.admin-login-trust svg {
	width: 12px;
	height: 12px;
	color: var(--login-green);
}

.admin-login-back {
	color: var(--login-muted);
	text-decoration: none;
	display: inline-flex;
	align-items: center;
	gap: 4px;
	transition: color 0.12s ease;
}

.admin-login-back:hover {
	color: var(--login-ink);
}

.admin-login-theme-toggle {
	align-self: center;
	background: transparent;
	border: 1px solid var(--login-line);
	color: var(--login-muted);
	font-size: 0.72rem;
	font-family: var(--login-font-mono);
	padding: 4px 8px;
	border-radius: var(--login-radius-sm);
	cursor: pointer;
	display: inline-flex;
	align-items: center;
	gap: 5px;
	transition: background 0.12s ease, color 0.12s ease;
}

.admin-login-theme-toggle:hover {
	background: var(--login-surface-hover);
	color: var(--login-ink);
}

.admin-login-theme-toggle:focus-visible {
	outline: 2px solid var(--login-brand);
	outline-offset: 1px;
}

@media (prefers-reduced-motion: reduce) {
	.admin-login-submit,
	.admin-login-input,
	.admin-login-back,
	.admin-login-brand {
		transition: none !important;
		transform: none !important;
	}
}
`

// RenderAdminLogin renders the admin login screen with Linear + Carbon density + Radix interaction styling.
func RenderAdminLogin(w io.Writer, csrfToken, nonce, returnTo string) error {
	_, err := fmt.Fprintf(w, `<!doctype html>
<html lang="tr" data-ui-theme="workbench">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Yönetici Girişi · HydroGo Studio</title>
<script nonce="%s">
(()=>{try{const saved=localStorage.getItem('hydro-ui-theme-v2');const variant=localStorage.getItem('hydro-theme')==='dark'?'dark':'light';if(saved&&saved.endsWith('-dark')){document.documentElement.setAttribute('data-theme','dark');}else if(saved&&saved.endsWith('-light')){document.documentElement.setAttribute('data-theme','light');}else{document.documentElement.setAttribute('data-theme',variant);}}catch(_){}})();
</script>
<style nonce="%s">
%s
</style>
</head>
<body class="admin-login-body">
<main class="admin-login-shell">
	<header class="admin-login-header">
		<a class="admin-login-brand" href="/" aria-label="HydroGo Ana Sayfa">
			<span class="admin-login-brand-icon">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
			</span>
			<span>HYDRO<strong>GO</strong></span>
		</a>
		<div class="admin-login-badge"><span class="admin-login-badge-dot"></span> STUDIO YÖNETİM</div>
	</header>

	<section class="admin-login-card">
		<div class="admin-login-title-row">
			<h1>Yönetici Girişi</h1>
			<p class="admin-login-lead">HydroGo Studio yönetim paneline erişmek için yetkili bilgilerinizi girin.</p>
		</div>

		<form id="login-form" class="admin-login-form" method="post">
			<input type="hidden" name="_csrf" value="%s">

			<div class="admin-login-field">
				<label class="admin-login-label" for="login-email">E-posta Adresi</label>
				<div class="admin-login-input-wrap">
					<span class="admin-login-input-icon">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>
					</span>
					<input id="login-email" class="admin-login-input" name="email" type="email" required autocomplete="username" placeholder="admin@site.com">
				</div>
			</div>

			<div class="admin-login-field">
				<label class="admin-login-label" for="login-password">Şifre</label>
				<div class="admin-login-input-wrap">
					<span class="admin-login-input-icon">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
					</span>
					<input id="login-password" class="admin-login-input" name="password" type="password" required autocomplete="current-password" placeholder="••••••••">
				</div>
			</div>

			<button class="admin-login-submit" type="submit">
				<span>Panele Giriş Yap</span>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"></line><polyline points="12 5 19 12 12 19"></polyline></svg>
			</button>

			<div id="login-status" class="admin-login-status" role="alert"></div>
		</form>

		<footer class="admin-login-footer">
			<div class="admin-login-trust">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
				<span>Güvenli yetkili oturum</span>
			</div>
			<a href="/" class="admin-login-back">← Ana Sayfa</a>
		</footer>
	</section>

	<button id="theme-toggle-btn" class="admin-login-theme-toggle" type="button">
		<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
		<span id="theme-toggle-label">Tema Değiştir</span>
	</button>
</main>

<script nonce="%s">
(function(){
	var form = document.getElementById('login-form');
	var status = document.getElementById('login-status');
	var themeBtn = document.getElementById('theme-toggle-btn');
	var themeLabel = document.getElementById('theme-toggle-label');

	function updateThemeLabel() {
		var cur = document.documentElement.getAttribute('data-theme') === 'dark' ? 'Koyu' : 'Açık';
		if (themeLabel) themeLabel.textContent = cur + ' Mod';
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
		status.className = 'admin-login-status status-' + type;
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
					showMsg('error', 'E-posta veya şifre hatalı.');
				} else if (response.status === 429) {
					showMsg('error', 'Çok fazla deneme yapıldı. Lütfen biraz bekleyin.');
				} else if (response.status === 403) {
					showMsg('error', 'Güvenlik doğrulaması başarısız. Sayfayı yenileyin.');
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
			showMsg('error', 'Bağlantı kurulamadı. Lütfen internet bağlantınızı kontrol edip tekrar deneyin.');
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
