package ui

import "html/template"

// AdminWorkbenchDashboardCSS is the structural layer for the Workbench. It
// intentionally owns layout, hierarchy, and interaction states separately
// from the palette tokens in AdminWorkbenchCSS.
const AdminWorkbenchComponentsCSS = `/*! tailwindcss v4.3.3 | MIT License | https://tailwindcss.com */
@layer properties{@supports (((-webkit-hyphens:none)) and (not (margin-trim:inline))) or ((-moz-orient:inline) and (not (color:rgb(from red r g b)))){*,:before,:after,::backdrop{--tw-font-weight:initial;--tw-shadow:0 0 #0000;--tw-shadow-color:initial;--tw-shadow-alpha:100%;--tw-inset-shadow:0 0 #0000;--tw-inset-shadow-color:initial;--tw-inset-shadow-alpha:100%;--tw-ring-color:initial;--tw-ring-shadow:0 0 #0000;--tw-inset-ring-color:initial;--tw-inset-ring-shadow:0 0 #0000;--tw-ring-inset:initial;--tw-ring-offset-width:0px;--tw-ring-offset-color:#fff;--tw-ring-offset-shadow:0 0 #0000;--tw-tracking:initial;--tw-leading:initial}}}@layer utilities;.wb-dashboard{gap:16px;max-width:1600px;margin:0 auto;display:grid}.wb-command-row{border-bottom:1px solid var(--line);justify-content:space-between;align-items:center;gap:12px;padding-bottom:12px;display:flex}.wb-breadcrumb{--tw-font-weight:600;color:var(--muted);align-items:center;gap:6px;font-size:.76rem;font-weight:600;display:flex}.wb-scope-badge{border-radius:var(--radius-sm);background-color:var(--bg-subtle);--tw-font-weight:700;color:var(--ink-secondary);border:1px solid var(--line);font-size:.7rem;font-weight:700;font-family:var(--font-mono);align-items:center;gap:5px;padding:3px 9px;display:inline-flex}.wb-scope-badge i{background-color:var(--brand);border-radius:50%;width:5px;height:5px}.wb-command-actions{align-items:center;gap:8px;display:flex}.wb-command-actions .button{border-radius:var(--radius-md);--tw-font-weight:600;height:32px;min-height:32px;padding:0 14px;font-size:.78rem;font-weight:600}.wb-hero{border-radius:var(--radius-lg);background-image:linear-gradient(180deg,var(--surface) 0%,var(--bg-subtle) 100%);--tw-shadow:var(--card-highlight);box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);border:1px solid var(--line);justify-content:space-between;align-items:center;gap:16px;padding:14px 20px;display:flex;position:relative;overflow:hidden}.wb-hero-copy{flex-direction:column;justify-content:center;display:flex}.wb-eyebrow{--tw-font-weight:700;--tw-tracking:.08em;letter-spacing:.08em;color:var(--brand-light);text-transform:uppercase;align-items:center;gap:6px;font-size:.66rem;font-weight:700;display:inline-flex}.wb-eyebrow i{background-color:var(--green);--tw-shadow:0 0 0 2px var(--tw-shadow-color,var(--green-subtle));width:6px;height:6px;box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);border-radius:2px}.wb-hero h2{--tw-leading:1.2;--tw-font-weight:700;--tw-tracking:-.03em;letter-spacing:-.03em;color:var(--ink);margin:4px 0 2px;font-size:1.2rem;font-weight:700;line-height:1.2}.wb-hero p{--tw-leading:1.4;color:var(--ink-secondary);margin:0;font-size:.8rem;line-height:1.4}.wb-hero-meta{color:var(--muted);flex-shrink:0;align-items:center;gap:8px;font-size:.72rem;display:flex}.wb-hero-tag{border-radius:var(--radius-xs);background-color:var(--bg-subtle);color:var(--ink-secondary);border:1px solid var(--line);font:600 .7rem var(--font-mono);align-items:center;gap:5px;padding:3px 8px;display:inline-flex}.wb-metric-strip{grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;display:grid}.wb-metric{border-radius:var(--radius-md);background-color:var(--surface);--tw-shadow:var(--card-highlight);min-height:114px;box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);border:1px solid var(--line);transition:all var(--motion-fast) var(--ease-standard);flex-direction:column;justify-content:space-between;padding:16px 18px;display:flex;position:relative;overflow:hidden}.wb-metric:hover{border-color:var(--brand);--tw-shadow:0 0 0 1px var(--tw-shadow-color,var(--brand-glow)), var(--card-highlight);box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)}.wb-metric-head{justify-content:space-between;align-items:center;width:100%;display:flex}.wb-metric label{--tw-font-weight:700;--tw-tracking:.05em;letter-spacing:.05em;color:var(--muted);text-transform:uppercase;font-size:.68rem;font-weight:700;display:block}.wb-metric-tag{border-radius:var(--radius-xs);background-color:var(--bg-subtle);--tw-tracking:.02em;letter-spacing:.02em;color:var(--muted);border:1px solid var(--line);font:700 .62rem var(--font-mono);padding:2px 7px}.wb-metric-tag.is-live{background-color:var(--green-subtle);color:var(--green);border-color:#10b9814d}.wb-metric strong{--tw-tracking:-.03em;letter-spacing:-.03em;color:var(--ink);font:700 1.6rem/1 var(--font-mono);margin:8px 0 6px;display:block}.wb-metric-footer{--tw-leading:1.3;color:var(--muted);justify-content:space-between;align-items:center;gap:6px;margin-top:auto;font-size:.72rem;line-height:1.3;display:flex}.wb-metric-footer .trend-up{--tw-font-weight:700;color:var(--green);font-weight:700}.wb-metric-footer .trend-neutral{color:var(--muted)}.wb-content-grid{grid-template-columns:minmax(0,1.35fr) minmax(320px,.65fr);align-items:stretch;gap:16px;display:grid}.wb-panel{border-radius:var(--radius-lg);background-color:var(--surface);--tw-shadow:var(--card-highlight);min-width:0;box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);border:1px solid var(--line);flex-direction:column;padding:18px 20px;display:flex}.wb-panel-header{border-bottom:1px solid var(--line);justify-content:space-between;align-items:flex-start;gap:10px;margin-bottom:14px;padding-bottom:10px;display:flex}.wb-panel-header h3{--tw-font-weight:700;--tw-tracking:-.01em;letter-spacing:-.01em;color:var(--ink);margin:0;font-size:.92rem;font-weight:700}.wb-panel-header p{color:var(--muted);margin:2px 0 0;font-size:.74rem}.wb-panel-link{--tw-font-weight:700;color:var(--brand-light);align-items:center;gap:4px;font-size:.74rem;font-weight:700;text-decoration-line:none;display:inline-flex}.wb-panel-link svg{width:12px;height:12px}.wb-content-grid>main.wb-panel{align-self:start}.wb-collection-grid{grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;display:grid}.wb-collection{border-radius:var(--radius-md);background-color:var(--bg-subtle);min-width:0;min-height:46px;color:var(--ink);--tw-shadow:var(--card-highlight);box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);border:1px solid var(--line);transition:all var(--motion-fast) var(--ease-standard);grid-template-columns:26px minmax(0,1fr) 12px;align-items:center;gap:8px;padding:8px 10px;text-decoration-line:none;display:grid}.wb-collection:hover{border-color:var(--brand);background-color:var(--surface-hover);--tw-shadow:0 0 0 1px var(--tw-shadow-color,var(--brand-glow)), var(--shadow-xs);box-shadow:var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)}.wb-collection-icon{border-radius:var(--radius-sm);background-color:var(--surface);width:26px;height:26px;color:var(--brand-light);border:1px solid var(--line);flex-shrink:0;place-items:center;display:grid}.wb-collection-icon svg{width:14px;height:14px}.wb-collection strong{--tw-leading:1.2;--tw-font-weight:600;white-space:nowrap;color:var(--ink);text-overflow:ellipsis;font-size:.78rem;font-weight:600;line-height:1.2;display:block;overflow:hidden}.wb-collection small{--tw-leading:1.25;white-space:nowrap;color:var(--muted);text-overflow:ellipsis;margin-top:2px;font-size:.66rem;line-height:1.25;display:block;overflow:hidden}.wb-collection>svg{width:12px;height:12px;color:var(--muted);transition:transform var(--motion-fast) ease;flex-shrink:0}.wb-collection:hover>svg{color:var(--brand-light);transform:translate(2px)}.wb-side-stack{flex-direction:column;gap:16px;height:100%;display:flex}.wb-side-stack .wb-panel{flex:1}.wb-action-list{gap:6px;display:grid}.wb-action{border-radius:var(--radius-md);color:var(--ink);transition:all var(--motion-fast) var(--ease-standard);border:1px solid #0000;grid-template-columns:30px 1fr 12px;align-items:center;gap:10px;padding:8px 10px;text-decoration-line:none;display:grid}.wb-action:hover{border-color:var(--line);background-color:var(--surface-hover)}.wb-action-icon{border-radius:var(--radius-sm);background-color:var(--bg-subtle);width:30px;height:30px;color:var(--brand-light);border:1px solid var(--line);flex-shrink:0;place-items:center;display:grid}.wb-action-icon svg{width:15px;height:15px}.wb-action strong{--tw-font-weight:600;color:var(--ink);font-size:.78rem;font-weight:600;display:block}.wb-action small{color:var(--muted);margin-top:1px;font-size:.68rem;display:block}.wb-action>svg{width:12px;height:12px;color:var(--muted);transition:transform var(--motion-fast) ease;flex-shrink:0}.wb-action:hover>svg{color:var(--brand-light);transform:translate(2px)}.wb-health-list{gap:8px;margin:0;padding:0;list-style:none;display:grid}.wb-health-list li{border-bottom:1px solid var(--line);justify-content:space-between;align-items:center;gap:10px;padding-bottom:7px;font-size:.76rem;display:flex}.wb-health-list li:last-child{border-bottom:0;padding-bottom:0}.wb-health-list span{color:var(--muted)}.wb-health-list strong{text-align:right;color:var(--ink);font-size:.76rem;font-family:var(--font-mono)}.wb-health-state{border-radius:var(--radius-sm);background-color:var(--green-subtle);--tw-font-weight:700;color:var(--green);border:1px solid #10b98140;align-items:center;gap:5px;padding:2px 7px;font-size:.68rem;font-weight:700;display:inline-flex}.wb-health-state i{background-color:var(--green);border-radius:1px;width:5px;height:5px}.wb-empty{border-radius:var(--radius-md);text-align:center;color:var(--muted);border:1px dashed var(--line);padding:14px;font-size:.75rem}@media (max-width:1000px){.wb-content-grid{grid-template-columns:1fr}}@media (max-width:1100px){.wb-collection-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}@media (max-width:700px){.wb-command-row{flex-direction:column;align-items:flex-start}.wb-command-actions{width:100%}.wb-command-actions .button{flex:1;justify-content:center}.wb-metric-strip{grid-template-columns:repeat(2,minmax(0,1fr))}.wb-collection-grid{grid-template-columns:1fr}.wb-panel{padding:14px}}@media (prefers-reduced-motion:reduce){.wb-metric,.wb-collection,.wb-action{transition:none}}@property --tw-font-weight{syntax:"*";inherits:false}@property --tw-shadow{syntax:"*";inherits:false;initial-value:0 0 #0000}@property --tw-shadow-color{syntax:"*";inherits:false}@property --tw-shadow-alpha{syntax:"<percentage>";inherits:false;initial-value:100%}@property --tw-inset-shadow{syntax:"*";inherits:false;initial-value:0 0 #0000}@property --tw-inset-shadow-color{syntax:"*";inherits:false}@property --tw-inset-shadow-alpha{syntax:"<percentage>";inherits:false;initial-value:100%}@property --tw-ring-color{syntax:"*";inherits:false}@property --tw-ring-shadow{syntax:"*";inherits:false;initial-value:0 0 #0000}@property --tw-inset-ring-color{syntax:"*";inherits:false}@property --tw-inset-ring-shadow{syntax:"*";inherits:false;initial-value:0 0 #0000}@property --tw-ring-inset{syntax:"*";inherits:false}@property --tw-ring-offset-width{syntax:"<length>";inherits:false;initial-value:0}@property --tw-ring-offset-color{syntax:"*";inherits:false;initial-value:#fff}@property --tw-ring-offset-shadow{syntax:"*";inherits:false;initial-value:0 0 #0000}@property --tw-tracking{syntax:"*";inherits:false}@property --tw-leading{syntax:"*";inherits:false}`

var adminWorkbenchDashboardTemplate = template.Must(template.New("admin-workbench-dashboard").Parse(`<div class="wb-dashboard tw:grid tw:gap-[16px] tw:max-w-[1600px] tw:m-[0_auto]">
<div class="wb-command-row tw:flex tw:justify-between tw:items-center tw:gap-[12px] tw:pb-[12px] tw:[border-bottom:1px_solid_var(--line)]">
	<div class="wb-breadcrumb tw:flex tw:items-center tw:gap-[6px] tw:text-[var(--muted)] tw:text-[.76rem] tw:font-[600]">
		<strong>Studio</strong>
		<span>/</span>
		<span class="wb-scope-badge tw:inline-flex tw:items-center tw:gap-[5px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--ink-secondary)] tw:text-[.7rem] tw:font-[700] tw:[padding:3px_9px] tw:[font-family:var(--font-mono)]"><i class="tw:w-[5px] tw:h-[5px] tw:rounded-[50%] tw:bg-[var(--brand)]"></i> Canlı Sistem</span>
	</div>
	<div class="wb-command-actions tw:flex tw:items-center tw:gap-[8px]">
		<a class="button button-quiet" href="/admin/reports"><span data-studio-icon="reports"></span> Telemetri &amp; Raporlar</a>
		<a class="button" href="/admin/news/new"><span data-studio-icon="plus"></span> Yeni İçerik</a>
	</div>
</div>

<section class="wb-hero tw:flex tw:items-center tw:justify-between tw:gap-[16px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-lg)] tw:bg-[linear-gradient(180deg,var(--surface)_0%,var(--bg-subtle)_100%)] tw:shadow-[var(--card-highlight)] tw:relative tw:overflow-hidden tw:[padding:14px_20px]">
	<div class="wb-hero-copy tw:flex tw:flex-col tw:justify-center">
		<span class="wb-eyebrow tw:inline-flex tw:items-center tw:gap-[6px] tw:text-[var(--brand-light)] tw:text-[.66rem] tw:font-[700] tw:tracking-[.08em] tw:uppercase"><i class="tw:w-[6px] tw:h-[6px] tw:rounded-[2px] tw:bg-[var(--green)] tw:shadow-[0_0_0_2px_var(--green-subtle)]"></i> HydroGo Core Motoru</span>
		<h2 class="tw:m-[4px_0_2px] tw:text-[1.2rem] tw:leading-[1.2] tw:tracking-[-.03em] tw:font-[700] tw:text-[var(--ink)]">HydroGo Yönetim Merkezi</h2>
		<p class="tw:m-[0] tw:text-[var(--ink-secondary)] tw:text-[.8rem] tw:leading-[1.4]">İçerik, e-ticaret ve üye operasyonları tek merkezden yönetiliyor.</p>
	</div>
	<div class="wb-hero-meta tw:flex tw:items-center tw:gap-[8px] tw:text-[.72rem] tw:text-[var(--muted)] tw:shrink-0">
		<span class="wb-hero-tag tw:inline-flex tw:items-center tw:gap-[5px] tw:rounded-[var(--radius-xs)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--ink-secondary)] tw:font-[600_.7rem_var(--font-mono)] tw:[padding:3px_8px]"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg> Canlı Proses</span>
		<span class="wb-hero-tag tw:inline-flex tw:items-center tw:gap-[5px] tw:rounded-[var(--radius-xs)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--ink-secondary)] tw:font-[600_.7rem_var(--font-mono)] tw:[padding:3px_8px]"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg> v1.0 Enterprise</span>
	</div>
</section>

<section class="wb-metric-strip tw:grid tw:grid-cols-[repeat(4,minmax(0,1fr))] tw:gap-[12px]" aria-label="Sistem özeti">
	<div class="wb-metric tw:flex tw:flex-col tw:justify-between tw:min-h-[114px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-md)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:relative tw:overflow-hidden tw:[padding:16px_18px]">
		<div class="wb-metric-head tw:flex tw:justify-between tw:items-center tw:w-full">
			<label class="tw:block tw:text-[var(--muted)] tw:text-[.68rem] tw:font-[700] tw:tracking-[.05em] tw:uppercase">HTTP İstekleri</label>
			<span class="wb-metric-tag is-live tw:rounded-[var(--radius-xs)] tw:bg-[var(--green-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--green)] tw:font-[700_.62rem_var(--font-mono)] tw:tracking-[.02em] tw:[border-color:rgba(16,185,129,.3)] tw:[padding:2px_7px]">Canlı</span>
		</div>
		<strong class="tw:block tw:m-[8px_0_6px] tw:text-[var(--ink)] tw:font-[700_1.6rem/1_var(--font-mono)] tw:tracking-[-.03em]">{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 24 L0 18 Q25 8 50 14 T100 6 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 18 Q25 8 50 14 T100 6"></path>
		</svg>
		<div class="wb-metric-footer tw:flex tw:items-center tw:justify-between tw:gap-[6px] tw:mt-auto tw:text-[.72rem] tw:text-[var(--muted)] tw:leading-[1.3]">
			<span>Oturum trafiği</span>
			<span class="trend-up tw:text-[var(--green)] tw:font-[700]">Aktif</span>
		</div>
	</div>
	<div class="wb-metric tw:flex tw:flex-col tw:justify-between tw:min-h-[114px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-md)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:relative tw:overflow-hidden tw:[padding:16px_18px]">
		<div class="wb-metric-head tw:flex tw:justify-between tw:items-center tw:w-full">
			<label class="tw:block tw:text-[var(--muted)] tw:text-[.68rem] tw:font-[700] tw:tracking-[.05em] tw:uppercase">Görüntülenme</label>
			<span class="wb-metric-tag tw:rounded-[var(--radius-xs)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--muted)] tw:font-[700_.62rem_var(--font-mono)] tw:tracking-[.02em] tw:[padding:2px_7px]">Genel</span>
		</div>
		<strong class="tw:block tw:m-[8px_0_6px] tw:text-[var(--ink)] tw:font-[700_1.6rem/1_var(--font-mono)] tw:tracking-[-.03em]">{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--green)" d="M0 24 L0 16 Q30 20 60 10 T100 4 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--green)" d="M0 16 Q30 20 60 10 T100 4"></path>
		</svg>
		<div class="wb-metric-footer tw:flex tw:items-center tw:justify-between tw:gap-[6px] tw:mt-auto tw:text-[.72rem] tw:text-[var(--muted)] tw:leading-[1.3]">
			<span>Sayfa ve vitrin trafiği</span>
			<span class="trend-neutral tw:text-[var(--muted)]">Stabil</span>
		</div>
	</div>
	<div class="wb-metric tw:flex tw:flex-col tw:justify-between tw:min-h-[114px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-md)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:relative tw:overflow-hidden tw:[padding:16px_18px]">
		<div class="wb-metric-head tw:flex tw:justify-between tw:items-center tw:w-full">
			<label class="tw:block tw:text-[var(--muted)] tw:text-[.68rem] tw:font-[700] tw:tracking-[.05em] tw:uppercase">Ortalama Yanıt</label>
			<span class="wb-metric-tag tw:rounded-[var(--radius-xs)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--muted)] tw:font-[700_.62rem_var(--font-mono)] tw:tracking-[.02em] tw:[padding:2px_7px]">Gecikme</span>
		</div>
		<strong class="tw:block tw:m-[8px_0_6px] tw:text-[var(--ink)] tw:font-[700_1.6rem/1_var(--font-mono)] tw:tracking-[-.03em]">{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}0 ms{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 24 L0 12 Q35 6 70 16 T100 8 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 12 Q35 6 70 16 T100 8"></path>
		</svg>
		<div class="wb-metric-footer tw:flex tw:items-center tw:justify-between tw:gap-[6px] tw:mt-auto tw:text-[.72rem] tw:text-[var(--muted)] tw:leading-[1.3]">
			<span>Süreç içi yanıt süresi</span>
			<span class="trend-up tw:text-[var(--green)] tw:font-[700]">Hızlı</span>
		</div>
	</div>
	<div class="wb-metric tw:flex tw:flex-col tw:justify-between tw:min-h-[114px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-md)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:relative tw:overflow-hidden tw:[padding:16px_18px]">
		<div class="wb-metric-head tw:flex tw:justify-between tw:items-center tw:w-full">
			<label class="tw:block tw:text-[var(--muted)] tw:text-[.68rem] tw:font-[700] tw:tracking-[.05em] tw:uppercase">Aktif Üyeler</label>
			<span class="wb-metric-tag is-live tw:rounded-[var(--radius-xs)] tw:bg-[var(--green-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--green)] tw:font-[700_.62rem_var(--font-mono)] tw:tracking-[.02em] tw:[border-color:rgba(16,185,129,.3)] tw:[padding:2px_7px]">Hesap</span>
		</div>
		<strong class="tw:block tw:m-[8px_0_6px] tw:text-[var(--ink)] tw:font-[700_1.6rem/1_var(--font-mono)] tw:tracking-[-.03em]">{{.MemberStats.ActiveMembers}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 24" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--purple, #a855f7)" d="M0 24 L0 20 Q40 18 70 10 T100 4 L100 24 Z"></path>
			<path class="spark-line" stroke="var(--purple, #a855f7)" d="M0 20 Q40 18 70 10 T100 4"></path>
		</svg>
		<div class="wb-metric-footer tw:flex tw:items-center tw:justify-between tw:gap-[6px] tw:mt-auto tw:text-[.72rem] tw:text-[var(--muted)] tw:leading-[1.3]">
			<span>Doğrulanmış: {{.MemberStats.VerifiedMembers}}</span>
			<span class="trend-up tw:text-[var(--green)] tw:font-[700]">Aktif</span>
		</div>
	</div>
</section>

<section class="wb-content-grid tw:grid tw:grid-cols-[minmax(0,1.35fr)_minmax(320px,.65fr)] tw:items-stretch tw:gap-[16px]">
	<main class="wb-panel tw:min-w-[0] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-lg)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:flex tw:flex-col tw:[padding:18px_20px]">
		<div class="wb-panel-header tw:flex tw:justify-between tw:items-start tw:gap-[10px] tw:pb-[10px] tw:[margin-bottom:14px] tw:[border-bottom:1px_solid_var(--line)]">
			<div>
				<h3 class="tw:m-[0] tw:text-[.92rem] tw:tracking-[-.01em] tw:font-[700] tw:text-[var(--ink)]">Koleksiyonlar &amp; İçerik Alanları</h3>
				<p class="tw:m-[2px_0_0] tw:text-[var(--muted)] tw:text-[.74rem]">Yönetilebilir veri modelleri ve varlıklar</p>
			</div>
			<a class="wb-panel-link tw:inline-flex tw:items-center tw:gap-[4px] tw:text-[var(--brand-light)] tw:text-[.74rem] tw:font-[700] tw:no-underline" href="/admin/pages">Tümünü gör <span data-studio-icon="arrow"></span></a>
		</div>
		<div class="wb-collection-grid tw:grid tw:grid-cols-[repeat(3,minmax(0,1fr))] tw:gap-[8px]">
			{{range .Collections}}
			<a class="wb-collection tw:grid tw:grid-cols-[26px_minmax(0,1fr)_12px] tw:items-center tw:gap-[8px] tw:min-w-[0] tw:min-h-[46px] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-md)] tw:text-[var(--ink)] tw:no-underline tw:bg-[var(--bg-subtle)] tw:shadow-[var(--card-highlight)] tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="{{if .Actions}}{{(index .Actions 0).Href}}{{else}}/admin/{{.ID}}{{end}}">
				<span class="wb-collection-icon tw:grid tw:place-items-center tw:w-[26px] tw:h-[26px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--surface)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="{{.ID}}"></span>
				<span>
					<strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)] tw:leading-[1.2] tw:whitespace-nowrap tw:overflow-hidden tw:text-ellipsis">{{.Label}}</strong>
					<small class="tw:block tw:mt-[2px] tw:text-[var(--muted)] tw:text-[.66rem] tw:leading-[1.25] tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap">{{.Description}}</small>
				</span>
				<span data-studio-icon="arrow"></span>
			</a>
			{{else}}
			<div class="wb-empty tw:p-[14px] tw:border-[1px_dashed_var(--line)] tw:rounded-[var(--radius-md)] tw:text-[var(--muted)] tw:text-[.75rem] tw:text-center">Henüz etkin koleksiyon bulunmuyor.</div>
			{{end}}
		</div>
	</main>

	<aside class="wb-side-stack tw:flex tw:flex-col tw:gap-[16px] tw:h-full">
		<article class="wb-panel tw:min-w-[0] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-lg)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:flex tw:flex-col tw:[padding:18px_20px]">
			<div class="wb-panel-header tw:flex tw:justify-between tw:items-start tw:gap-[10px] tw:pb-[10px] tw:[margin-bottom:14px] tw:[border-bottom:1px_solid_var(--line)]">
				<div>
					<h3 class="tw:m-[0] tw:text-[.92rem] tw:tracking-[-.01em] tw:font-[700] tw:text-[var(--ink)]">Hızlı İşlemler</h3>
					<p class="tw:m-[2px_0_0] tw:text-[var(--muted)] tw:text-[.74rem]">Sık kullanılan işlemler</p>
				</div>
			</div>
			<nav class="wb-action-list tw:grid tw:gap-[6px]" aria-label="Hızlı işlemler">
				<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/news/new">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="news"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Haber yayınla</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Yeni editoryal içerik ekle</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/commerce/products/new">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="commerce_products"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Ürün oluştur</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Kataloğa yeni ürün gir</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/pages/new">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="pages"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Yeni sayfa ekle</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Bloklarla sayfa inşa et</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/media">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="media"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Medya havuzu</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Varlıkları yönet ve yükle</small></span>
					<span data-studio-icon="arrow"></span>
				</a>
				{{if .PushEnabled}}<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/push">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="notifications"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Bildirim yayınla</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Web ve mobil cihazlara duyur</small></span>
					<span data-studio-icon="arrow"></span>
				</a>{{end}}
				{{if .EmailPreviewEnabled}}<a class="wb-action tw:grid tw:grid-cols-[30px_1fr_12px] tw:items-center tw:gap-[10px] tw:rounded-[var(--radius-md)] tw:border-[1px_solid_transparent] tw:text-[var(--ink)] tw:no-underline tw:transition-[all_var(--motion-fast)_var(--ease-standard)] tw:[padding:8px_10px]" href="/admin/mailer/test-preview">
					<span class="wb-action-icon tw:grid tw:place-items-center tw:w-[30px] tw:h-[30px] tw:rounded-[var(--radius-sm)] tw:bg-[var(--bg-subtle)] tw:border-[1px_solid_var(--line)] tw:text-[var(--brand-light)] tw:shrink-0" data-studio-icon="settings"></span>
					<span><strong class="tw:block tw:text-[.78rem] tw:font-[600] tw:text-[var(--ink)]">Test email preview</strong><small class="tw:block tw:mt-[1px] tw:text-[var(--muted)] tw:text-[.68rem]">Verify delivery to your inbox</small></span>
					<span data-studio-icon="arrow"></span>
				</a>{{end}}
			</nav>
		</article>

		<article class="wb-panel tw:min-w-[0] tw:border-[1px_solid_var(--line)] tw:rounded-[var(--radius-lg)] tw:bg-[var(--surface)] tw:shadow-[var(--card-highlight)] tw:flex tw:flex-col tw:[padding:18px_20px]">
			<div class="wb-panel-header tw:flex tw:justify-between tw:items-start tw:gap-[10px] tw:pb-[10px] tw:[margin-bottom:14px] tw:[border-bottom:1px_solid_var(--line)]">
				<div>
					<h3 class="tw:m-[0] tw:text-[.92rem] tw:tracking-[-.01em] tw:font-[700] tw:text-[var(--ink)]">Çalışma Durumu</h3>
					<p class="tw:m-[2px_0_0] tw:text-[var(--muted)] tw:text-[.74rem]">Sistem sağlığı</p>
				</div>
				<span class="wb-health-state tw:inline-flex tw:items-center tw:gap-[5px] tw:text-[var(--green)] tw:text-[.68rem] tw:font-[700] tw:rounded-[var(--radius-sm)] tw:bg-[var(--green-subtle)] tw:border-[1px_solid_rgba(16,185,129,.25)] tw:[padding:2px_7px]"><i class="tw:w-[5px] tw:h-[5px] tw:rounded-[1px] tw:bg-[var(--green)]"></i> Aktif</span>
			</div>
			<ul class="wb-health-list tw:grid tw:gap-[8px] tw:m-[0] tw:p-[0] tw:list-none">
				<li><span class="tw:text-[var(--muted)]">Aktif üyeler</span><strong class="tw:text-[var(--ink)] tw:text-[.76rem] tw:text-right tw:[font-family:var(--font-mono)]">{{.MemberStats.ActiveMembers}}</strong></li>
				<li><span class="tw:text-[var(--muted)]">Doğrulanmış hesaplar</span><strong class="tw:text-[var(--ink)] tw:text-[.76rem] tw:text-right tw:[font-family:var(--font-mono)]">{{.MemberStats.VerifiedMembers}}</strong></li>
				<li><span class="tw:text-[var(--muted)]">Sistem yanıt durumu</span><strong class="tw:text-[var(--ink)] tw:text-[.76rem] tw:text-right tw:[font-family:var(--font-mono)]">{{if .Metrics}}{{.Metrics.AverageLatencyMS}} ms{{else}}Normal{{end}}</strong></li>
			</ul>
		</article>
	</aside>
</section>
</div>`))
