package ui

import "html/template"

// AdminWorkbenchDashboardCSS is the structural layer for the Workbench. It
// intentionally owns layout, hierarchy, and interaction states separately
// from the palette tokens in AdminWorkbenchCSS.
const AdminWorkbenchComponentsCSS = `html[data-ui-theme="workbench"] .wb-dashboard{display:grid;gap:16px;max-width:1600px;margin:0 auto}
html[data-ui-theme="workbench"] .wb-command-row{display:flex;justify-content:space-between;align-items:center;gap:12px;padding-bottom:12px;border-bottom:1px solid var(--line)}
html[data-ui-theme="workbench"] .wb-breadcrumb{display:flex;align-items:center;gap:6px;color:var(--muted);font-size:.76rem;font-weight:600}
html[data-ui-theme="workbench"] .wb-scope-badge{display:inline-flex;align-items:center;gap:5px;padding:3px 9px;border-radius:var(--radius-sm);background:var(--bg-subtle);border:1px solid var(--line);color:var(--ink-secondary);font-size:.7rem;font-weight:700;font-family:var(--font-mono)}
html[data-ui-theme="workbench"] .wb-scope-badge i{width:5px;height:5px;border-radius:50%;background:var(--brand)}
html[data-ui-theme="workbench"] .wb-command-actions{display:flex;align-items:center;gap:8px}
html[data-ui-theme="workbench"] .wb-command-actions .button{min-height:32px;height:32px;padding:0 14px;font-size:.78rem;border-radius:var(--radius-md);font-weight:600}
html[data-ui-theme="workbench"] .wb-hero{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:14px 20px;border:1px solid var(--line);border-radius:var(--radius-lg);background:linear-gradient(180deg,var(--surface) 0%,var(--bg-subtle) 100%);box-shadow:var(--card-highlight);position:relative;overflow:hidden}
html[data-ui-theme="workbench"] .wb-hero-copy{display:flex;flex-direction:column;justify-content:center}
html[data-ui-theme="workbench"] .wb-eyebrow{display:inline-flex;align-items:center;gap:6px;color:var(--brand-light);font-size:.66rem;font-weight:700;letter-spacing:.08em;text-transform:uppercase}
html[data-ui-theme="workbench"] .wb-eyebrow i{width:6px;height:6px;border-radius:2px;background:var(--green);box-shadow:0 0 0 2px var(--green-subtle)}
html[data-ui-theme="workbench"] .wb-hero h2{margin:4px 0 2px;font-size:1.2rem;line-height:1.2;letter-spacing:-.03em;font-weight:700;color:var(--ink)}
html[data-ui-theme="workbench"] .wb-hero p{margin:0;color:var(--ink-secondary);font-size:.8rem;line-height:1.4}
html[data-ui-theme="workbench"] .wb-hero-meta{display:flex;align-items:center;gap:8px;font-size:.72rem;color:var(--muted);flex-shrink:0}
html[data-ui-theme="workbench"] .wb-hero-tag{display:inline-flex;align-items:center;gap:5px;padding:3px 8px;border-radius:var(--radius-xs);background:var(--bg-subtle);border:1px solid var(--line);color:var(--ink-secondary);font:600 .7rem var(--font-mono)}
html[data-ui-theme="workbench"] .wb-metric-strip{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px}
html[data-ui-theme="workbench"] .wb-metric{display:flex;flex-direction:column;justify-content:space-between;min-height:114px;padding:16px 18px;border:1px solid var(--line);border-radius:var(--radius-md);background:var(--surface);box-shadow:var(--card-highlight);transition:all var(--motion-fast) var(--ease-standard);position:relative;overflow:hidden}
html[data-ui-theme="workbench"] .wb-metric:hover{border-color:var(--brand);box-shadow:0 0 0 1px var(--brand-glow),var(--card-highlight)}
html[data-ui-theme="workbench"] .wb-metric-head{display:flex;justify-content:space-between;align-items:center;width:100%}
html[data-ui-theme="workbench"] .wb-metric label{display:block;color:var(--muted);font-size:.68rem;font-weight:700;letter-spacing:.05em;text-transform:uppercase}
html[data-ui-theme="workbench"] .wb-metric-tag{padding:2px 7px;border-radius:var(--radius-xs);background:var(--bg-subtle);border:1px solid var(--line);color:var(--muted);font:700 .62rem var(--font-mono);letter-spacing:.02em}
html[data-ui-theme="workbench"] .wb-metric-tag.is-live{background:var(--green-subtle);color:var(--green);border-color:rgba(16,185,129,0.3)}
html[data-ui-theme="workbench"] .wb-metric strong{display:block;margin:8px 0 6px;color:var(--ink);font:700 1.6rem/1 var(--font-mono);letter-spacing:-.03em}
html[data-ui-theme="workbench"] .wb-metric-footer{display:flex;align-items:center;justify-content:space-between;gap:6px;margin-top:auto;font-size:.72rem;color:var(--muted);line-height:1.3}
html[data-ui-theme="workbench"] .wb-metric-footer .trend-up{color:var(--green);font-weight:700}
html[data-ui-theme="workbench"] .wb-metric-footer .trend-neutral{color:var(--muted)}
html[data-ui-theme="workbench"] .wb-content-grid{display:grid;grid-template-columns:minmax(0,1.35fr) minmax(320px,.65fr);gap:16px;align-items:stretch}
html[data-ui-theme="workbench"] .wb-panel{min-width:0;padding:18px 20px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);box-shadow:var(--card-highlight);display:flex;flex-direction:column}
html[data-ui-theme="workbench"] .wb-panel-header{display:flex;justify-content:space-between;align-items:flex-start;gap:10px;margin-bottom:14px;padding-bottom:10px;border-bottom:1px solid var(--line)}
html[data-ui-theme="workbench"] .wb-panel-header h3{margin:0;font-size:.92rem;letter-spacing:-.01em;font-weight:700;color:var(--ink)}
html[data-ui-theme="workbench"] .wb-panel-header p{margin:2px 0 0;color:var(--muted);font-size:.74rem}
html[data-ui-theme="workbench"] .wb-panel-link{display:inline-flex;align-items:center;gap:4px;color:var(--brand-light);font-size:.74rem;font-weight:700;text-decoration:none}
html[data-ui-theme="workbench"] .wb-panel-link svg{width:12px;height:12px}
html[data-ui-theme="workbench"] .wb-collection-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px;flex:1}
html[data-ui-theme="workbench"] .wb-collection{display:grid;grid-template-columns:30px minmax(0,1fr) 14px;align-items:center;gap:12px;min-width:0;padding:12px 14px;border:1px solid var(--line);border-radius:var(--radius-md);color:var(--ink);text-decoration:none;background:var(--bg-subtle);box-shadow:var(--card-highlight);transition:all var(--motion-fast) var(--ease-standard)}
html[data-ui-theme="workbench"] .wb-collection:hover{background:var(--surface-hover);border-color:var(--brand);box-shadow:0 0 0 1px var(--brand-glow),var(--shadow-xs)}
html[data-ui-theme="workbench"] .wb-collection-icon{display:grid;place-items:center;width:30px;height:30px;border-radius:var(--radius-sm);background:var(--surface);border:1px solid var(--line);color:var(--brand-light)}
html[data-ui-theme="workbench"] .wb-collection-icon svg{width:15px;height:15px}
html[data-ui-theme="workbench"] .wb-collection strong,html[data-ui-theme="workbench"] .wb-collection small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
html[data-ui-theme="workbench"] .wb-collection strong{font-size:.8rem;font-weight:600;color:var(--ink)}
html[data-ui-theme="workbench"] .wb-collection small{margin-top:2px;color:var(--muted);font-size:.7rem}
html[data-ui-theme="workbench"] .wb-collection>svg{width:13px;height:13px;color:var(--muted);transition:transform var(--motion-fast) ease}
html[data-ui-theme="workbench"] .wb-collection:hover>svg{color:var(--brand-light);transform:translateX(2px)}
html[data-ui-theme="workbench"] .wb-side-stack{display:flex;flex-direction:column;gap:16px;height:100%}
html[data-ui-theme="workbench"] .wb-side-stack .wb-panel{flex:1}
html[data-ui-theme="workbench"] .wb-action-list{display:grid;gap:6px}
html[data-ui-theme="workbench"] .wb-action{display:grid;grid-template-columns:28px 1fr 12px;align-items:center;gap:10px;padding:8px 10px;border-radius:var(--radius-md);border:1px solid transparent;color:var(--ink);text-decoration:none;transition:all var(--motion-fast) var(--ease-standard)}
html[data-ui-theme="workbench"] .wb-action:hover{background:var(--surface-hover);border-color:var(--line)}
html[data-ui-theme="workbench"] .wb-action-icon{display:grid;place-items:center;width:28px;height:28px;border-radius:var(--radius-sm);background:var(--bg-subtle);border:1px solid var(--line);color:var(--brand-light)}
html[data-ui-theme="workbench"] .wb-action-icon svg{width:14px;height:14px}
html[data-ui-theme="workbench"] .wb-action strong{display:block;font-size:.78rem;font-weight:600;color:var(--ink)}
html[data-ui-theme="workbench"] .wb-action small{display:block;margin-top:1px;color:var(--muted);font-size:.68rem}
html[data-ui-theme="workbench"] .wb-action>svg{width:12px;height:12px;color:var(--muted);transition:transform var(--motion-fast) ease}
html[data-ui-theme="workbench"] .wb-action:hover>svg{color:var(--brand-light);transform:translateX(2px)}
html[data-ui-theme="workbench"] .wb-health-list{display:grid;gap:8px;margin:0;padding:0;list-style:none}
html[data-ui-theme="workbench"] .wb-health-list li{display:flex;justify-content:space-between;align-items:center;gap:10px;padding-bottom:7px;border-bottom:1px solid var(--line);font-size:.76rem}
html[data-ui-theme="workbench"] .wb-health-list li:last-child{padding-bottom:0;border-bottom:0}
html[data-ui-theme="workbench"] .wb-health-list span{color:var(--muted)}
html[data-ui-theme="workbench"] .wb-health-list strong{color:var(--ink);font-size:.76rem;text-align:right;font-family:var(--font-mono)}
html[data-ui-theme="workbench"] .wb-health-state{display:inline-flex;align-items:center;gap:5px;color:var(--green);font-size:.68rem;font-weight:700;padding:2px 7px;border-radius:var(--radius-sm);background:var(--green-subtle);border:1px solid rgba(16,185,129,.25)}
html[data-ui-theme="workbench"] .wb-health-state i{width:5px;height:5px;border-radius:1px;background:var(--green)}
html[data-ui-theme="workbench"] .wb-empty{padding:14px;border:1px dashed var(--line);border-radius:var(--radius-md);color:var(--muted);font-size:.75rem;text-align:center}
@media(max-width:1000px){.wb-content-grid{grid-template-columns:1fr}}
@media(max-width:700px){html[data-ui-theme="workbench"] .wb-command-row{align-items:flex-start;flex-direction:column}html[data-ui-theme="workbench"] .wb-command-actions{width:100%}html[data-ui-theme="workbench"] .wb-command-actions .button{flex:1;justify-content:center}html[data-ui-theme="workbench"] .wb-metric-strip{grid-template-columns:repeat(2,minmax(0,1fr))}html[data-ui-theme="workbench"] .wb-collection-grid{grid-template-columns:1fr}html[data-ui-theme="workbench"] .wb-panel{padding:14px}}
@media(prefers-reduced-motion:reduce){html[data-ui-theme="workbench"] .wb-metric,html[data-ui-theme="workbench"] .wb-collection,html[data-ui-theme="workbench"] .wb-action{transition:none}}`

var adminWorkbenchDashboardTemplate = template.Must(template.New("admin-workbench-dashboard").Parse(`<div class="wb-dashboard">
<div class="wb-command-row">
	<div class="wb-breadcrumb">
		<strong>Studio</strong>
		<span>/</span>
		<span class="wb-scope-badge"><i></i> Canlı Sistem</span>
	</div>
	<div class="wb-command-actions">
		<a class="button button-quiet" href="/admin/reports"><span data-studio-icon="reports"></span> Telemetri &amp; Raporlar</a>
		<a class="button" href="/admin/news/new"><span data-studio-icon="plus"></span> Yeni İçerik</a>
	</div>
</div>

<section class="wb-hero">
	<div class="wb-hero-copy">
		<span class="wb-eyebrow"><i></i> HydroGo Core Motoru</span>
		<h2>HydroGo Yönetim Merkezi</h2>
		<p>İçerik, e-ticaret ve üye operasyonları tek merkezden yönetiliyor.</p>
	</div>
	<div class="wb-hero-meta">
		<span class="wb-hero-tag"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg> Canlı Proses</span>
		<span class="wb-hero-tag"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg> v1.0 Enterprise</span>
	</div>
</section>

<section class="wb-metric-strip" aria-label="Sistem özeti">
	<div class="wb-metric">
		<div class="wb-metric-head">
			<label>HTTP İstekleri</label>
			<span class="wb-metric-tag is-live">Canlı</span>
		</div>
		<strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 24 L0 18 Q25 8 50 14 T100 6 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 18 Q25 8 50 14 T100 6"></path>
		</svg>
		<div class="wb-metric-footer">
			<span>Oturum trafiği</span>
			<span class="trend-up">Aktif</span>
		</div>
	</div>
	<div class="wb-metric">
		<div class="wb-metric-head">
			<label>Görüntülenme</label>
			<span class="wb-metric-tag">Genel</span>
		</div>
		<strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--green)" d="M0 24 L0 16 Q30 20 60 10 T100 4 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--green)" d="M0 16 Q30 20 60 10 T100 4"></path>
		</svg>
		<div class="wb-metric-footer">
			<span>Sayfa ve vitrin trafiği</span>
			<span class="trend-neutral">Stabil</span>
		</div>
	</div>
	<div class="wb-metric">
		<div class="wb-metric-head">
			<label>Ortalama Yanıt</label>
			<span class="wb-metric-tag">Gecikme</span>
		</div>
		<strong>{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}0 ms{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 24 L0 12 Q35 6 70 16 T100 8 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 12 Q35 6 70 16 T100 8"></path>
		</svg>
		<div class="wb-metric-footer">
			<span>Süreç içi yanıt süresi</span>
			<span class="trend-up">Hızlı</span>
		</div>
	</div>
	<div class="wb-metric">
		<div class="wb-metric-head">
			<label>Aktif Üyeler</label>
			<span class="wb-metric-tag is-live">Hesap</span>
		</div>
		<strong>{{.MemberStats.ActiveMembers}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--purple, #a855f7)" d="M0 24 L0 20 Q40 18 70 10 T100 4 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--purple, #a855f7)" d="M0 20 Q40 18 70 10 T100 4"></path>
		</svg>
		<div class="wb-metric-footer">
			<span>Doğrulanmış: {{.MemberStats.VerifiedMembers}}</span>
			<span class="trend-up">Aktif</span>
		</div>
	</div>
</section>

<section class="wb-content-grid">
	<main class="wb-panel">
		<div class="wb-panel-header">
			<div>
				<h3>Koleksiyonlar &amp; İçerik Alanları</h3>
				<p>Yönetilebilir veri modelleri ve varlıklar</p>
			</div>
			<a class="wb-panel-link" href="/admin/pages">Tümünü gör <span data-studio-icon="arrow"></span></a>
		</div>
		<div class="wb-collection-grid">
			{{range .Collections}}
			<a class="wb-collection" href="/admin/{{.ID}}">
				<span class="wb-collection-icon" data-studio-icon="{{.ID}}"></span>
				<span>
					<strong>{{.Label}}</strong>
					<small>{{.Description}}</small>
				</span>
				<span data-studio-icon="arrow"></span>
			</a>
			{{else}}
			<div class="wb-empty">Henüz etkin koleksiyon bulunmuyor.</div>
			{{end}}
		</div>
	</main>

	<aside class="wb-side-stack">
		<article class="wb-panel">
			<div class="wb-panel-header">
				<div>
					<h3>Hızlı İşlemler</h3>
					<p>Sık kullanılan işlemler</p>
				</div>
			</div>
			<nav class="wb-action-list" aria-label="Hızlı işlemler">
				<a class="wb-action" href="/admin/news/new">
					<span class="wb-action-icon" data-studio-icon="news"></span>
					<span><strong>Haber yayınla</strong><small>Yeni editoryal içerik ekle</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action" href="/admin/commerce/products/new">
					<span class="wb-action-icon" data-studio-icon="commerce_products"></span>
					<span><strong>Ürün oluştur</strong><small>Kataloğa yeni ürün gir</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action" href="/admin/pages/new">
					<span class="wb-action-icon" data-studio-icon="pages"></span>
					<span><strong>Yeni sayfa ekle</strong><small>Bloklarla sayfa inşa et</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action" href="/admin/media">
					<span class="wb-action-icon" data-studio-icon="media"></span>
					<span><strong>Medya havuzu</strong><small>Varlıkları yönet ve yükle</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
			</nav>
		</article>

		<article class="wb-panel">
			<div class="wb-panel-header">
				<div>
					<h3>Çalışma Durumu</h3>
					<p>Sistem sağlığı</p>
				</div>
				<span class="wb-health-state"><i></i> Aktif</span>
			</div>
			<ul class="wb-health-list">
				<li><span>Aktif üyeler</span><strong>{{.MemberStats.ActiveMembers}}</strong></li>
				<li><span>Doğrulanmış hesaplar</span><strong>{{.MemberStats.VerifiedMembers}}</strong></li>
				<li><span>Sistem yanıt durumu</span><strong>{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}Normal{{end}}</strong></li>
			</ul>
		</article>
	</aside>
</section>
</div>`))
