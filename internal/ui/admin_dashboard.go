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

var adminDashboardTemplate = template.Must(template.New("admin-dashboard").Parse(`<section class="dashboard-welcome"><div><p class="eyebrow">YÖNETİM MERKEZİ</p><h2>İçerik ve commerce kontrolü.</h2><p class="dashboard-subtitle">{{.Email}} · Bugün çalışma alanınızda neler olduğunu tek bakışta görün.</p></div><div class="studio-actions"><a class="button" href="/admin/news#new-news">+ Yeni haber</a><a class="button button-quiet" href="/public/news/tr/html">Canlı site ↗</a></div></section>
<section class="dashboard-kpis"><article class="dashboard-kpi"><span class="kpi-icon kpi-blue">◈</span><span class="kpi-label">İçerik koleksiyonu</span><strong>{{len .Collections}}</strong><small>aktif modül</small></article><article class="dashboard-kpi"><span class="kpi-icon kpi-purple">▤</span><span class="kpi-label">Yayın yüzeyi</span><strong>REST + GraphQL</strong><small>headless hazır</small></article><article class="dashboard-kpi"><span class="kpi-icon kpi-green">✓</span><span class="kpi-label">Sistem durumu</span><strong class="healthy">Operational</strong><small>health ve ready kontrolü</small></article></section>
<section class="dashboard-grid"><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">HIZLI ERİŞİM</p><h2>Çalışma alanları</h2></div><span class="muted">Modül bazlı yönetim</span></div><div class="collection-grid">{{range .Collections}}<a class="collection-tile" href="{{(index .Actions 0).Href}}"><span class="tile-icon">{{if eq .ID "news"}}◈{{else if eq .ID "pages"}}▤{{else}}◫{{end}}</span><span><strong>{{.Label}}</strong><small>{{.Description}}</small></span><b>→</b></a>{{end}}</div></article><article class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">RAPORLAR</p><h2>Operasyon özeti</h2></div><a class="text-link" href="/admin/reports">Tüm raporlar →</a></div><div class="report-list"><div><span class="report-bar bar-blue" style="width:82%"></span><span>Yayınlanan içerikler</span><strong>82%</strong></div><div><span class="report-bar bar-purple" style="width:64%"></span><span>Commerce katalog doluluğu</span><strong>64%</strong></div><div><span class="report-bar bar-green" style="width:94%"></span><span>Platform sağlığı</span><strong>94%</strong></div></div><p class="muted report-note">Bu özet canlı sistem kontrolleri ve kayıtlı koleksiyon metadatası için başlangıç görünümüdür.</p></article></section>
<section class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">SİSTEM</p><h2>Bağlantılar ve araçlar</h2></div></div><div class="tool-grid"><a class="tool-tile" href="/api/v1/commerce/products"><strong>Commerce REST API</strong><small>Ürün ve katalog verisini incele</small><b>↗</b></a><a class="tool-tile" href="/public/news/tr/html"><strong>Newsroom frontend</strong><small>Son kullanıcı görünümünü aç</small><b>↗</b></a><a class="tool-tile" href="/api/graphql"><strong>GraphQL endpoint</strong><small>Headless sorgu yüzeyi</small><b>↗</b></a><a class="tool-tile" href="/healthz"><strong>Health check</strong><small>Servis sağlık kontrolü</small><b>↗</b></a></div></section>`))

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	if err := AdminStart(w, "Dashboard", "Yönetim merkezi", data.Nonce, data.Active); err != nil {
		return err
	}
	if err := adminDashboardTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}
