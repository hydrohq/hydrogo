package ui

import (
	"html/template"
)

// AdminNovaDashboardLayoutCSS styles adminNovaDashboardTemplate: a from-scratch
// dashboard layout (stat card grid, a content table, and an activity list)
// modeled on the shadcn/Vercel-style reference the customer asked for,
// distinct from the shared wb-* markup the other three modes render. It also
// hides whichever dashboard variant is not active for the current family, so
// both can be server-rendered once and toggled purely by CSS (ADR-0041's
// client-only family selection has no server-side signal to branch on).
const AdminNovaDashboardLayoutCSS = `html:not([data-ui-theme="nova"]) .nova-dashboard{display:none}
html[data-ui-theme="nova"] .wb-dashboard{display:none}
html:not([data-ui-theme="nova"]) .nova-topnav-wrap{display:none}
html[data-ui-theme="nova"] .studio-app{grid-template-columns:minmax(0,1fr)}
html[data-ui-theme="nova"] .studio-sidebar{display:none}
html[data-ui-theme="nova"] .studio-mobile-menu-toggle{display:none}
html[data-ui-theme="nova"] .nova-topnav-wrap{display:flex;align-items:center;gap:20px;min-width:0}
html[data-ui-theme="nova"] .nova-topnav-brand{flex-shrink:0;display:grid;place-items:center;width:30px;height:30px;border-radius:var(--radius-sm);background:var(--brand);color:#fff}
html[data-ui-theme="nova"] .nova-topnav-links{display:flex;align-items:center;gap:20px;min-width:0;overflow-x:auto}
html[data-ui-theme="nova"] .nova-topnav-link{flex-shrink:0;padding:7px 12px;border-radius:var(--radius-sm);color:var(--muted);font-size:.84rem;font-weight:600;text-decoration:none;white-space:nowrap}
html[data-ui-theme="nova"] .nova-topnav-link:hover{background:var(--bg-subtle);color:var(--ink)}
html[data-ui-theme="nova"] .nova-topnav-link.is-active{background:var(--brand-subtle);color:var(--brand-light)}
html[data-ui-theme="nova"] .studio-search{flex-shrink:0;max-width:260px}
@media(max-width:900px){html[data-ui-theme="nova"] .studio-search{display:none}}
html[data-ui-theme="nova"] .studio-topbar{height:64px;min-height:64px;max-height:64px;background:var(--surface);backdrop-filter:blur(10px)}
html[data-ui-theme="nova"] .nova-topnav-brand{background:transparent;color:var(--ink);width:24px;height:24px;border-radius:0}
html[data-ui-theme="nova"] .nova-topnav-link{padding:0;border-radius:0;font-weight:500}
html[data-ui-theme="nova"] .nova-topnav-link:hover{background:none;color:var(--ink)}
html[data-ui-theme="nova"] .nova-topnav-link.is-active{background:none;color:var(--ink);font-weight:600}
html[data-ui-theme="nova"] .studio-search{height:36px;background:var(--surface);margin-left:auto}
html[data-ui-theme="nova"] .studio-search kbd{display:none}
html[data-ui-theme="nova"] .studio-badge-status{display:none}
html[data-ui-theme="nova"] .studio-user{padding:0;background:transparent;border:0}
html[data-ui-theme="nova"] .studio-user .user-avatar{width:32px;height:32px;border-radius:50%;background:var(--surface);color:var(--muted);border:1px solid var(--line)}
html[data-ui-theme="nova"] .studio-user .user-name,html[data-ui-theme="nova"] .studio-user .chevron-icon{display:none}
html[data-ui-theme="nova"] .ui-family-select{border-color:transparent;background:transparent}
html[data-ui-theme="nova"] .ui-family-select:hover{background:var(--bg-subtle)}
html[data-ui-theme="nova"] .theme-toggle-btn{border:1px solid var(--line);background:var(--surface)}
html[data-ui-theme="nova"] .theme-toggle-btn:hover{background:var(--bg-subtle);border-color:var(--line-light)}
html[data-ui-theme="nova"] .nova-dashboard{display:grid;gap:20px;max-width:1600px;margin:0 auto}
html[data-ui-theme="nova"] .nova-toolbar{display:flex;align-items:center;justify-content:space-between;gap:16px}
html[data-ui-theme="nova"] .nova-toolbar h2{margin:0 0 2px;font-size:1.3rem;font-weight:700;letter-spacing:-.03em;color:var(--ink)}
html[data-ui-theme="nova"] .nova-toolbar p{margin:0;color:var(--muted);font-size:.84rem}
html[data-ui-theme="nova"] .nova-stat-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}
html[data-ui-theme="nova"] .nova-stat-card{padding:18px 20px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);box-shadow:var(--card-highlight)}
html[data-ui-theme="nova"] .nova-stat-top{display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:10px}
html[data-ui-theme="nova"] .nova-stat-top span:first-child{color:var(--muted);font-size:.78rem;font-weight:600}
html[data-ui-theme="nova"] .nova-stat-icon{display:grid;place-items:center;width:26px;height:26px;color:var(--muted)}
html[data-ui-theme="nova"] .nova-stat-icon svg{width:16px;height:16px;stroke-width:2}
html[data-ui-theme="nova"] .nova-stat-card strong{display:block;font-size:1.6rem;font-weight:700;letter-spacing:-.02em;color:var(--ink)}
html[data-ui-theme="nova"] .nova-stat-card small{display:block;margin-top:4px;color:var(--muted);font-size:.76rem}
html[data-ui-theme="nova"] .nova-content-grid{display:grid;grid-template-columns:minmax(0,1.5fr) minmax(300px,1fr);gap:16px;align-items:start}
html[data-ui-theme="nova"] .nova-side-stack{display:grid;gap:16px;min-width:0}
html[data-ui-theme="nova"] .nova-status-dot{display:inline-flex;align-items:center;gap:5px;padding:2px 8px;border-radius:999px;background:var(--green-subtle);color:var(--green);font-size:.68rem;font-weight:700}
html[data-ui-theme="nova"] .nova-status-dot::before{content:'';width:5px;height:5px;border-radius:50%;background:var(--green)}
html[data-ui-theme="nova"] .nova-status-list{display:grid;gap:0;margin:0;padding:0 20px 16px}
html[data-ui-theme="nova"] .nova-status-list>div{display:flex;justify-content:space-between;gap:10px;padding:8px 0;border-bottom:1px solid var(--line)}
html[data-ui-theme="nova"] .nova-status-list>div:last-child{border-bottom:0;padding-bottom:0}
html[data-ui-theme="nova"] .nova-status-list dt{margin:0;color:var(--muted);font-size:.78rem}
html[data-ui-theme="nova"] .nova-status-list dd{margin:0;color:var(--ink);font-weight:600;font-size:.78rem}
html[data-ui-theme="nova"] .nova-table-card,html[data-ui-theme="nova"] .nova-activity-card{border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);box-shadow:var(--card-highlight);overflow:hidden}
html[data-ui-theme="nova"] .nova-card-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;padding:18px 20px 14px}
html[data-ui-theme="nova"] .nova-card-head h3{margin:0;font-size:.98rem;font-weight:700;letter-spacing:-.01em;color:var(--ink)}
html[data-ui-theme="nova"] .nova-card-head p{margin:2px 0 0;color:var(--muted);font-size:.78rem}
html[data-ui-theme="nova"] .nova-card-head a{color:var(--brand-light);font-size:.8rem;font-weight:600;text-decoration:none;white-space:nowrap}
html[data-ui-theme="nova"] .nova-card-head a:hover{text-decoration:underline}
html[data-ui-theme="nova"] .nova-table-wrap{width:100%;overflow-x:auto;-webkit-overflow-scrolling:touch}
html[data-ui-theme="nova"] .nova-table{width:100%;border-collapse:collapse;min-width:600px}
html[data-ui-theme="nova"] .nova-table th{padding:8px 20px;border-top:1px solid var(--line);border-bottom:1px solid var(--line);background:var(--bg-subtle);color:var(--muted);font-size:.68rem;font-weight:700;letter-spacing:.04em;text-transform:uppercase;text-align:left}
html[data-ui-theme="nova"] .nova-table td{padding:12px 20px;border-bottom:1px solid var(--line);font-size:.84rem;color:var(--ink);vertical-align:middle}
html[data-ui-theme="nova"] .nova-table tr:last-child td{border-bottom:0}
html[data-ui-theme="nova"] .nova-table td.muted{color:var(--muted)}
html[data-ui-theme="nova"] .nova-table-action a{color:var(--brand-light);font-weight:600;text-decoration:none;font-size:.8rem}
html[data-ui-theme="nova"] .nova-table-action a:hover{text-decoration:underline}
html[data-ui-theme="nova"] .nova-table .nova-empty{color:var(--muted);text-align:center;padding:24px 20px}
html[data-ui-theme="nova"] .nova-badge{display:inline-flex;align-items:center;padding:2px 9px;border-radius:999px;background:var(--brand-subtle);color:var(--brand-light);font-size:.7rem;font-weight:700}
html[data-ui-theme="nova"] .nova-activity-list{display:flex;flex-direction:column;padding:4px 10px 14px}
html[data-ui-theme="nova"] .nova-activity-row{display:flex;align-items:center;gap:12px;padding:8px 10px;border-radius:var(--radius-md);color:inherit;text-decoration:none}
html[data-ui-theme="nova"] .nova-activity-row:hover{background:var(--bg-subtle)}
html[data-ui-theme="nova"] .nova-activity-icon{flex-shrink:0;display:grid;place-items:center;width:32px;height:32px;border-radius:var(--radius-sm);background:var(--bg-subtle);border:1px solid var(--line);color:var(--muted)}
html[data-ui-theme="nova"] .nova-activity-icon svg{width:15px;height:15px}
html[data-ui-theme="nova"] .nova-activity-row:hover .nova-activity-icon{border-color:var(--line-light);color:var(--ink)}
html[data-ui-theme="nova"] .nova-activity-copy{min-width:0}
html[data-ui-theme="nova"] .nova-activity-copy strong{display:block;font-size:.84rem;font-weight:600;color:var(--ink);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
html[data-ui-theme="nova"] .nova-activity-copy small{display:block;margin-top:1px;color:var(--muted);font-size:.72rem;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
@media(max-width:1000px){html[data-ui-theme="nova"] .nova-stat-grid{grid-template-columns:repeat(2,minmax(0,1fr))}html[data-ui-theme="nova"] .nova-content-grid{grid-template-columns:1fr}}
@media(max-width:640px){html[data-ui-theme="nova"] .nova-stat-grid{grid-template-columns:1fr}html[data-ui-theme="nova"] .nova-toolbar{flex-direction:column;align-items:flex-start;gap:12px}html[data-ui-theme="nova"] .nova-toolbar .button{width:100%;justify-content:center}}`

var adminNovaDashboardTemplate = template.Must(template.New("admin-nova-dashboard").Parse(`<div class="nova-dashboard">
	<div class="nova-toolbar">
		<div>
			<h2>Genel Bakış</h2>
			<p>Sitenizin son durumu tek bakışta.</p>
		</div>
		<a class="button" href="/admin/news/new"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Yeni İçerik</a>
	</div>

	<section class="nova-stat-grid" aria-label="Sistem özeti">
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Toplam İstek</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M22 12h-4l-3 9L9 3l-3 9H2"></path></svg></span></div>
			<strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong>
			<small>{{if .Metrics}}Anlık aktif: {{.Metrics.InFlight}}{{else}}İstek yok{{end}}</small>
		</article>
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Görüntülenme</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></span></div>
			<strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong>
			<small>Public &amp; mağaza sayfaları</small>
		</article>
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Aktif Üyeler</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle></svg></span></div>
			<strong>{{.MemberStats.ActiveMembers}}</strong>
			<small>Doğrulanmış: {{.MemberStats.VerifiedMembers}}</small>
		</article>
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Ortalama Yanıt</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg></span></div>
			<strong>{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}0 ms{{end}}</strong>
			<small>Süreç içi ortalama</small>
		</article>
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Hatalı İstekler</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"></polygon><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg></span></div>
			<strong>{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</strong>
			<small>4xx ve 5xx yanıtlar</small>
		</article>
		<article class="nova-stat-card">
			<div class="nova-stat-top"><span>Kayıtlı Üyeler</span><span class="nova-stat-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg></span></div>
			<strong>{{.MemberStats.TotalMembers}}</strong>
			<small>{{.MemberStats.ActiveMembers}} aktif</small>
		</article>
	</section>

	<section class="nova-content-grid">
		<div class="nova-table-card">
			<div class="nova-card-head">
				<div>
					<h3>İçerik Alanları</h3>
					<p>Yönetilebilir koleksiyonlar ve varlıklar</p>
				</div>
				<a href="/admin/news">Tümünü gör</a>
			</div>
			<div class="nova-table-wrap">
				<table class="nova-table">
					<thead><tr><th>Alan</th><th>Açıklama</th><th>Yetenek</th><th></th></tr></thead>
					<tbody>
						{{$visible := 0}}
						{{range .Collections}}
						{{if ne .ID "pages"}}
						{{$visible = 1}}
						<tr>
							<td><strong>{{.Label}}</strong></td>
							<td class="muted">{{.Description}}</td>
							<td><span class="nova-badge">{{len .Kits}} yetenek</span></td>
							<td class="nova-table-action"><a href="{{if .Actions}}{{(index .Actions 0).Href}}{{else}}/admin/{{.ID}}{{end}}">Yönet</a></td>
						</tr>
						{{end}}
						{{end}}
						{{if eq $visible 0}}
						<tr><td colspan="4" class="nova-empty">Henüz etkin koleksiyon bulunmuyor.</td></tr>
						{{end}}
					</tbody>
				</table>
			</div>
		</div>

		<div class="nova-side-stack">
			<div class="nova-activity-card">
				<div class="nova-card-head">
					<div>
						<h3>Hızlı İşlemler</h3>
						<p>Sık kullanılan kısayollar</p>
					</div>
				</div>
				<nav class="nova-activity-list" aria-label="Hızlı işlemler">
					<a class="nova-activity-row" href="/admin/news/new"><span class="nova-activity-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 0 6.5 22H20V4H7a3 3 0 0 0-3 3v12.5Z"></path><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path><path d="M9 8h7"></path><path d="M9 12h6"></path></svg></span><span class="nova-activity-copy"><strong>Haber yayınla</strong><small>Yeni editoryal içerik ekle</small></span></a>
					{{if .CommerceModuleEnabled}}<a class="nova-activity-row" href="/admin/commerce/products/new"><span class="nova-activity-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 7 9-4 9 4-9 4Z"></path><path d="m3 7 9 4 9-4"></path><path d="M3 7v10l9 4 9-4V7"></path><path d="M12 11v10"></path></svg></span><span class="nova-activity-copy"><strong>Ürün oluştur</strong><small>Kataloğa yeni ürün gir</small></span></a>{{end}}
					<a class="nova-activity-row" href="/admin/media"><span class="nova-activity-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"></rect><circle cx="8.5" cy="9.5" r="1.5"></circle><path d="m21 15-4-4L7 20"></path></svg></span><span class="nova-activity-copy"><strong>Medya havuzu</strong><small>Varlıkları yönet ve yükle</small></span></a>
					{{if .PushEnabled}}<a class="nova-activity-row" href="/admin/push"><span class="nova-activity-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path></svg></span><span class="nova-activity-copy"><strong>Bildirim yayınla</strong><small>Web ve mobil cihazlara duyur</small></span></a>{{end}}
					{{if .EmailPreviewEnabled}}<a class="nova-activity-row" href="/admin/mailer/test-preview"><span class="nova-activity-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"></rect><path d="m22 7-10 5L2 7"></path></svg></span><span class="nova-activity-copy"><strong>Test email preview</strong><small>Verify delivery to your inbox</small></span></a>{{end}}
				</nav>
			</div>

			<div class="nova-activity-card">
				<div class="nova-card-head">
					<div>
						<h3>Sistem Durumu</h3>
						<p>Anlık çalışma bilgileri</p>
					</div>
					<span class="nova-status-dot">Aktif</span>
				</div>
				<dl class="nova-status-list">
					<div><dt>Aktif üyeler</dt><dd>{{.MemberStats.ActiveMembers}}</dd></div>
					<div><dt>Doğrulanmış hesaplar</dt><dd>{{.MemberStats.VerifiedMembers}}</dd></div>
					<div><dt>Anlık istek</dt><dd>{{if .Metrics}}{{.Metrics.InFlight}}{{else}}0{{end}}</dd></div>
					<div><dt>Yanıt süresi</dt><dd>{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}Normal{{end}}</dd></div>
				</dl>
			</div>
		</div>
	</section>
</div>`))
