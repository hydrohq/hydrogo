package ui

import (
	"html/template"
	"io"
)

type AdminControlData struct {
	Email  string
	Nonce  string
	CSRF   string
	Active string
	Locale string
}

var adminControlTemplate = template.Must(template.New("admin-control").Parse(`<div class="control-container">
	<!-- Üst Grid: Site Yapılandırması & Medya Hattı -->
	<div class="dashboard-grid">
		<!-- Site Yapılandırması & Redirects -->
		<article class="studio-card dashboard-card">
			<div class="dashboard-card-head">
				<div class="card-title-group">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--brand-light);"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
					<h2>Site Yapılandırması</h2>
					<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
						<span class="studio-tip-popover">
							<span class="tip-eyebrow">YAPILANDIRMA</span>
							<p>Sitenin genel ayar parametrelerini (site.title vb.) ve URL yönlendirme kurallarını buradan yönetin.</p>
						</span>
					</span>
				</div>
				<span class="muted" style="font-size:0.8rem;">Anahtar / Değer &amp; URL Kuralları</span>
			</div>

			<div style="display:flex;flex-direction:column;gap:20px;">
				<!-- Ayar Formu -->
				<div>
					<h3 style="font-size:0.95rem;font-weight:700;margin:0 0 10px;color:var(--ink);display:flex;align-items:center;gap:6px;">
						<span>Ayar Kaydet / Güncelle</span>
					</h3>
					<form class="studio-form" id="setting-form" style="margin-top:0;">
						<label style="grid-column:1/-1;">
							<span>Ayar Anahtarı (Key)</span>
							<input name="key" placeholder="Örn: site.title, site.tagline, contact.email" required>
						</label>
						<label style="grid-column:1/-1;">
							<span>Değer (Value)</span>
							<textarea name="value" rows="2" style="min-height:70px;" placeholder="Ayar değerini girin..." required></textarea>
						</label>
						<div style="grid-column:1/-1;display:flex;justify-content:flex-end;">
							<button type="submit" class="button">Ayarı Kaydet</button>
						</div>
					</form>
				</div>

				<div style="height:1px;background:var(--line);"></div>

				<!-- Redirect Formu -->
				<div>
					<h3 style="font-size:0.95rem;font-weight:700;margin:0 0 10px;color:var(--ink);">URL Yönlendirme (Redirect)</h3>
					<form class="studio-form" id="redirect-form" style="margin-top:0;">
						<label>
							<span>Kaynak Yol</span>
							<input name="source_path" placeholder="/eski-sayfa" required>
						</label>
						<label>
							<span>Hedef Yol</span>
							<input name="target_path" placeholder="/yeni-sayfa" required>
						</label>
						<label style="grid-column:1/-1;">
							<span>HTTP Durum Kodu</span>
							<select name="status">
								<option value="301">301 - Kalıcı Yönlendirme (Permanent Redirect)</option>
								<option value="308">308 - Kalıcı Yönlendirme (Permanent Preserve Method)</option>
								<option value="302">302 - Geçici Yönlendirme (Found / Temporary)</option>
								<option value="307">307 - Geçici Yönlendirme (Temporary Preserve Method)</option>
							</select>
						</label>
						<div style="grid-column:1/-1;display:flex;justify-content:flex-end;">
							<button type="submit" class="button button-quiet">Yönlendirmeyi Kaydet</button>
						</div>
					</form>
				</div>
			</div>
		</article>

		<!-- Medya Kütüphanesi & Asset Hattı -->
		<article class="studio-card dashboard-card">
			<div class="dashboard-card-head">
				<div class="card-title-group">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--green);"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
					<h2>Medya &amp; Asset Hattı</h2>
					<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
						<span class="studio-tip-popover">
							<span class="tip-eyebrow">ASSET PIPELINE</span>
							<p>Yüklenen görseller otomatik olarak güvenli kontrol, MIME doğrulaması ve pending &rarr; ready hattından geçer.</p>
						</span>
					</span>
				</div>
				<span class="status status-success" style="font-size:0.75rem;">JPEG, PNG, WebP</span>
			</div>

			<form class="studio-form" id="media-form" enctype="multipart/form-data" style="margin-top:0;">
				<input type="hidden" name="_csrf" value="{{.CSRF}}">
				<label style="grid-column:1/-1;">
					<span>Yüklenecek Görsel Dosyası</span>
					<input name="file" type="file" accept="image/jpeg,image/png,image/webp" required style="padding:8px 12px;background:var(--bg-subtle);">
				</label>
				<div style="grid-column:1/-1;display:flex;align-items:center;justify-content:space-between;gap:12px;margin-top:6px;">
					<small class="muted" style="font-size:0.8rem;">Maksimum dosya boyutu: 10MB</small>
					<button type="submit" class="button">Görseli Yükle</button>
				</div>
			</form>

			<div style="height:1px;background:var(--line);margin:20px 0;"></div>

			<div>
				<h3 style="font-size:0.92rem;font-weight:700;margin:0 0 10px;color:var(--ink);">Yayın Yüzeyleri &amp; Form Kutuları</h3>
				<div class="collection-grid">
					<a class="collection-tile" href="/site/settings" target="_blank">
						<span class="tile-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg></span>
						<span>
							<strong>Public Site Ayarları JSON ↗</strong>
							<small>Uygulamanın dışa sunduğu genel yapılandırma çıktısı</small>
						</span>
						<span class="arrow">→</span>
					</a>
					<a class="collection-tile" href="/site/navigation/tr" target="_blank">
						<span class="tile-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg></span>
						<span>
							<strong>TR Navigasyon Menüsü ↗</strong>
							<small>Yayınlanan menü ve hiyerarşi ağacı JSON çıktısı</small>
						</span>
						<span class="arrow">→</span>
					</a>
					<a class="collection-tile" href="/admin/site/inbox/contact" target="_blank">
						<span class="tile-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg></span>
						<span>
							<strong>İletişim Formu Gelen Kutusu ↗</strong>
							<small>Web sitesinden gönderilen form mesajları</small>
						</span>
						<span class="arrow">→</span>
					</a>
				</div>
			</div>
		</article>
	</div>

	<!-- Orta Grid: Kullanıcı & Davet, Sistem Tanıları -->
	<div class="dashboard-grid">
		<!-- Kullanıcı & Davet -->
		<article class="studio-card dashboard-card">
			<div class="dashboard-card-head">
				<div class="card-title-group">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--brand-light);"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="8.5" cy="7" r="4"></circle><line x1="20" y1="8" x2="20" y2="14"></line><line x1="23" y1="11" x2="17" y2="11"></line></svg>
					<h2>Yönetici &amp; Editör Daveti</h2>
					<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
						<span class="studio-tip-popover">
							<span class="tip-eyebrow">ERİŞİM YÖNETİMİ</span>
							<p>Yeni bir yönetici veya editör için güvenli davet e-postası tetikleyin.</p>
						</span>
					</span>
				</div>
				<span class="muted" style="font-size:0.8rem;">E-posta &amp; Rol Belirleme</span>
			</div>

			<form class="studio-form" id="invite-form" style="margin-top:0;">
				<input type="hidden" name="_csrf" value="{{.CSRF}}">
				<label>
					<span>Davet Edilecek E-posta</span>
					<input name="email" type="email" placeholder="ornek@alanadi.com" required>
				</label>
				<label>
					<span>Atanacak Rol</span>
					<select name="role">
						<option value="editor">Editor (İçerik &amp; Taslak Yönetimi)</option>
						<option value="viewer">Viewer (Salt Okunur Gözlem)</option>
						<option value="admin">Admin (Tam Yetkili Yönetici)</option>
					</select>
				</label>
				<div style="grid-column:1/-1;display:flex;align-items:center;justify-content:space-between;gap:12px;">
					<small class="muted" style="font-size:0.8rem;">Davet bağlantısı 24 saat geçerlidir.</small>
					<button type="submit" class="button">Davet Gönder</button>
				</div>
			</form>

			<div style="height:1px;background:var(--line);margin:20px 0;"></div>

			<div style="display:flex;gap:10px;flex-wrap:wrap;">
				<a class="button button-quiet" href="/auth/me" target="_blank" style="flex:1;min-height:38px;justify-content:center;">
					<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
					<span>Hesap &amp; MFA Durumu ↗</span>
				</a>
				<a class="button button-quiet" href="/admin/permissions" style="flex:1;min-height:38px;justify-content:center;">
					<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
					<span>Yetki Matrisi</span>
				</a>
			</div>
		</article>

		<!-- Sistem & Geliştirici Uçları -->
		<article class="studio-card dashboard-card">
			<div class="dashboard-card-head">
				<div class="card-title-group">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--brand-light);"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
					<h2>Sistem Tanıları &amp; API Uçları</h2>
					<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
						<span class="studio-tip-popover">
							<span class="tip-eyebrow">OBSERVABILITY &amp; API</span>
							<p>Canlılık, hazırlık, Prometheus metrikleri ve headless API uç noktaları.</p>
						</span>
					</span>
				</div>
				<span class="muted" style="font-size:0.8rem;">Gözlemlenebilirlik</span>
			</div>

			<div class="tool-grid" style="grid-template-columns:repeat(2,minmax(0,1fr));">
				<a class="tool-tile" href="/healthz" target="_blank">
					<div class="tool-header">
						<span class="tool-badge tool-badge-green">HTTP 200</span>
						<b>↗</b>
					</div>
					<strong>Healthz (Liveness)</strong>
					<small>Temel servis çalışma kontrolü</small>
				</a>

				<a class="tool-tile" href="/readyz" target="_blank">
					<div class="tool-header">
						<span class="tool-badge tool-badge-green">READY</span>
						<b>↗</b>
					</div>
					<strong>Readyz (Readiness)</strong>
					<small>Veritabanı &amp; havuz hazırlığı</small>
				</a>

				<a class="tool-tile" href="/metrics" target="_blank">
					<div class="tool-header">
						<span class="tool-badge">METRICS</span>
						<b>↗</b>
					</div>
					<strong>Prometheus Metrikleri</strong>
					<small>İstek, gecikme ve bellek verileri</small>
				</a>

				<a class="tool-tile" href="/api/graphql" target="_blank">
					<div class="tool-header">
						<span class="tool-badge tool-badge-purple">GRAPHQL</span>
						<b>↗</b>
					</div>
					<strong>GraphQL Endpoint</strong>
					<small>Headless veri sorgu arabirimi</small>
				</a>
			</div>
		</article>
	</div>

	<!-- Alt Bar: Hızlı Modül Kısayolları -->
	<section class="studio-card" style="margin-top:0;">
		<div class="dashboard-card-head" style="margin-bottom:14px;padding-bottom:10px;">
			<div class="card-title-group">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" style="color:var(--brand-light);"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
				<h2>Modül Yönetim Kısayolları</h2>
				<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
					<span class="studio-tip-popover">
						<span class="tip-eyebrow">HIZLI ERİŞİM</span>
						<p>Sık kullanılan içerik, e-ticaret ve üye yönetimi panellerine doğrudan geçiş.</p>
					</span>
				</span>
			</div>
			<span class="muted" style="font-size:0.8rem;">Doğrudan Erişim</span>
		</div>
		<div class="studio-actions" style="margin:0;">
			<a class="button" href="/admin/news">Haber Yönetimi</a>
			<a class="button button-quiet" href="/admin/pages">Sayfa Yönetimi</a>
			<a class="button button-quiet" href="/admin/commerce/products">Ürün Kataloğu</a>
			<a class="button button-quiet" href="/admin/commerce/orders">Siparişler</a>
			<a class="button button-quiet" href="/admin/members">Üyeler</a>
			<a class="button button-quiet" href="/admin/reports">Raporlar &amp; Analitik</a>
		</div>
	</section>
</div>

<script nonce="{{.Nonce}}">
(()=>{
	const csrf = {{printf "%q" .CSRF}};
	const notify = (msg, type = 'info', title = '') => {
		if (typeof window.HydroToast === 'function') {
			window.HydroToast(msg, type, title);
		}
	};

	const postJSON = async (form, url, data, successMsg) => {
		const btn = form.querySelector('button[type="submit"]');
		if (btn) btn.classList.add('is-loading');
		try {
			const res = await fetch(url, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'X-CSRF-Token': csrf
				},
				body: JSON.stringify(data)
			});
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'HTTP ' + res.status);
			}
			notify(successMsg, 'success', 'İşlem Başarılı');
			form.reset();
		} catch (err) {
			notify('İşlem sırasında bir hata oluştu.', 'error', 'Hata');
		} finally {
			if (btn) btn.classList.remove('is-loading');
		}
	};

	// Site Ayarı Kaydet
	document.getElementById('setting-form')?.addEventListener('submit', e => {
		e.preventDefault();
		const form = e.currentTarget;
		const f = new FormData(form);
		postJSON(form, '/admin/site/settings', {
			key: f.get('key'),
			value: f.get('value'),
			public: true
		}, 'Site ayarı başarıyla kaydedildi.');
	});

	// URL Redirect Kaydet
	document.getElementById('redirect-form')?.addEventListener('submit', e => {
		e.preventDefault();
		const form = e.currentTarget;
		const f = new FormData(form);
		postJSON(form, '/admin/site/redirects', {
			source_path: f.get('source_path'),
			target_path: f.get('target_path'),
			status: Number(f.get('status'))
		}, 'URL yönlendirme kuralı başarıyla kaydedildi.');
	});

	// Medya Yükle
	document.getElementById('media-form')?.addEventListener('submit', async e => {
		e.preventDefault();
		const form = e.currentTarget;
		const btn = form.querySelector('button[type="submit"]');
		if (btn) btn.classList.add('is-loading');
		try {
			const res = await fetch('/api/media', {
				method: 'POST',
				headers: { 'X-CSRF-Token': csrf },
				body: new FormData(form)
			});
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'HTTP ' + res.status);
			}
			notify('Medya başarıyla yüklendi ve işleme alındı.', 'success', 'Yükleme Başarılı');
			form.reset();
		} catch (err) {
			notify('Medya yüklenirken hata oluştu.', 'error', 'Yükleme Başarısız');
		} finally {
			if (btn) btn.classList.remove('is-loading');
		}
	});

	// Davet Gönder
	document.getElementById('invite-form')?.addEventListener('submit', async e => {
		e.preventDefault();
		const form = e.currentTarget;
		const btn = form.querySelector('button[type="submit"]');
		if (btn) btn.classList.add('is-loading');
		try {
			const res = await fetch('/admin/auth/invites', {
				method: 'POST',
				headers: { 'X-CSRF-Token': csrf },
				body: new URLSearchParams(new FormData(form))
			});
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'HTTP ' + res.status);
			}
			notify('Davet e-postası başarıyla gönderildi.', 'success', 'Davet Gönderildi');
			form.reset();
		} catch (err) {
			notify('Davet gönderilemedi.', 'error', 'İşlem Başarısız');
		} finally {
			if (btn) btn.classList.remove('is-loading');
		}
	});
})();
</script>`))

func RenderAdminControl(w io.Writer, data AdminControlData) error {
	if err := AdminStartLocale(w, "Yönetim araçları", "Platform yönetimi", data.Nonce, "developer", data.Locale); err != nil {
		return err
	}
	if _, err := io.WriteString(w, renderDeveloperTabs("control")); err != nil {
		return err
	}
	if err := adminControlTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}
