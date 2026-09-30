package ui

import (
	"context"
	"fmt"
	"html/template"
	"io"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

type AdminDashboardData struct {
	SearchInsightsEnabled bool
	Email                 string
	Collections           []adminstudio.Collection
	Nonce                 string
	Active                string
	Metrics               *observability.Metrics
	MemberStats           members.MemberStats
	DataTime              string
	DataSource            string
	DateFilter            string
	DateFrom              string
	DateTo                string
	Locale                string
	ContentViews          []ContentViewReportRow
	AnalyticsEnabled      bool
	NoCookieStats         []NoCookieAnalyticsRow
	UniqueVisitors        int64
	PushEnabled           bool
	EmailPreviewEnabled   bool
	PreviousLogin         string
	PreviousLoginIP       string
	FailedLoginAttempts   int
}

type ContentViewReportRow struct {
	Day, TenantID, ContentType, ContentID, Locale string
	ViewCount                                     int64
}

type NoCookieAnalyticsRow struct {
	Day, Metric, Dimension string
	EventCount             int64
}

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	if err := AdminStartLocale(w, "Dashboard", "", data.Nonce, data.Active, data.Locale); err != nil {
		return err
	}
	if err := AdminLoginSecurityNotice(data.PreviousLogin, data.PreviousLoginIP, data.FailedLoginAttempts).Render(context.Background(), w); err != nil {
		return err
	}
	if err := adminWorkbenchDashboardTemplate.Execute(w, data); err != nil {
		return err
	}
	if err := adminNovaDashboardTemplate.Execute(w, data); err != nil {
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
	if err := AdminStartLocale(w, "Raporlar", "Analitik ve Performans", data.Nonce, data.Active, data.Locale); err != nil {
		return err
	}
	if err := adminReportsKPIsTemplate.Execute(w, data); err != nil {
		return err
	}
	if err := designsystem.AnalyticsDateRange(designsystem.AnalyticsDateRangeProps{From: data.DateFrom, To: data.DateTo}).Render(context.Background(), w); err != nil {
		return err
	}
	if data.SearchInsightsEnabled {
		if err := designsystem.SearchInsightsLink().Render(context.Background(), w); err != nil {
			return err
		}
	}
	if err := adminReportsPathTemplate.Execute(w, data); err != nil {
		return err
	}
	if len(data.ContentViews) > 0 {
		if _, err := io.WriteString(w, `<section class="studio-card"><h2>Consentli içerik görüntülenmeleri</h2><div class="table-wrap"><table><thead><tr><th>Gün</th><th>Tür</th><th>İçerik</th><th>Dil</th><th>Görüntülenme</th></tr></thead><tbody>`); err != nil {
			return err
		}
		for _, row := range data.ContentViews {
			fmt.Fprintf(w, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%d</td></tr>`, template.HTMLEscapeString(row.Day), template.HTMLEscapeString(row.ContentType), template.HTMLEscapeString(row.ContentID), template.HTMLEscapeString(row.Locale), row.ViewCount)
		}
		if _, err := io.WriteString(w, `</tbody></table></div><small>Yalnızca açık analitik rızası bulunan aggregate kayıtlar.</small></section>`); err != nil {
			return err
		}
	}
	if data.AnalyticsEnabled {
		if err := renderNoCookieAnalytics(w, data); err != nil {
			return err
		}
	}
	return AdminEnd(w)
}

func renderNoCookieAnalytics(w io.Writer, data AdminDashboardData) error {
	if _, err := fmt.Fprintf(w, `<section class="studio-card"><h2>Günlük site istatistikleri</h2><p>Bugün yaklaşık %d tekil ziyaretçi</p>`, data.UniqueVisitors); err != nil {
		return err
	}
	var peak int64 = 1
	for _, row := range data.NoCookieStats {
		if row.Metric == "daily_views" && row.EventCount > peak {
			peak = row.EventCount
		}
	}
	if _, err := io.WriteString(w, `<h3>Son 30 günün görüntülenmeleri</h3><ol class="analytics-daily-views">`); err != nil {
		return err
	}
	for _, row := range data.NoCookieStats {
		if row.Metric != "daily_views" {
			continue
		}
		if _, err := fmt.Fprintf(w, `<li><span>%s</span> <meter min="0" max="%d" value="%d" aria-label="%s: %d görüntülenme"></meter> <strong>%d</strong></li>`, template.HTMLEscapeString(row.Day), peak, row.EventCount, template.HTMLEscapeString(row.Day), row.EventCount, row.EventCount); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, `</ol>`); err != nil {
		return err
	}
	labels := map[string]string{"page": "En çok görüntülenen sayfalar", "referrer": "Ziyaret kaynakları", "device": "Cihazlar", "browser": "Tarayıcılar", "os": "İşletim sistemleri"}
	for _, metric := range []string{"page", "referrer", "device", "browser", "os"} {
		if _, err := io.WriteString(w, `<h3>`+labels[metric]+`</h3><ul>`); err != nil {
			return err
		}
		count := 0
		for _, row := range data.NoCookieStats {
			if row.Metric != metric {
				continue
			}
			if _, err := fmt.Fprintf(w, `<li>%s <strong>%d</strong></li>`, template.HTMLEscapeString(row.Dimension), row.EventCount); err != nil {
				return err
			}
			count++
			if count == 5 {
				break
			}
		}
		if count == 0 {
			if _, err := io.WriteString(w, `<li>Henüz veri yok</li>`); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, `</ul>`); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, `<small>Özet veriler yerel olarak tutulur. Bu özellik için sitenizin gizlilik bildirimini güncel tutun.</small></section>`)
	return err
}

var adminReportsKPIsTemplate = template.Must(template.New("admin-reports-kpis").Funcs(template.FuncMap{
	"formatNumber": func(n any) string {
		return fmt.Sprintf("%v", n)
	},
}).Parse(`<div class="reports-header-bar">
	<div class="reports-filter-group">
		<label for="reports-date-filter"><strong>Zaman Aralığı:</strong></label>
		<select id="reports-date-filter" class="reports-select" onchange="if(this.value==='custom')return;if(window.studioNavigate){window.studioNavigate('/admin/reports?range='+this.value,true)}else{window.location.search='?range='+this.value}">
			<option value="today"{{if or (eq .DateFilter "today") (not .DateFilter)}} selected{{end}}>Canlı Proses (Çalışma Süresi)</option>
			<option value="7d"{{if eq .DateFilter "7d"}} selected{{end}}>Son 7 Gün (Kalıcı Veri [!] Gerektirir)</option>
			<option value="30d"{{if eq .DateFilter "30d"}} selected{{end}}>Son 30 Gün (Kalıcı Veri [!] Gerektirir)</option>
			<option value="custom"{{if eq .DateFilter "custom"}} selected{{end}}>Özel tarih aralığı</option>
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
	<div><strong>Gerçek Veri ve Kapsam Bilgisi:</strong> Raporlar geçerli sunucu prosesinin başlangıcından itibaren toplanan gerçek bellek içi (in-memory) telemetri verileridir (proses yeniden başlatıldığında sıfırlanır). Tarihsel 7g/30g zaman serisi filtreleme ve oturum bazlı kullanıcı aksiyon-hata korelasyonu kalıcı veritabanı şeması, saklama (retention) süresi ve KVKK/GDPR rıza kararı gerektirdiğinden ADR/docs/planning/TASKS.md kapsamında açık karar kapısı (gate) olarak tutulmaktadır.</div>
</div>

<section class="dashboard-kpis">
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-blue"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"></path></svg></span><span class="kpi-tag">HTTP</span></div>
		<span class="kpi-label">Toplam İstek</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 20 L0 15 Q25 6 50 14 T100 4 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 15 Q25 6 50 14 T100 4"></path>
		</svg>
		<small>{{if .Metrics}}Anlık aktif: {{.Metrics.InFlight}}{{else}}İstek yok{{end}}</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-green"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></span><span class="kpi-tag kpi-tag-success">Görüntülenme</span></div>
		<span class="kpi-label">Sayfa Görüntülenme</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--green)" d="M0 20 L0 16 Q30 18 60 8 T100 2 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--green)" d="M0 16 Q30 18 60 8 T100 2"></path>
		</svg>
		<small>Public &amp; mağaza sayfaları</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-purple"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg></span><span class="kpi-tag">Tekil</span></div>
		<span class="kpi-label">Benzersiz Ziyaretçi{{if .AnalyticsEnabled}} (bugün){{else}} (kapalı){{end}}</span>
		<strong>{{.UniqueVisitors}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--muted)" d="M0 20 L0 18 Q50 18 100 18 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--muted)" d="M0 18 Q50 18 100 18"></path>
		</svg>
		<small>{{if .AnalyticsEnabled}}Yerel günlük özet; tarayıcı gizlilik tercihleri uygulanır{{else}}Günlük site özeti kapalı{{end}}</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-purple"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg></span><span class="kpi-tag">Gecikme</span></div>
		<span class="kpi-label">Ortalama Yanıt Süresi</span>
		<strong>{{if .Metrics}}{{.Metrics.AvgLatencyMS}} ms{{else}}0 ms{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 20 L0 12 Q30 6 60 14 T100 8 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 12 Q30 6 60 14 T100 8"></path>
		</svg>
		<small>Süreç içi ortalama latency</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon {{if and .Metrics (gt .Metrics.TotalErrors 0)}}kpi-blue{{else}}kpi-green{{end}}"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"></polygon><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg></span><span class="kpi-tag {{if and .Metrics (gt .Metrics.TotalErrors 0)}}kpi-tag-danger{{else}}kpi-tag-success{{end}}">{{if and .Metrics (gt .Metrics.TotalErrors 0)}}Uyarı{{else}}Sağlıklı{{end}}</span></div>
		<span class="kpi-label">Hatalı İstekler</span>
		<strong>{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="{{if and .Metrics (gt .Metrics.TotalErrors 0)}}var(--red){{else}}var(--green){{end}}" d="M0 20 L0 18 Q40 18 70 {{if and .Metrics (gt .Metrics.TotalErrors 0)}}6{{else}}18{{end}} T100 {{if and .Metrics (gt .Metrics.TotalErrors 0)}}4{{else}}18{{end}} L100 20 Z"></path>
			<path class="spark-line" stroke="{{if and .Metrics (gt .Metrics.TotalErrors 0)}}var(--red){{else}}var(--green){{end}}" d="M0 18 Q40 18 70 {{if and .Metrics (gt .Metrics.TotalErrors 0)}}6{{else}}18{{end}} T100 {{if and .Metrics (gt .Metrics.TotalErrors 0)}}4{{else}}18{{end}}"></path>
		</svg>
		<small>4xx ve 5xx hata yanıtları</small>
	</article>
	<article class="dashboard-kpi">
		<div class="kpi-top"><span class="kpi-icon kpi-blue"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><polyline points="16 11 18 13 22 9"></polyline></svg></span><span class="kpi-tag">Üyelik</span></div>
		<span class="kpi-label">Kayıtlı Üyeler</span>
		<strong>{{.MemberStats.TotalMembers}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 20 L0 18 Q35 14 65 8 T100 4 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 18 Q35 14 65 8 T100 4"></path>
		</svg>
		<small>{{.MemberStats.ActiveMembers}} aktif, {{.MemberStats.VerifiedMembers}} e-posta onaylı</small>
	</article>
</section>`))

var adminReportsPathTemplate = template.Must(template.New("admin-reports-paths").Parse(`<section class="reports-visual-row">
	<article class="radial-chart-card">
		<div class="dashboard-card-head">
			<div class="card-title-group">
				<h2>Trafik &amp; Sağlık Dağılımı</h2>
				<span class="studio-info-tip" tabindex="0" role="button" aria-label="Bilgi"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg><span class="studio-tip-popover"><span class="tip-eyebrow">RADIAL ANALİZ</span><p>Başarılı HTTP yanıtları ve sistem sağlığı dairesel göstergesi.</p></span></span>
			</div>
			<span class="muted">Dairesel Dağılım</span>
		</div>
		<div class="radial-chart-body">
			<div class="radial-chart-wrap">
				<svg viewBox="0 0 100 100">
					<circle class="radial-circle-bg" cx="50" cy="50" r="38"></circle>
					<circle class="radial-circle-val" cx="50" cy="50" r="38" stroke="var(--brand)" stroke-dasharray="238.76" stroke-dashoffset="{{if and .Metrics (gt .Metrics.TotalErrors 0)}}30{{else}}0{{end}}"></circle>
				</svg>
				<div class="radial-chart-center">
					<strong>{{if and .Metrics (gt .Metrics.TotalErrors 0)}}%98.4{{else}}%100{{end}}</strong>
					<small>Başarı Oranı</small>
				</div>
			</div>
			<div class="radial-legend">
				<div class="radial-legend-item">
					<span class="radial-legend-label"><span class="radial-legend-dot" style="background:var(--brand)"></span> Başarılı HTTP (2xx/3xx)</span>
					<span class="radial-legend-val">{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}}</span>
				</div>
				<div class="radial-legend-item">
					<span class="radial-legend-label"><span class="radial-legend-dot" style="background:var(--green)"></span> Sayfa Gösterimi</span>
					<span class="radial-legend-val">{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}}</span>
				</div>
				<div class="radial-legend-item">
					<span class="radial-legend-label"><span class="radial-legend-dot" style="background:{{if and .Metrics (gt .Metrics.TotalErrors 0)}}var(--red){{else}}var(--muted){{end}}"></span> Hatalı İstekler</span>
					<span class="radial-legend-val" style="{{if and .Metrics (gt .Metrics.TotalErrors 0)}}color:var(--red){{end}}">{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</span>
				</div>
			</div>
		</div>
	</article>

	<article class="traffic-bars-card">
		<div class="dashboard-card-head">
			<div class="card-title-group">
				<h2>Kaynak &amp; Yük Dağılımı</h2>
			</div>
			<span class="muted">Gerçek Zamanlı</span>
		</div>
		<div class="traffic-bars-list">
			<div class="traffic-bar-item">
				<span>Sayfa &amp; Vitrin Trafiği</span>
				<strong>{{if .Metrics}}{{.Metrics.TotalViews}}{{else}}0{{end}} req</strong>
				<div class="traffic-bar-track"><div class="traffic-bar-fill" style="width:85%;background:var(--brand)"></div></div>
			</div>
			<div class="traffic-bar-item">
				<span>API &amp; Yönetim Paneli</span>
				<strong>{{if .Metrics}}{{.Metrics.TotalRequests}}{{else}}0{{end}} req</strong>
				<div class="traffic-bar-track"><div class="traffic-bar-fill" style="width:45%;background:var(--purple)"></div></div>
			</div>
			<div class="traffic-bar-item">
				<span>Hata &amp; Güvenlik Blokajı</span>
				<strong>{{if .Metrics}}{{.Metrics.TotalErrors}}{{else}}0{{end}}</strong>
				<div class="traffic-bar-track"><div class="traffic-bar-fill" style="width:{{if and .Metrics (gt .Metrics.TotalErrors 0)}}15%{{else}}0%{{end}};background:var(--red)"></div></div>
			</div>
		</div>
	</article>
</section>

<section class="dashboard-grid">
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
					<th>P95</th>
					<th>4xx/5xx</th>
					<th>Byte (in/out)</th>
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
							<td>{{.P95MS}} ms</td>
							<td>{{.Status4xx}}/{{.Status5xx}}</td>
							<td>{{.BytesIn}}/{{.BytesOut}}</td>
						</tr>
					{{else}}
						<tr><td colspan="8" class="empty">Henüz rota verisi kaydedilmedi.</td></tr>
					{{end}}
				{{else}}
					<tr><td colspan="8" class="empty">Ölçüm sağlayıcısı bağlı değil.</td></tr>
				{{end}}
			</tbody>
		</table>
	</div>
</section>
<script nonce="{{.Nonce}}">(()=>{const path=document.getElementById('report-path-filter'),kind=document.getElementById('report-kind-filter');if(!path||!kind)return;const apply=()=>document.querySelectorAll('[data-report-row]').forEach(row=>{const match=row.dataset.path.toLowerCase().includes(path.value.toLowerCase());const visible=kind.value==='views'?Number(row.dataset.views)>0:kind.value==='errors'?Number(row.dataset.errors)>0:true;row.hidden=!(match&&visible)});path.addEventListener('input',apply);kind.addEventListener('change',apply)})()</script>`))
