package ui

import (
	"fmt"
	"html/template"
	"io"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type AdminDashboardData struct {
	Email       string
	Collections []adminstudio.Collection
	Nonce       string
	Active      string
	Metrics     *observability.Metrics
	MemberStats members.MemberStats
	DataTime    string
	DataSource  string
	DateFilter  string
}

var adminDashboardCleanTemplate = template.Must(template.New("admin-dashboard-clean").Parse(`<section class="dashboard-grid"><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><h2>Çalışma Alanları</h2></div></div><div class="collection-grid">{{range .Collections}}{{if ne .ID "commerce_variants"}}<a class="collection-tile" href="{{(index .Actions 0).Href}}"><span class="tile-icon" data-studio-icon="folder"></span><span><strong>{{.Label}}</strong><small>{{.Description}}</small></span><b class="arrow" data-studio-icon="arrow"></b></a>{{end}}{{end}}<a class="collection-tile" href="/admin/commerce/orders"><span class="tile-icon" data-studio-icon="orders"></span><span><strong>Siparişler</strong><small>Müşteri siparişleri ve ödeme durumları</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/members"><span class="tile-icon" data-studio-icon="folder"></span><span><strong>Üyeler</strong><small>Kullanıcı hesapları, oturumlar ve güvenlik</small></span><b class="arrow" data-studio-icon="arrow"></b></a></div></article><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><h2>Hızlı İşlemler</h2></div></div><div class="collection-grid"><a class="collection-tile" href="/admin/news/new"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Haber Yaz</strong><small>Haber taslağı oluştur ve yayınla</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/pages/new"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Sayfa Ekle</strong><small>Yeni statik veya dinamik sayfa</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/commerce/products#new-product"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Ürün Ekle</strong><small>Katalog, fiyat ve stok girişi</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/media"><span class="tile-icon" data-studio-icon="media"></span><span><strong>Medya Kütüphanesi</strong><small>Görsel ve dosya yönetimi</small></span><b class="arrow" data-studio-icon="arrow"></b></a></div></article></section>`))

var adminDashboardWorkbenchTemplate = template.Must(template.New("admin-dashboard-workbench").Parse(`<section class="dashboard-welcome-panel">
	<div><span class="dashboard-status-line"><i></i> Sistem hazır</span><h2>Bugün neyi yönetiyoruz?</h2><p>İçeriğinizi, kataloğunuzu ve operasyonunuzu tek bir çalışma alanından yönetin.</p></div>
	<div class="dashboard-welcome-actions"><a class="button" href="/admin/news/new"><span data-studio-icon="plus"></span> Yeni içerik</a><a class="button button-quiet" href="/admin/reports">Raporları aç <span data-studio-icon="arrow"></span></a></div>
</section>
<section class="dashboard-overview" aria-label="Genel bakış">
	<article class="dashboard-stat dashboard-stat-accent"><span class="dashboard-stat-label">Toplam istek</span><strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong><small>Bu çalışma oturumunda</small><span class="dashboard-stat-glyph" data-studio-icon="operations"></span></article>
	<article class="dashboard-stat"><span class="dashboard-stat-label">Sayfa görüntülenme</span><strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong><small>Public ve mağaza</small><span class="dashboard-stat-glyph" data-studio-icon="reports"></span></article>
	<article class="dashboard-stat"><span class="dashboard-stat-label">Ortalama yanıt</span><strong>{{if .Metrics}}{{.Metrics.AvgLatencyMS}}{{else}}0{{end}}<em> ms</em></strong><small>İstek başına</small><span class="dashboard-stat-glyph" data-studio-icon="operations"></span></article>
	<article class="dashboard-stat {{if and .Metrics (gt .Metrics.TotalErrors 0)}}is-alert{{end}}"><span class="dashboard-stat-label">Hatalı istek</span><strong>{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</strong><small>{{if and .Metrics (gt .Metrics.TotalErrors 0)}}İncelenmeli{{else}}Sistem sağlıklı{{end}}</small><span class="dashboard-stat-glyph" data-studio-icon="security"></span></article>
</section>
<section class="dashboard-workspace-grid">
	<article class="studio-card dashboard-card dashboard-workspaces"><div class="dashboard-card-head"><div><span class="dashboard-section-kicker">Çalışma alanları</span><h2>İçerik ve ticaret</h2></div><a class="text-link" href="/admin/control">Yönet <span data-studio-icon="arrow"></span></a></div><div class="collection-grid">{{range .Collections}}{{if ne .ID "commerce_variants"}}<a class="collection-tile" href="{{(index .Actions 0).Href}}"><span class="tile-icon" data-studio-icon="folder"></span><span><strong>{{.Label}}</strong><small>{{.Description}}</small></span><b class="arrow" data-studio-icon="arrow"></b></a>{{end}}{{end}}<a class="collection-tile" href="/admin/commerce/orders"><span class="tile-icon" data-studio-icon="orders"></span><span><strong>Siparişler</strong><small>Müşteri siparişleri ve ödeme durumları</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/members"><span class="tile-icon" data-studio-icon="folder"></span><span><strong>Üyeler</strong><small>{{.MemberStats.TotalMembers}} kayıtlı hesap</small></span><b class="arrow" data-studio-icon="arrow"></b></a></div></article>
	<aside class="dashboard-side-stack"><article class="studio-card dashboard-card dashboard-quick"><div class="dashboard-card-head"><div><span class="dashboard-section-kicker">Kısayollar</span><h2>Hızlı işlemler</h2></div></div><div class="quick-action-list"><a href="/admin/news/new"><span data-studio-icon="plus"></span><span><strong>Yeni haber yaz</strong><small>Taslak oluştur ve yayınla</small></span><b data-studio-icon="arrow"></b></a><a href="/admin/pages/new"><span data-studio-icon="plus"></span><span><strong>Yeni sayfa ekle</strong><small>Bloklarla içerik oluştur</small></span><b data-studio-icon="arrow"></b></a><a href="/admin/media"><span data-studio-icon="media"></span><span><strong>Medya yükle</strong><small>Görsel ve dosya yönet</small></span><b data-studio-icon="arrow"></b></a></div></article><article class="studio-card dashboard-card dashboard-health"><div class="dashboard-card-head"><div><span class="dashboard-section-kicker">Durum</span><h2>Çalışma ortamı</h2></div><span class="dashboard-health-dot"><i></i> Sağlıklı</span></div><dl><div><dt>Üyeler</dt><dd>{{.MemberStats.ActiveMembers}} aktif</dd></div><div><dt>Doğrulanmış</dt><dd>{{.MemberStats.VerifiedMembers}} hesap</dd></div><div><dt>Veri kaynağı</dt><dd>{{if .DataSource}}{{.DataSource}}{{else}}Proses içi{{end}}</dd></div></dl><a class="text-link" href="/admin/operations">Operasyon merkezine git <span data-studio-icon="arrow"></span></a></article></aside>
</section>`))

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	if err := AdminStart(w, "Dashboard", "", data.Nonce, data.Active); err != nil {
		return err
	}
	if err := adminDashboardWorkbenchTemplate.Execute(w, data); err != nil {
		return err
	}
	for _, collection := range data.Collections {
		if len(collection.Kits) == 0 || collection.ID == "commerce_variants" {
			continue
		}
		if _, err := io.WriteString(w, `<section class="studio-card studio-kit-summary"><h2>Etkin yetenekler · `+template.HTMLEscapeString(collection.Label)+`</h2>`); err != nil {
			return err
		}
		if err := adminstudio.RenderKitBadges(w, collection.Kits); err != nil {
			return err
		}
		if _, err := io.WriteString(w, `</section>`); err != nil {
			return err
		}
	}
	return AdminEnd(w)
}

func RenderAdminReports(w io.Writer, data AdminDashboardData) error {
	if err := AdminStart(w, "Raporlar", "Analitik ve Performans", data.Nonce, data.Active); err != nil {
		return err
	}
	if err := adminReportsKPIsTemplate.Execute(w, data); err != nil {
		return err
	}
	if err := adminReportsPathTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}

var adminReportsKPIsTemplate = template.Must(template.New("admin-reports-kpis").Funcs(template.FuncMap{
	"formatNumber": func(n any) string {
		return fmt.Sprintf("%v", n)
	},
}).Parse(`<div class="reports-header-bar">
	<div class="reports-filter-group">
		<label for="reports-date-filter"><strong>Zaman Aralığı:</strong></label>
		<select id="reports-date-filter" class="reports-select" onchange="if(window.studioNavigate){window.studioNavigate('/admin/reports?range='+this.value,true)}else{window.location.search='?range='+this.value}">
			<option value="today"{{if or (eq .DateFilter "today") (not .DateFilter)}} selected{{end}}>Canlı Proses (Çalışma Süresi)</option>
			<option value="7d"{{if eq .DateFilter "7d"}} selected{{end}}>Son 7 Gün (Kalıcı Veri [!] Gerektirir)</option>
			<option value="30d"{{if eq .DateFilter "30d"}} selected{{end}}>Son 30 Gün (Kalıcı Veri [!] Gerektirir)</option>
			<option value="all"{{if eq .DateFilter "all"}} selected{{end}}>Tüm Zamanlar (Canlı Proses Boyunca)</option>
		</select>
		<span class="reports-source-badge"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg> {{if .DataSource}}{{.DataSource}}{{else}}Proses İçi Bellek Telemetrisi{{end}}</span>
		{{if .DataTime}}<span class="reports-time-badge"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg> Veri Zamanı: {{.DataTime}}</span>{{end}}
	</div>
	<div class="reports-export-group">
		<a class="button button-quiet" href="/admin/reports/export?format=csv" download="hydro-report.csv"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg> <span>CSV Dışa Aktar</span></a>
		<a class="button button-quiet" href="/admin/reports/export?format=json" download="hydro-report.json" target="_blank"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg> <span>JSON Dışa Aktar</span></a>
	</div>
</div>

<div class="reports-notice-box">
	<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
	<div><strong>Gerçek Veri ve Kapsam Bilgisi:</strong> Raporlar geçerli sunucu prosesinin başlangıcından itibaren toplanan gerçek bellek içi (in-memory) telemetri verileridir (proses yeniden başlatıldığında sıfırlanır). Tarihsel 7g/30g zaman serisi filtreleme ve oturum bazlı kullanıcı aksiyon-hata korelasyonu kalıcı veritabanı şeması, saklama (retention) süresi ve KVKK/GDPR rıza kararı gerektirdiğinden ADR/TASKS.md kapsamında açık karar kapısı (gate) olarak tutulmaktadır.</div>
</div>

<section class="dashboard-kpis">
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-blue"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"></path></svg></span><span class="kpi-tag">HTTP</span></div>
		<span class="kpi-label">Toplam İstek</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong>
		<small>{{if .Metrics}}Anlık aktif: {{.Metrics.InFlight}}{{else}}İstek yok{{end}}</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-green"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></span><span class="kpi-tag kpi-tag-success">Görüntülenme</span></div>
		<span class="kpi-label">Sayfa Görüntülenme</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong>
		<small>Public &amp; mağaza sayfaları</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-purple"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg></span><span class="kpi-tag">Tekil</span></div>
		<span class="kpi-label">Benzersiz Ziyaretçi (kapalı)</span>
		<strong>{{if .Metrics}}{{.Metrics.UniqueVisitorsCount}}{{else}}0{{end}}</strong>
		<small>Açık rıza gerektiren analitik modülü etkin değil</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-purple"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg></span><span class="kpi-tag">Gecikme</span></div>
		<span class="kpi-label">Ortalama Yanıt Süresi</span>
		<strong>{{if .Metrics}}{{.Metrics.AvgLatencyMS}} ms{{else}}0 ms{{end}}</strong>
		<small>Süreç içi ortalama latency</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon {{if and .Metrics (gt .Metrics.TotalErrors 0)}}kpi-blue{{else}}kpi-green{{end}}"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"></polygon><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg></span><span class="kpi-tag {{if and .Metrics (gt .Metrics.TotalErrors 0)}}kpi-tag-danger{{else}}kpi-tag-success{{end}}">{{if and .Metrics (gt .Metrics.TotalErrors 0)}}Uyarı{{else}}Sağlıklı{{end}}</span></div>
		<span class="kpi-label">Hatalı İstekler</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</strong>
		<small>4xx ve 5xx hata yanıtları</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-blue"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><polyline points="16 11 18 13 22 9"></polyline></svg></span><span class="kpi-tag">Üyelik</span></div>
		<span class="kpi-label">Kayıtlı Üyeler</span>
		<strong>{{.MemberStats.TotalMembers}}</strong>
		<small>{{.MemberStats.ActiveMembers}} aktif, {{.MemberStats.VerifiedMembers}} e-posta onaylı</small>
	</article>
</section>`))

var adminReportsPathTemplate = template.Must(template.New("admin-reports-paths").Parse(`<section class="dashboard-grid">
	<article class="studio-card dashboard-card">
		<div class="dashboard-card-head">
			<div class="card-title-group">
				<h2>En Çok Görüntülenen Rotalar</h2>
				<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg><span class="studio-tip-popover"><span class="tip-eyebrow">EN ÇOK OKUNAN</span><p>Ziyaretçiler tarafından en çok görüntülenen sayfaların gerçek zamanlı dağılımı.</p></span></span>
			</div>
			<span class="muted">Görüntülenmeye göre sıralı</span>
		</div>
		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Rota</th>
						<th>Görüntülenme</th>
						<th>Toplam İstek</th>
						<th>Ort. Süre</th>
					</tr>
				</thead>
				<tbody>
					{{if .Metrics}}
						{{range .Metrics.TopPages 5}}
							<tr>
								<td><code>{{.Path}}</code></td>
								<td><strong>{{.Views}}</strong></td>
								<td>{{.Requests}}</td>
								<td>{{.AvgMS}} ms</td>
							</tr>
						{{else}}
							<tr><td colspan="4" class="empty">Henüz görüntülenme verisi kaydedilmedi.</td></tr>
						{{end}}
					{{else}}
						<tr><td colspan="4" class="empty">Metrik toplayıcı bağlı değil.</td></tr>
					{{end}}
				</tbody>
			</table>
		</div>
	</article>

	<article class="studio-card dashboard-card">
		<div class="dashboard-card-head">
			<div class="card-title-group">
				<h2>En Çok Hata Alan Rotalar</h2>
				<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg><span class="studio-tip-popover"><span class="tip-eyebrow">HATA FREKANSI</span><p>HTTP 4xx ve 5xx durum kodu döndüren endpoint'lerin hata frekansı dağılımı.</p></span></span>
			</div>
			<span class="muted">Hata sayısına göre sıralı</span>
		</div>
		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Rota</th>
						<th>Hata Sayısı</th>
						<th>Toplam İstek</th>
						<th>Durum</th>
					</tr>
				</thead>
				<tbody>
					{{if .Metrics}}
						{{range .Metrics.TopErrors 5}}
							<tr>
								<td><code>{{.Path}}</code></td>
								<td><strong style="color:var(--red)">{{.Errors}}</strong></td>
								<td>{{.Requests}}</td>
								<td><span class="status status-danger">İncelenmeli</span></td>
							</tr>
						{{else}}
							<tr><td colspan="4" class="empty">Hata kaydedilen rota bulunmuyor (Sistem kusursuz çalışıyor).</td></tr>
						{{end}}
					{{else}}
						<tr><td colspan="4" class="empty">Metrik toplayıcı bağlı değil.</td></tr>
					{{end}}
				</tbody>
			</table>
		</div>
	</article>
</section>

<section class="studio-card dashboard-card report-data-card">
	<div class="dashboard-card-head">
		<div class="card-title-group">
			<h2>Tüm Rotalar ve Performans Analizi</h2>
			<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg><span class="studio-tip-popover"><span class="tip-eyebrow">DETAYLI GÖZLEM</span><p>Canlı istek alan tüm endpoint'lerin gecikme, istek ve hata oranları tablosu.</p></span></span>
		</div>
		<span class="muted">Filtreleyin ve arayın</span>
	</div>
	<div class="report-filters">
		<label>Rota ara <input id="report-path-filter" type="search" placeholder="/public, /shop, /member…"></label>
		<label>Göster <select id="report-kind-filter"><option value="all">Tümü</option><option value="views">Görüntülenme Olanlar</option><option value="errors">Hata Alanlar</option></select></label>
	</div>
	<div class="table-wrap">
		<table id="report-path-table">
			<thead>
				<tr>
					<th>Rota</th>
					<th>Görüntülenme</th>
					<th>Toplam İstek</th>
					<th>Hata</th>
					<th>Ort. Yanıt Süresi</th>
				</tr>
			</thead>
			<tbody>
				{{if .Metrics}}
					{{range .Metrics.PathStats}}
						<tr data-report-row data-path="{{.Path}}" data-views="{{.Views}}" data-errors="{{.Errors}}">
							<td><code>{{.Path}}</code></td>
							<td>{{.Views}}</td>
							<td>{{.Requests}}</td>
							<td>{{if gt .Errors 0}}<strong style="color:var(--red)">{{.Errors}}</strong>{{else}}0{{end}}</td>
							<td>{{.AvgMS}} ms</td>
						</tr>
					{{else}}
						<tr><td colspan="5" class="empty">Henüz rota verisi kaydedilmedi.</td></tr>
					{{end}}
				{{else}}
					<tr><td colspan="5" class="empty">Ölçüm sağlayıcısı bağlı değil.</td></tr>
				{{end}}
			</tbody>
		</table>
	</div>
</section>
<script nonce="{{.Nonce}}">(()=>{const path=document.getElementById('report-path-filter'),kind=document.getElementById('report-kind-filter');if(!path||!kind)return;const apply=()=>document.querySelectorAll('[data-report-row]').forEach(row=>{const match=row.dataset.path.toLowerCase().includes(path.value.toLowerCase());const visible=kind.value==='views'?Number(row.dataset.views)>0:kind.value==='errors'?Number(row.dataset.errors)>0:true;row.hidden=!(match&&visible)});path.addEventListener('input',apply);kind.addEventListener('change',apply)})()</script>`))
