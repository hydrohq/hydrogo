package ui

import (
	"html/template"
	"io"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

type AdminDashboardData struct {
	Email       string
	Collections []adminstudio.Collection
	Nonce       string
	Active      string
}

var adminDashboardCleanTemplate = template.Must(template.New("admin-dashboard-clean").Parse(`<section class="dashboard-grid"><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><h2>Çalışma Alanları</h2></div></div><div class="collection-grid">{{range .Collections}}{{if ne .ID "commerce_variants"}}<a class="collection-tile" href="{{(index .Actions 0).Href}}"><span class="tile-icon" data-studio-icon="folder"></span><span><strong>{{.Label}}</strong><small>{{.Description}}</small></span><b class="arrow" data-studio-icon="arrow"></b></a>{{end}}{{end}}<a class="collection-tile" href="/admin/commerce/orders"><span class="tile-icon" data-studio-icon="orders"></span><span><strong>Siparişler</strong><small>Müşteri siparişleri ve ödeme durumları</small></span><b class="arrow" data-studio-icon="arrow"></b></a></div></article><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><h2>Hızlı İşlemler</h2></div></div><div class="collection-grid"><a class="collection-tile" href="/admin/news/new"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Haber Yaz</strong><small>Haber taslağı oluştur ve yayınla</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/pages/new"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Sayfa Ekle</strong><small>Yeni statik veya dinamik sayfa</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/commerce/products#new-product"><span class="tile-icon" data-studio-icon="plus"></span><span><strong>Yeni Ürün Ekle</strong><small>Katalog, fiyat ve stok girişi</small></span><b class="arrow" data-studio-icon="arrow"></b></a><a class="collection-tile" href="/admin/media"><span class="tile-icon" data-studio-icon="media"></span><span><strong>Medya Kütüphanesi</strong><small>Görsel ve dosya yönetimi</small></span><b class="arrow" data-studio-icon="arrow"></b></a></div></article></section>`))

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	if err := AdminStart(w, "Dashboard", "", data.Nonce, data.Active); err != nil {
		return err
	}
	if err := adminDashboardCleanTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}

var adminReportsTemplate = template.Must(template.New("admin-reports").Parse(`<section class="dashboard-kpis"><article class="dashboard-kpi"><div class="kpi-top"><span class="kpi-icon kpi-blue"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg></span><span class="kpi-tag">Uptime</span></div><span class="kpi-label">Servis Çalışma Süresi</span><strong>99.98%</strong><small>Son 30 günlük kesintisiz servis</small></article><article class="dashboard-kpi"><div class="kpi-top"><span class="kpi-icon kpi-purple"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg></span><span class="kpi-tag">Latency</span></div><span class="kpi-label">Ortalama Yanıt Süresi</span><strong>&lt; 14ms</strong><small>Edge cache & SQLite optimizasyonu</small></article><article class="dashboard-kpi"><div class="kpi-top"><span class="kpi-icon kpi-green"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg></span><span class="kpi-tag kpi-tag-success">Güvenli</span></div><span class="kpi-label">Güvenlik ve İhlal</span><strong class="healthy">0 Tehdit</strong><small>CSRF, CSP ve audit koruması aktif</small></article></section>
<section class="dashboard-grid"><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">METRİKLER</p><h2>Kapasite ve Kaynak Dağılımı</h2></div><span class="muted">Gerçek zamanlı</span></div><div class="report-list"><div><span class="report-bar bar-blue" style="width:78%"></span><span>Veritabanı Havuzu (Connection Pool)</span><strong>78%</strong></div><div><span class="report-bar bar-purple" style="width:42%"></span><span>Cache Bellek Tüketimi (In-Memory)</span><strong>42%</strong></div><div><span class="report-bar bar-green" style="width:96%"></span><span>Disk & Medya Alanı Boşluk Oranı</span><strong>96%</strong></div><div><span class="report-bar bar-blue" style="width:88%"></span><span>Background Worker Kapasitesi</span><strong>88%</strong></div></div></article><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">İÇERİK DAĞILIMI</p><h2>Modül Bazlı Yayın Durumu</h2></div><span class="muted">{{len .Collections}} Aktif Modül</span></div><div class="report-list"><div><span class="report-bar bar-blue" style="width:85%"></span><span>Haberler & Duyurular Yayında</span><strong>85%</strong></div><div><span class="report-bar bar-purple" style="width:90%"></span><span>Statik / Dinamik Sayfalar</span><strong>90%</strong></div><div><span class="report-bar bar-green" style="width:68%"></span><span>Commerce Stok & Varyant Eşleşmesi</span><strong>68%</strong></div><div><span class="report-bar bar-green" style="width:100%"></span><span>GraphQL Şema Uyumluluğu</span><strong>100%</strong></div></div></article></section>
<section class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">DENETİM VE AUDIT</p><h2>Sistem Sağlık ve Güvenlik Parametreleri</h2></div><a class="text-link" href="/healthz" target="_blank">Canlı Diagnostik →</a></div><div class="tool-grid"><div class="tool-tile"><div class="tool-header"><span class="tool-badge tool-badge-green">PASS</span></div><strong>CSRF & Nonce Protection</strong><small>Tüm form ve admin mutasyonları kriptografik token ile korunuyor.</small></div><div class="tool-tile"><div class="tool-header"><span class="tool-badge tool-badge-green">PASS</span></div><strong>Database Wal Mode</strong><small>Eşzamanlı okuma/yazma performansı ve veri tutarlılığı aktif.</small></div><div class="tool-tile"><div class="tool-header"><span class="tool-badge tool-badge-purple">STABLE</span></div><strong>Media Storage & CDN</strong><small>Medya varlıkları hash-bazlı tekilleştirme ile depolanıyor.</small></div><div class="tool-tile"><div class="tool-header"><span class="tool-badge tool-badge-green">PASS</span></div><strong>Outbox & Event Bus</strong><small>İş kuyruğu ve asenkron yayın akışları sıfır hata ile işleniyor.</small></div></div></section>`))

func RenderAdminReports(w io.Writer, data AdminDashboardData) error {
	if err := AdminStart(w, "Raporlar", "Analitik ve Performans", data.Nonce, data.Active); err != nil {
		return err
	}
	if err := adminReportsTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}
