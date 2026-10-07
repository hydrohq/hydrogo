package ui

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type AdminAreaData struct {
	Email                    string
	Nonce                    string
	CSRF                     string
	Area                     string
	Locale                   string
	TenantID                 string
	LocaleSaved              bool
	SoundAlertsEnabled       bool
	FontToolsEnabled         bool
	Users                    []*auth.User
	EmailVerificationEnabled bool
	PasswordResetLinkEnabled bool
	Permissions              []string
	IPBans                   []SecurityBanRow
}

type SecurityBanRow struct{ Address, Expires string }

type passwordStrengthWriter struct{ io.Writer }

func (w passwordStrengthWriter) Write(data []byte) (int, error) {
	markup := string(data)
	var component bytes.Buffer
	if err := designsystem.PasswordStrengthMeter().Render(context.Background(), &component); err == nil {
		markup = strings.Replace(markup, `name="new_password" minlength="8" autocomplete="new-password"`, `name="new_password" minlength="8" autocomplete="new-password" data-password-strength`, 1)
		markup = strings.Replace(markup, `autocomplete="new-password"></label>`, `autocomplete="new-password"></label>`+component.String(), 1)
	}
	return w.Writer.Write([]byte(markup))
}

type AdminResourceUsageData struct {
	Email    string
	Nonce    string
	Snapshot observability.RuntimeSnapshot
	Locale   string
}

var adminResourceUsageTemplate = template.Must(template.New("admin-resource-usage").Parse(`<section class="resource-page" data-testid="resource-page">
<div class="resource-banner"><div><span class="eyebrow">CANLI PROCESS GÖRÜNÜMÜ</span><h2>Sistem kaynakları</h2><p>Bu ekran yalnızca mevcut process'in ölçebildiği anlık değerleri gösterir. Tarihsel veri, host CPU yüzdesi ve disk telemetrisi bu uygulama tarafından uydurulmaz.</p></div><div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;"><span class="resource-status {{if .Snapshot.ReadinessState}}is-ready{{else}}is-waiting{{end}}"><i></i>{{if .Snapshot.ReadinessState}}Hazır{{else}}Hazırlanıyor{{end}}</span><a href="/admin/developer/resources" class="button button-quiet" style="font-size:.76rem;height:28px;min-height:28px;">Yenile ↻</a></div></div>
<div class="resource-kpis">
	<div class="resource-kpi">
		<small>Heap kullanımı</small>
		<strong>{{.HeapAlloc}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 20 L0 14 Q25 6 50 12 T100 4 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 14 Q25 6 50 12 T100 4"></path>
		</svg>
		<span>{{.HeapInUse}} kullanım alanı</span>
	</div>
	<div class="resource-kpi">
		<small>Goroutine</small>
		<strong>{{.Snapshot.Goroutines}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--green)" d="M0 20 L0 16 Q35 12 70 8 T100 2 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--green)" d="M0 16 Q35 12 70 8 T100 2"></path>
		</svg>
		<span>{{.Snapshot.GOMAXPROCS}} paralel çalışma limiti</span>
	</div>
	<div class="resource-kpi">
		<small>İstek / gecikme</small>
		<strong>{{.Snapshot.Requests}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--purple, #a855f7)" d="M0 20 L0 10 Q30 18 60 8 T100 4 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--purple, #a855f7)" d="M0 10 Q30 18 60 8 T100 4"></path>
		</svg>
		<span>Ortalama {{.Snapshot.AvgLatencyMS}} ms</span>
	</div>
	<div class="resource-kpi">
		<small>Uptime</small>
		<strong>{{.Uptime}}</strong>
		<svg class="wb-sparkline" viewBox="0 0 100 20" preserveAspectRatio="none">
			<path class="spark-area" fill="var(--brand)" d="M0 20 L0 15 Q40 10 70 12 T100 6 L100 20 Z"></path>
			<path class="spark-line" stroke="var(--brand)" d="M0 15 Q40 10 70 12 T100 6"></path>
		</svg>
		<span>Başlangıç: {{.StartedAt}}</span>
	</div>
</div>
<div class="resource-grid"><article class="studio-card"><div class="resource-card-title"><h3>Process belleği</h3><span class="resource-badge">runtime.ReadMemStats</span></div><dl class="resource-list"><dt>Heap alloc</dt><dd>{{.HeapAlloc}}</dd><dt>Heap in-use</dt><dd>{{.HeapInUse}}</dd><dt>Heap system</dt><dd>{{.HeapSys}}</dd><dt>Heap objects</dt><dd>{{.Snapshot.HeapObjects}}</dd></dl></article><article class="studio-card"><div class="resource-card-title"><h3>HTTP ve trafik</h3><span class="resource-badge">Process ömrü</span></div><dl class="resource-list"><dt>Toplam istek</dt><dd>{{.Snapshot.Requests}}</dd><dt>Aktif istek</dt><dd>{{.Snapshot.InFlight}}</dd><dt>Sayfa görüntüleme</dt><dd>{{.Snapshot.Views}}</dd><dt>4xx / 5xx hata</dt><dd>{{.Snapshot.Errors}}</dd><dt>Benzersiz ziyaretçi</dt><dd>{{.Snapshot.UniqueVisitors}}</dd></dl></article>{{if .Snapshot.HasDatabase}}<article class="studio-card"><div class="resource-card-title"><h3>Veritabanı havuzu</h3><span class="resource-badge">sql.DB</span></div><dl class="resource-list"><dt>Açık bağlantı</dt><dd>{{.Snapshot.Database.OpenConnections}} / {{.Snapshot.Database.MaxOpenConnections}}</dd><dt>Kullanımda</dt><dd>{{.Snapshot.Database.InUseConnections}}</dd><dt>Boşta</dt><dd>{{.Snapshot.Database.IdleConnections}}</dd><dt>Bekleme sayısı</dt><dd>{{.Snapshot.Database.WaitCount}}</dd></dl></article>{{end}}{{if .Snapshot.HasCache}}<article class="studio-card"><div class="resource-card-title"><h3>Cache</h3><span class="resource-badge">public content</span></div><dl class="resource-list"><dt>Hit / miss</dt><dd>{{.Snapshot.Cache.Hits}} / {{.Snapshot.Cache.Misses}}</dd><dt>Load / hata</dt><dd>{{.Snapshot.Cache.Loads}} / {{.Snapshot.Cache.LoadErrors}}</dd><dt>Eviction / expiry</dt><dd>{{.Snapshot.Cache.Evictions}} / {{.Snapshot.Cache.Expirations}}</dd></dl></article>{{end}}{{if .Snapshot.HasJobs}}<article class="studio-card"><div class="resource-card-title"><h3>Arka plan işleri</h3><span class="resource-badge">bounded queue</span></div><dl class="resource-list"><dt>Bekleyen / lease</dt><dd>{{.Snapshot.Jobs.Depth}} / {{.Snapshot.Jobs.Leased}}</dd><dt>Dead-letter</dt><dd>{{.Snapshot.Jobs.DeadLettered}}</dd><dt>Worker paralelliği</dt><dd>{{.Snapshot.WorkerConcurrency}}</dd><dt>İşlenen / başarısız</dt><dd>{{.Snapshot.WorkerProcessed}} / {{.Snapshot.WorkerFailures}}</dd></dl></article>{{end}}</div>
<p class="resource-note">Başlangıç: {{.StartedAt}} · Readiness hataları: {{.Snapshot.ReadinessFails}} · Değerler süreç yeniden başladığında sıfırlanan process-local telemetridir.</p>
</section>`))

type adminResourceUsageView struct {
	AdminResourceUsageData
	HeapAlloc string
	HeapInUse string
	HeapSys   string
	Uptime    string
	StartedAt string
}

func RenderDeveloperTabs(activeTab string) string {
	tabs := []struct {
		ID    string
		Label string
		Href  string
		Icon  string
	}{
		{ID: "console", Label: "API & Entegrasyon", Href: "/admin/developer", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="m8 9-3 3 3 3"></path><path d="m16 9 3 3-3 3"></path><path d="m14 5-4 14"></path></svg>`},
		{ID: "resources", Label: "Sistem Kaynakları", Href: "/admin/developer/resources", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M12 20V10"></path><path d="M18 20V4"></path><path d="M6 20v-4"></path></svg>`},
		{ID: "link-check", Label: "Bağlantı Denetimi", Href: "/admin/site/link-check", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>`},
		{ID: "broadcast", Label: "Staff Notice", Href: "/admin/site/broadcast", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="m3 11 18-5v12L3 14v-3z"></path><path d="M11.6 16.8a3 3 0 1 1-5.8-1.6"></path></svg>`},
		{ID: "custom-code", Label: "Custom Code", Href: "/admin/site/custom-code", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>`},
		{ID: "control", Label: "Yönetim Araçları", Href: "/admin/control", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M4 21v-7"></path><path d="M4 10V3"></path><path d="M12 21v-9"></path><path d="M12 8V3"></path><path d="M20 21v-5"></path><path d="M20 12V3"></path><path d="M1 14h6"></path><path d="M9 8h6"></path><path d="M17 16h6"></path></svg>`},
		{ID: "security", Label: "Güvenlik & Yetkiler", Href: "/admin/security", Icon: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"></path><path d="m9 12 2 2 4-4"></path></svg>`},
	}
	var sb strings.Builder
	sb.WriteString(`<nav class="studio-tabs-nav" aria-label="Geliştirici sekmeleri">`)
	for _, t := range tabs {
		activeClass := ""
		if t.ID == activeTab {
			activeClass = " is-active"
		}
		sb.WriteString(fmt.Sprintf(`<a href="%s" class="studio-tab-item%s">%s<span>%s</span></a>`, t.Href, activeClass, t.Icon, t.Label))
	}
	sb.WriteString(`</nav>`)
	return sb.String()
}

func renderDeveloperTabs(activeTab string) string {
	return RenderDeveloperTabs(activeTab)
}

func RenderAdminResourceUsage(w io.Writer, data AdminResourceUsageData) error {
	s := data.Snapshot
	view := adminResourceUsageView{
		AdminResourceUsageData: data,
		HeapAlloc:              formatBytes(s.HeapAllocBytes),
		HeapInUse:              formatBytes(s.HeapInUseBytes),
		HeapSys:                formatBytes(s.HeapSysBytes),
		Uptime:                 formatDuration(s.Uptime),
		StartedAt:              s.StartedAt.Local().Format("02.01.2006 15:04:05"),
	}
	if err := AdminStartLocale(w, "Kaynak tüketimi", "GELİŞTİRİCİ", "resources", data.Locale); err != nil {
		return err
	}
	if _, err := io.WriteString(w, renderDeveloperTabs("resources")); err != nil {
		return err
	}
	if err := adminResourceUsageTemplate.Execute(w, view); err != nil {
		return fmt.Errorf("render resource usage: %w", err)
	}
	return AdminEnd(w)
}

func formatBytes(value uint64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB"}
	v := float64(value)
	for _, unit := range units {
		v /= 1024
		if v < 1024 || unit == "GB" {
			return fmt.Sprintf("%.1f %s", v, unit)
		}
	}
	return fmt.Sprintf("%d B", value)
}

func formatDuration(value time.Duration) string {
	if value < time.Minute {
		return fmt.Sprintf("%ds", int(value.Seconds()))
	}
	return fmt.Sprintf("%dd %02dh %02dm", int(value/(24*time.Hour)), int(value/time.Hour)%24, int(value/time.Minute)%60)
}

func adminAreaEnglishLabels(area string) (string, string) {
	switch area {
	case "account":
		return "Administrator Account Settings", "ACCOUNT"
	case "developer":
		return "API & Integration Console", "DEVELOPER CENTER"
	case "security":
		return "Security & Access Control", "SECURITY CENTER"
	case "permissions":
		return "Role & Permission Matrix (RBAC)", "PERMISSION MANAGEMENT"
	case "users":
		return "Admin Users", "USER MANAGEMENT"
	case "media":
		return "Media & Asset Pipeline", "MEDIA LIBRARY"
	case "site":
		return "Site & Publishing Surface", "SITE CONFIGURATION"
	default:
		return "Administration Area", "STUDIO"
	}
}

func selectedLocale(current, option string) string {
	if current == option {
		return ` selected`
	}
	return ""
}

func RenderAdminArea(w io.Writer, data AdminAreaData) error {
	var title, areaLabel string
	switch data.Area {
	case "developer":
		title, areaLabel = "API & Entegrasyon Konsolu", "GELİŞTİRİCİ MERKEZİ"
	case "operations":
		title, areaLabel = "Sistem Sağlığı & Operasyon", "OPERASYON MERKEZİ"
	case "security":
		title, areaLabel = "Güvenlik & Erişim Kontrolü", "GÜVENLİK MERKEZİ"
	case "permissions":
		title, areaLabel = "Rol & Yetki Matrisi (RBAC)", "YETKİ YÖNETİMİ"
	case "users":
		title, areaLabel = "Admin Kullanıcıları", "KULLANICI YÖNETİMİ"
	case "account":
		title, areaLabel = "Yönetici Hesap Ayarları", "HESAP BİLGİLERİ"
	case "media":
		title, areaLabel = "Medya & Asset Pipeline", "MEDYA KÜTÜPHANESİ"
	case "site":
		title, areaLabel = "Site & Yayın Yüzeyi", "SİTE YAPILANDIRMASI"
	default:
		title, areaLabel = "Yönetim Alanı", "STUDIO"
	}

	activeItem := data.Area
	if activeItem == "security" || activeItem == "control" || activeItem == "operations" {
		activeItem = "developer"
	}
	if data.Locale == "en" {
		title, areaLabel = adminAreaEnglishLabels(data.Area)
	}
	if err := AdminStartLocale(w, title, areaLabel, activeItem, data.Locale); err != nil {
		return err
	}

	nonce := template.HTMLEscapeString(data.Nonce)
	csrf := template.HTMLEscapeString(data.CSRF)
	email := template.HTMLEscapeString(data.Email)

	switch data.Area {
	case "site":
		body := templ.ComponentFunc(func(ctx context.Context, output io.Writer) error {
			links := []uicomponents.LinkProps{{Href: "/admin/site/seo", Label: "SEO ayarları", Class: "button button-primary"}, {Href: "/admin/site/redirect-rules", Label: "Yönlendirmeler", Class: "button button-quiet"}, {Href: "/admin/site/navigation/editor", Label: "Menüler", Class: "button button-quiet"}, {Href: "/admin/translations", Label: "Çeviriler", Class: "button button-quiet"}}
			if data.FontToolsEnabled {
				links = append(links, uicomponents.LinkProps{Href: "/admin/site/fonts", Label: "Yazı tipi", Class: "button button-quiet"})
			}
			for _, link := range links {
				if err := uicomponents.Link(link).Render(ctx, output); err != nil {
					return err
				}
			}
			return nil
		})
		if err := uicomponents.Card(uicomponents.CardProps{Title: "Site keşfi ve bağlantılar", Body: body}).Render(context.Background(), w); err != nil {
			return err
		}
	case "developer":
		if _, err := io.WriteString(w, renderDeveloperTabs("console")); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(w, `<div class="dev-console" data-testid="developer-area">
  <div class="dev-banner">
    <div>
      <span class="eyebrow">HEADLESS &amp; API MERKEZİ</span>
      <h2>Geliştirici &amp; Entegrasyon Konsolu</h2>
      <p>HydroGo Core, headless mimariyle tüm içerik, e-ticaret ve medya yeteneklerini tip-güvenli JSON ve GraphQL API'leri üzerinden bağımsız frontend ve uygulamalara sunar.</p>
    </div>
    <a href="/admin/developer/resources" class="button"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 20V10"></path><path d="M18 20V4"></path><path d="M6 20v-4"></path></svg> <span>Canlı Kaynak Tüketimi →</span></a>
  </div>

  <div class="dev-endpoints-grid">
    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-get">GET</span>
          <span class="dev-path">/api/v1/commerce/products</span>
        </div>
        <div class="dev-ep-actions">
          <button class="button button-quiet" type="button" data-test-endpoint="/api/v1/commerce/products">▶ Canlı Test Et</button>
          <a class="button button-quiet" href="/api/v1/commerce/products" target="_blank">JSON Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.78rem;">E-Ticaret ürün kataloğu, aktif fiyatlar, varyant stokları ve görsellerin genel JSON projeksiyonu.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-get">GET</span>
          <span class="dev-path">/api/news?locale=tr</span>
        </div>
        <div class="dev-ep-actions">
          <button class="button button-quiet" type="button" data-test-endpoint="/api/news?locale=tr">▶ Canlı Test Et</button>
          <a class="button button-quiet" href="/api/news?locale=tr" target="_blank">JSON Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.78rem;">Yayınlanmış editoryal haberler, öne çıkan manşetler, etiketler ve medya referansları listesi.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-post">POST</span>
          <span class="dev-path">/api/graphql</span>
        </div>
        <div class="dev-ep-actions">
          <button class="button button-quiet" type="button" data-test-graphql>▶ Canlı Query Test Et</button>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.78rem;">Next.js, React veya mobil uygulamalar için tek istekte birleşik içerik ve katalog sorgulama uç noktası.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-get">GET</span>
          <span class="dev-path">/readyz &amp; /healthz</span>
        </div>
        <div class="dev-ep-actions">
          <button class="button button-quiet" type="button" data-test-endpoint="/readyz">▶ Readiness Test Et</button>
          <a class="button button-quiet" href="/readyz" target="_blank">Status Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.78rem;">Kubernetes, Docker ve yük dengeleyiciler (Reverse Proxy) için canlılık ve veritabanı hazır bulunuşluk sondaları.</div>
      <pre class="dev-preview-box"></pre>
    </div>
  </div>

  <div class="dev-two-col">
    <div class="dev-code-card">
      <h3>🚀 Hızlı API Tüketimi (cURL &amp; Fetch)</h3>
      <p>Uygulamanızdan içerik ve ürünleri çekmek için standart HTTP istemcinizi kullanabilirsiniz:</p>
      <pre class="dev-code-snippet">curl -X GET "http://127.0.0.1:8090/api/v1/commerce/products" \
  -H "Accept: application/json"</pre>
    </div>
    <div class="dev-code-card">
      <h3>📦 Tip Güvenli Entegrasyon</h3>
      <p>HydroGo SDK ve GraphQL şeması sayesinde tüm tipler derleme anında doğrulanır. Drift ve şema uyuşmazlığı engellenir.</p>
      <pre class="dev-code-snippet">// Go SDK Örneği
client := hydrogo.NewClient("http://127.0.0.1:8090")
products, err := client.Commerce.ListProducts(ctx)</pre>
    </div>
  </div>
</div>
<script nonce="%s">
document.addEventListener('click', event => {
  const endpointButton = event.target.closest('[data-test-endpoint]');
  if (endpointButton) testEndpoint(endpointButton, endpointButton.dataset.testEndpoint);
  const graphqlButton = event.target.closest('[data-test-graphql]');
  if (graphqlButton) testGraphQL(graphqlButton);
});
function testEndpoint(btn, url){
  const box = btn.closest('.dev-endpoint-card').querySelector('.dev-preview-box');
  btn.disabled = true;
  btn.textContent = 'Yükleniyor…';
  fetch(url, { headers: { 'Accept': 'application/json' } })
    .then(r => r.text())
    .then(txt => {
      try { txt = JSON.stringify(JSON.parse(txt), null, 2); } catch(e){}
      box.textContent = txt;
      box.style.display = 'block';
      btn.disabled = false;
      btn.textContent = '✓ Yanıt Alındı';
    })
    .catch(err => {
      box.textContent = 'Hata: İstek tamamlanamadı.';
      box.style.display = 'block';
      btn.disabled = false;
      btn.textContent = 'Tekrar Dene';
    });
}

function testGraphQL(btn){
  const box = btn.closest('.dev-endpoint-card').querySelector('.dev-preview-box');
  btn.disabled = true;
  btn.textContent = 'Query çalışıyor…';
  fetch('/api/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
     body: JSON.stringify({ query: '{ products { id title priceCents currency } }' })
  })
    .then(r => r.text())
    .then(txt => {
      try { txt = JSON.stringify(JSON.parse(txt), null, 2); } catch(e){}
      box.textContent = txt;
      box.style.display = 'block';
      btn.disabled = false;
      btn.textContent = '✓ Query Tamam';
    })
    .catch(err => {
      box.textContent = 'GraphQL isteği tamamlanamadı.';
      box.style.display = 'block';
      btn.disabled = false;
      btn.textContent = 'Tekrar Dene';
    });
}
</script>`, nonce)

	case "operations":
		_, _ = io.WriteString(w, `<div class="ops-grid" data-testid="operations-area">
  <div class="ops-kpis">
    <div class="ops-kpi"><small>SERVİS DURUMU</small><strong>Canlı doğrulama</strong><span>Durumu /readyz üzerinden kontrol edin</span></div>
    <div class="ops-kpi"><small>VERİTABANI</small><strong>Yapılandırmaya bağlı</strong><span>Etkin adaptörü dağıtım ayarlarından doğrulayın</span></div>
    <div class="ops-kpi"><small>ARKA PLAN KUYRUKLARI</small><strong>Gözlem gerekli</strong><span>Bekleyen işleri operasyon panelinden inceleyin</span></div>
    <div class="ops-kpi"><small>DEAD-LETTER JOBS</small><strong>Gözlem gerekli</strong><span>Hata durumunu canlı veriden doğrulayın</span></div>
  </div>

  <div class="ops-two-col">
    <div class="ops-card">
      <h3>🛠️ Yedekleme &amp; Felaket Kurtarma (Backup &amp; Restore)</h3>
      <p>Yedekleme ve geri yükleme prosedürlerini bu ortamın çalıştırma kılavuzuna göre yürütün; işlemi üretim benzeri bir ortamda doğrulamadan başarılı kabul etmeyin.</p>
      <div style="font-size:.8rem;color:var(--ink);font-weight:600;margin-bottom:4px;">CLI Yedekleme Komutu:</div>
      <div class="ops-cli-box">hydro backup create --config hydro.yaml</div>
      <div style="font-size:.8rem;color:var(--ink);font-weight:600;margin:10px 0 4px;">Sağlık Kontrolü (Doctor):</div>
      <div class="ops-cli-box">hydro doctor --config hydro.yaml</div>
      <div class="ops-action-row">
        <a href="/admin/control#operations" class="button">Operasyon Kontrol Paneli ↗</a>
        <a href="/admin/developer/resources" class="button button-quiet">Canlı Kaynak Metrikleri</a>
        <a href="/admin/commerce/refunds/pending" class="button button-quiet">Bekleyen İadeler</a>
      </div>
    </div>

    <div class="ops-card">
      <h3>🛡️ Blue/Green Dağıtım &amp; Güvenlik Fensi</h3>
      <p>Blue/green geçişi ve geri alma, sürüm uyumluluğu ile işletim prosedürlerine bağlıdır. Canlı ortamda uygulanmadan önce geçiş planını test edin.</p>
      <ul style="margin:0 0 14px;padding-left:18px;color:var(--muted);font-size:.8rem;line-height:1.55;">
        <li>Şema ve uygulama sürümlerinin uyumluluğunu yayın öncesi doğrulayın.</li>
        <li>Medya, oturum ve arka plan işlerinin sürekliliği için geri dönüş planı hazırlayın.</li>
        <li>Geri yüklemeyi önce izole bir ortamda deneyin.</li>
      </ul>
      <div style="font-size:.76rem;padding:8px 12px;background:var(--bg-subtle);border-radius:var(--radius-sm);border:1px solid var(--line);color:var(--muted);">
        Sistem kimliği ve sürümü için çalışma zamanı durum uç noktasını kullanın.
      </div>
    </div>
  </div>
</div>`)

	case "security":
		if _, err := io.WriteString(w, renderDeveloperTabs("security")); err != nil {
			return err
		}
		if err := designsystem.TokenLifetimeLink().Render(context.Background(), w); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(w, `<div class="sec-grid" data-testid="security-area">
  <div class="sec-kpis">
    <div class="sec-kpi"><small>KİMLİK KORUMASI</small><strong>Güçlü Şifreleme</strong><span>Endüstri standardı hash &amp; salt koruması</span></div>
    <div class="sec-kpi"><small>OTURUM GÜVENLİĞİ</small><strong>Güvenli Oturum</strong><span>Yetkisiz erişime karşı korumalı çerezler</span></div>
    <div class="sec-kpi"><small>İÇERİK GÜVENLİĞİ</small><strong>Aktif Kalkan</strong><span>Zararlı script ve kod enjeksiyonu engelleme</span></div>
  </div>

  <div class="sec-two-col">
    <div class="sec-card">
      <h3>✉️ Yeni Yönetici / Editör Davet Et</h3>
      <p>Yeni bir ekip üyesine Studio erişimi vermek için güvenli davet bağlantısı oluşturun:</p>
      <form class="studio-form sec-form-row" method="post" action="/admin/auth/invites" data-admin-invite>
        <input type="hidden" name="_csrf" value="%s">
        <input type="hidden" name="_action" value="invite">
        <label>E-posta Adresi <input type="email" name="email" required placeholder="yeni-editor@sirket.com"></label>
        <label>Erişim Rolü
          <select name="role">
            <option value="editor">Editor (İçerik &amp; Taslak Düzenleme)</option>
            <option value="admin">Admin (Tam Yönetim Yetkisi)</option>
          </select>
        </label>
        <button class="button" type="submit" style="margin-top:6px;width:auto;display:inline-flex;align-self:start;">Davet Bağlantısı Oluştur ✉️</button>
        <p role="status" aria-live="polite" data-invite-status></p>
        <div data-invite-result hidden>
          <label for="admin-invite-link">Davet bağlantısı</label>
          <input id="admin-invite-link" type="url" readonly data-invite-link>
          <button class="button button-quiet" type="button" data-invite-copy>Bağlantıyı kopyala</button>
        </div>
      </form>
    </div>

    <div class="sec-card">
      <h3>🔐 İki Faktörlü Doğrulama (MFA / TOTP)</h3>
      <p>Yönetici hesabınızı Google Authenticator veya 1Password gibi doğrulama uygulamaları ile ek koruma altına alın.</p>
      <div style="background:var(--bg-subtle);border:1px solid var(--line);border-radius:var(--radius-md);padding:12px 14px;margin-bottom:12px;">
        <div style="font-size:.84rem;color:var(--ink);font-weight:600;">Oturum Açan: %s</div>
        <div style="font-size:.74rem;color:var(--muted);margin-top:2px;">Rol: Sistem Yöneticisi (Admin)</div>
      </div>
      <a href="/admin/control#security" class="button button-quiet" style="display:inline-flex;align-items:center;gap:6px;padding:0 12px;text-decoration:none;">MFA Kurulumunu Yönet 🔑</a>
    </div>
  </div>

  <div class="perm-table-card">
    <div style="padding:12px 16px;border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between;">
      <strong style="font-size:.88rem;color:var(--ink);">🛡️ Rol &amp; Yetki Matrisi (RBAC)</strong>
      <span class="status-badge">En Az Yetki İlkesi</span>
    </div>
    <div style="overflow-x:auto;">
      <table class="perm-table">
        <thead>
          <tr>
            <th>Çalışma Alanı / Modül</th>
            <th>👑 Admin (Yönetici)</th>
            <th>✍️ Editor (Editör)</th>
            <th>👁️ Viewer (İzleyici / API)</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>📰 Haberler &amp; İçerik</strong></td>
            <td><span class="badge-perm badge-full">✓ Tam Yetki &amp; Yayın</span></td>
            <td><span class="badge-perm badge-edit">✏️ Taslak &amp; Düzenle</span></td>
            <td><span class="badge-perm badge-view">👁️ Yalnızca Oku</span></td>
          </tr>
          <tr>
            <td><strong>📄 Sayfalar &amp; Bloklar</strong></td>
            <td><span class="badge-perm badge-full">✓ Tam Yetki &amp; Yayın</span></td>
            <td><span class="badge-perm badge-edit">✏️ Taslak &amp; Düzenle</span></td>
            <td><span class="badge-perm badge-view">👁️ Yalnızca Oku</span></td>
          </tr>
          <tr>
            <td><strong>🛍️ Ürün Kataloğu &amp; Varyantlar</strong></td>
            <td><span class="badge-perm badge-full">✓ Ürün Ekle / Sil / Fiyat</span></td>
            <td><span class="badge-perm badge-view">👁️ Katalog İnceleme</span></td>
            <td><span class="badge-perm badge-view">👁️ Public JSON API</span></td>
          </tr>
          <tr>
            <td><strong>📦 Siparişler &amp; Müşteri Verisi</strong></td>
            <td><span class="badge-perm badge-full">✓ Sipariş Yönetimi</span></td>
            <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
            <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
          </tr>
          <tr>
            <td><strong>👥 Üye &amp; Kullanıcı Hesapları</strong></td>
            <td><span class="badge-perm badge-full">✓ Üye Yönetimi &amp; Oturum</span></td>
            <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
            <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
          </tr>
          <tr>
            <td><strong>🖼️ Medya Kütüphanesi</strong></td>
            <td><span class="badge-perm badge-full">✓ Yükle &amp; Sil</span></td>
            <td><span class="badge-perm badge-edit">✏️ Görsel Yükle</span></td>
            <td><span class="badge-perm badge-view">👁️ CDN / WebP</span></td>
          </tr>
          <tr>
            <td><strong>⚙️ Sistem, Yedekleme &amp; Operasyon</strong></td>
            <td><span class="badge-perm badge-full">✓ CLI &amp; Tam Kontrol</span></td>
            <td><span class="badge-perm badge-none">✗ Yetkisiz</span></td>
            <td><span class="badge-perm badge-none">✗ Yetkisiz</span></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</div>`, csrf, email)
		_, _ = io.WriteString(w, `<section class="sec-card" aria-labelledby="ip-ban-title"><h3 id="ip-ban-title">IP engelleme</h3><p>Bir IP adresini veya CIDR aralığını geçici ya da süresiz engelleyin.</p><form method="post" action="/admin/security/bans" class="studio-form"><input type="hidden" name="_csrf" value="`+csrf+`">`)
		if err := uicomponents.Input(uicomponents.InputProps{ID: "security-ban-address", Name: "address", Label: "IP adresi veya CIDR", Type: "text", Placeholder: "192.0.2.10 veya 192.0.2.0/24", Required: true, MaxLength: 64, HasMaxLength: true}).Render(context.Background(), w); err != nil {
			return err
		}
		if err := uicomponents.Input(uicomponents.InputProps{ID: "security-ban-duration", Name: "duration", Label: "Süre (örn. 24h; boş bırakılırsa süresiz)", Type: "text", Placeholder: "24h", MaxLength: 32, HasMaxLength: true}).Render(context.Background(), w); err != nil {
			return err
		}
		if err := uicomponents.Button(uicomponents.ButtonProps{Label: "IP engelle", Type: "submit", Variant: uicomponents.Primary}).Render(context.Background(), w); err != nil {
			return err
		}
		_, _ = io.WriteString(w, `</form><form method="post" action="/admin/security/bans/remove" class="studio-form"><input type="hidden" name="_csrf" value="`+csrf+`">`)
		if err := uicomponents.Input(uicomponents.InputProps{ID: "security-unban-address", Name: "address", Label: "Engeli kaldırılacak adres veya CIDR", Type: "text", Required: true, MaxLength: 64, HasMaxLength: true}).Render(context.Background(), w); err != nil {
			return err
		}
		if err := uicomponents.Button(uicomponents.ButtonProps{Label: "Engeli kaldır", Type: "submit", Variant: uicomponents.Quiet}).Render(context.Background(), w); err != nil {
			return err
		}
		_, _ = io.WriteString(w, `</form>`)
		rows := make([][]string, 0, len(data.IPBans))
		for _, item := range data.IPBans {
			rows = append(rows, []string{item.Address, item.Expires})
		}
		if err := uicomponents.Table(uicomponents.TableProps{Caption: "Engellenen IP adresleri", Columns: []string{"Adres", "Bitiş"}, Rows: rows, EmptyMessage: "Etkin engelleme yok."}).Render(context.Background(), w); err != nil {
			return err
		}
		_, _ = io.WriteString(w, `</section>`)

	case "permissions":
		_, _ = io.WriteString(w, `<div class="perm-grid" data-testid="permissions-area">
  <div class="perm-table-card">
    <table class="perm-table">
      <thead>
        <tr>
          <th>Çalışma Alanı / Modül</th>
          <th>👑 Admin (Yönetici)</th>
          <th>✍️ Editor (Editör)</th>
          <th>👁️ Viewer (İzleyici / API)</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td><strong>📰 Haberler &amp; İçerik</strong></td>
          <td><span class="badge-perm badge-full">✓ Tam Yetki &amp; Yayın</span></td>
          <td><span class="badge-perm badge-edit">✏️ Taslak &amp; Düzenle</span></td>
          <td><span class="badge-perm badge-view">👁️ Yalnızca Oku</span></td>
        </tr>
        <tr>
          <td><strong>📄 Sayfalar &amp; Bloklar</strong></td>
          <td><span class="badge-perm badge-full">✓ Tam Yetki &amp; Yayın</span></td>
          <td><span class="badge-perm badge-edit">✏️ Taslak &amp; Düzenle</span></td>
          <td><span class="badge-perm badge-view">👁️ Yalnızca Oku</span></td>
        </tr>
        <tr>
          <td><strong>🛍️ Ürün Kataloğu &amp; Varyantlar</strong></td>
          <td><span class="badge-perm badge-full">✓ Ürün Ekle / Sil / Fiyat</span></td>
          <td><span class="badge-perm badge-view">👁️ Katalog İnceleme</span></td>
          <td><span class="badge-perm badge-view">👁️ Public JSON API</span></td>
        </tr>
        <tr>
          <td><strong>📦 Siparişler &amp; Müşteri Verisi</strong></td>
          <td><span class="badge-perm badge-full">✓ Sipariş Yönetimi</span></td>
          <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
          <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
        </tr>
        <tr>
          <td><strong>👥 Üye &amp; Kullanıcı Hesapları</strong></td>
          <td><span class="badge-perm badge-full">✓ Üye Yönetimi &amp; Oturum</span></td>
          <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
          <td><span class="badge-perm badge-none">✗ Erişim Yok</span></td>
        </tr>
        <tr>
          <td><strong>🖼️ Medya Kütüphanesi</strong></td>
          <td><span class="badge-perm badge-full">✓ Yükle &amp; Sil</span></td>
          <td><span class="badge-perm badge-edit">✏️ Görsel Yükle</span></td>
          <td><span class="badge-perm badge-view">👁️ CDN / WebP</span></td>
        </tr>
        <tr>
          <td><strong>⚙️ Sistem, Yedekleme &amp; Operasyon</strong></td>
          <td><span class="badge-perm badge-full">✓ CLI &amp; Tam Kontrol</span></td>
          <td><span class="badge-perm badge-none">✗ Yetkisiz</span></td>
          <td><span class="badge-perm badge-none">✗ Yetkisiz</span></td>
        </tr>
      </tbody>
    </table>
  </div>

  <div class="perm-rules-card">
    <h3>🛡️ En Az Yetki Prensibi (Principle of Least Privilege)</h3>
    <p>Yetkilendirme kuralları yalnızca arayüzdeki menüleri gizlemekle kalmaz; tüm servis ve veri işlemlerinde sistem yetkilendirme çekirdeği tarafından merkezi olarak zorunlu doğrulanır.</p>
    <div style="display:flex;gap:8px;flex-wrap:wrap;">
      <a href="/admin/security" class="button">Güvenlik &amp; Davet Merkezi →</a>
      <a href="/admin/users" class="button button-quiet">Kullanıcıları Yönet</a>
    </div>
  </div>
		</div>`)
		if len(data.Permissions) != 0 {
			_, _ = io.WriteString(w, `<section class="studio-card" data-testid="edition-permission-catalog"><h2>Edition resource permissions</h2><p class="muted">Only permissions declared by active Edition resources can be assigned. Resource policy still decides how each permission is enforced.</p><ul>`)
			for _, permission := range data.Permissions {
				_, _ = fmt.Fprintf(w, `<li><code>%s</code></li>`, template.HTMLEscapeString(permission))
			}
			_, _ = io.WriteString(w, `</ul></section>`)
		}

	case "users":
		usersTitle, usersIntro, emptyText := "Admin Studio Kullanıcıları", "Bu liste yalnızca yönetim paneline giriş yapan hesapları içerir. Frontend üyeleri Üyeler ekranında ayrı tutulur.", "Henüz admin kullanıcısı bulunamadı."
		roleLabel, activeLabel, saveLabel := "Rol", "Durum", "Kaydet"
		if data.Locale == "en" {
			usersTitle, usersIntro, emptyText = "Admin Studio users", "This list contains Admin Studio accounts only. Frontend members remain separate on the Members screen.", "No admin users found."
			roleLabel, activeLabel, saveLabel = "Role", "Status", "Save"
		}
		_, _ = fmt.Fprintf(w, `<div class="studio-card" data-testid="users-area"><div class="dashboard-card-head"><div><h2>%s</h2><p class="muted">%s</p></div></div><div class="studio-table-wrap"><table class="studio-table"><thead><tr><th>Email</th><th>%s</th><th>%s</th><th>Actions</th></tr></thead><tbody>`, usersTitle, usersIntro, roleLabel, activeLabel)
		if len(data.Users) == 0 {
			_, _ = fmt.Fprintf(w, `<tr><td colspan="4">%s</td></tr>`, emptyText)
		}
		for _, u := range data.Users {
			roleOptions := fmt.Sprintf(`<select name="role" aria-label="%s · %s"><option value="admin"%s>Admin</option><option value="editor"%s>Editor</option><option value="viewer"%s>Viewer</option></select>`, template.HTMLEscapeString(u.Email), roleLabel, selectedLocale(string(u.Role), "admin"), selectedLocale(string(u.Role), "editor"), selectedLocale(string(u.Role), "viewer"))
			permissionControls := ""
			permissions := data.Permissions
			if len(permissions) == 0 {
				permissions = auth.KnownPermissions
			}
			for _, permission := range permissions {
				value := "inherit"
				if allowed, ok := u.PermissionOverrides[permission]; ok {
					if allowed {
						value = "allow"
					} else {
						value = "deny"
					}
				}
				permissionControls += fmt.Sprintf(`<label style="display:flex;align-items:center;gap:8px;font-size:.78rem;"><span style="flex:1;">%s</span><select name="perm_%s"><option value="inherit"%s>Inherit</option><option value="allow"%s>Allow</option><option value="deny"%s>Deny</option></select></label>`, template.HTMLEscapeString(permission), template.HTMLEscapeString(permission), selectedLocale(value, "inherit"), selectedLocale(value, "allow"), selectedLocale(value, "deny"))
			}
			status := "Inactive"
			if u.Active {
				status = "Active"
			}
			if data.Locale == "tr" && u.Active {
				status = "Aktif"
			}
			if data.Locale == "tr" && !u.Active {
				status = "Pasif"
			}
			verificationLabel, verifiedLabel := "Doğrulama bekliyor", "Doğrulanmış"
			if data.Locale == "en" {
				verificationLabel, verifiedLabel = "Unverified", "Verified"
			}
			verification := `<span class="status-pill status-pill--draft">` + verificationLabel + `</span>`
			resend := ""
			if u.EmailVerifiedAt != nil {
				verification = `<span class="status-pill status-pill--published">` + verifiedLabel + `</span>`
			} else if u.ID != "" && data.EmailVerificationEnabled {
				label := "Doğrulama e-postasını yeniden gönder"
				if data.Locale == "en" {
					label = "Resend verification email"
				}
				resend = fmt.Sprintf(`<form method="post" action="/auth/users/%s/email/verification/resend"><input type="hidden" name="_csrf" value="%s"><button class="button button-quiet" type="submit">%s</button></form>`, template.HTMLEscapeString(u.ID), csrf, label)
			}
			resetLink := ""
			if data.PasswordResetLinkEnabled && u.Active && u.ID != "" {
				buttonLabel := "Parola sıfırlama bağlantısını kopyala"
				if data.Locale == "en" {
					buttonLabel = "Copy password reset link"
				}
				resetLink = fmt.Sprintf(`<button class="button button-quiet" type="button" data-password-reset-link="/auth/users/%s/password-reset-link">%s</button><span role="status" aria-live="polite" data-password-reset-status></span>`, template.HTMLEscapeString(u.ID), buttonLabel)
			}
			_, _ = fmt.Fprintf(w, `<tr><td><strong>%s</strong><div>%s</div></td><td><form method="post" action="/auth/users/%s/role" class="inline-form"><input type="hidden" name="_csrf" value="%s">%s <button class="button button-quiet" type="submit">%s</button></form></td><td>%s</td><td>%s%s%s<details style="margin-top:8px;"><summary>Custom permissions</summary><form method="post" action="/auth/users/%s/permissions" class="studio-form" style="margin-top:8px;"><input type="hidden" name="_csrf" value="%s">%s<button class="button button-quiet" type="submit">%s</button></form></details></td></tr>`, template.HTMLEscapeString(u.Email), verification, template.HTMLEscapeString(u.ID), csrf, roleOptions, saveLabel, status, resend, resetLink, func() string {
				if u.ID == "" {
					return ""
				}
				return fmt.Sprintf(`<form method="post" action="/auth/users/%s/disable"><input type="hidden" name="_csrf" value="%s"><button class="button button-quiet" type="submit">Disable</button></form>`, template.HTMLEscapeString(u.ID), csrf)
			}(), template.HTMLEscapeString(u.ID), csrf, permissionControls, saveLabel)
		}
		_, _ = io.WriteString(w, `</tbody></table></div></div>`)

	case "account":
		w = passwordStrengthWriter{Writer: w}
		profileTitle := "👤 Yönetici Profil Bilgileri"
		profileIntro := "Admin Studio paneline erişim sağlayan mevcut yetkili hesap:"
		securityTitle := "🔒 Hesap Güvenliği &amp; Parola Yenileme"
		emailLabel, roleLabel, sessionLabel, scopeLabel := "E-Posta:", "Rol:", "Oturum Tipi:", "Erişim Alanı:"
		roleValue, sessionValue, scopeValue := "👑 Sistem Yöneticisi (Admin)", "Güvenli Oturum", "Tüm Modüller &amp; Operasyon"
		securityIntro := "Yönetici parolanız yüksek güvenlik standartları ile korunmaktadır. Güvenlik politikası gereği güçlü parolalar (en az 8 karakter) zorunludur."
		passwordLabel := "Parola Değiştirme:"
		passwordHelp := "Parolanızı yenilemek için Güvenlik Merkezinden şifre sıfırlama akışını başlatabilir veya CLI üzerinden <code>hydro user reset-password</code> komutunu çalıştırabilirsiniz."
		accountSave, currentPasswordLabel, newPasswordLabel, confirmPasswordLabel := "Hesap Bilgilerini Kaydet", "Mevcut Parola", "Yeni Parola", "Yeni Parola (Tekrar)"
		accountHelp := "Email veya parolanızı değiştirmek için mevcut parolanızı doğrulayın. Değişiklikten sonra güvenlik için tüm oturumlar kapatılır."
		appearanceTitle, appearanceIntro, appearanceLabel := "Görünüm", "Studio'nun görsel temasını seçin. Tercihiniz bu tarayıcıda saklanır.", "Tema"
		if data.Locale == "en" {
			appearanceTitle, appearanceIntro, appearanceLabel = "Appearance", "Choose Studio's visual theme. Your choice is saved in this browser.", "Theme"
		}
		if data.Locale == "en" {
			profileTitle = "👤 Administrator profile"
			profileIntro = "The administrator account currently authorized to access Admin Studio:"
			securityTitle = "🔒 Account security &amp; password reset"
			emailLabel, roleLabel, sessionLabel, scopeLabel = "Email:", "Role:", "Session type:", "Access scope:"
			roleValue, sessionValue, scopeValue = "👑 System administrator (Admin)", "Secure session", "All modules &amp; operations"
			securityIntro = "Your administrator password is protected by strong security standards. Passwords must contain at least 8 characters."
			passwordLabel = "Change password:"
			passwordHelp = "Start a password reset flow from the Security Center or run <code>hydro user reset-password</code> from the CLI."
			accountSave, currentPasswordLabel, newPasswordLabel, confirmPasswordLabel = "Save account changes", "Current password", "New password", "New password (confirm)"
			accountHelp = "Verify your current password to change your email or password. All sessions will be signed out after the change."
		}
		languageSaved := ""
		if data.LocaleSaved {
			languageSaved = `<p class="studio-kit-summary" role="status">` + template.HTMLEscapeString(i18n.TForTenant(data.TenantID, data.Locale, "admin.language.saved")) + `</p>`
		}
		_, _ = fmt.Fprintf(w, `<div class="acc-grid">
  <div class="acc-card">
    <h3>%s</h3>
    <p>%s</p>
    <div style="margin:12px 0;">
      <div class="acc-info-row"><span class="acc-info-label">%s</span><span class="acc-info-val">%s</span></div>
      <div class="acc-info-row"><span class="acc-info-label">%s</span><span class="acc-info-val" style="color:var(--brand-light);">%s</span></div>
      <div class="acc-info-row"><span class="acc-info-label">%s</span><span class="acc-info-val">%s</span></div>
      <div class="acc-info-row"><span class="acc-info-label">%s</span><span class="acc-info-val">%s</span></div>
    </div>
    <div style="margin-top:16px;display:flex;gap:8px;">
      <a href="/admin/security" class="button">Güvenlik &amp; Yetki Ayarları</a>
      <button type="button" data-admin-logout class="button button-quiet" style="cursor:pointer;">Oturumu Kapat</button>
    </div>
  </div>

  <div class="acc-card">
    <h3>%s</h3>
    <p>%s</p>
    <div style="background:var(--bg-subtle);border:1px solid var(--line);border-radius:var(--radius-md);padding:14px;margin-top:12px;">
      <div style="font-weight:700;font-size:.85rem;color:var(--ink);margin-bottom:4px;">%s</div>
      <div style="font-size:.78rem;color:var(--muted);line-height:1.45;">%s</div>
    </div>
  </div>
  <div class="acc-card">
    <h3>✏️ %s</h3>
    <p>%s</p>
    <form method="post" action="/auth/me/credentials" class="studio-form" style="margin-top:14px;">
      <input type="hidden" name="_csrf" value="%s">
      <label class="studio-form-field"><span>%s</span><input type="email" name="email" value="%s" autocomplete="email" required></label>
      <label class="studio-form-field"><span>%s</span><input type="password" name="current_password" autocomplete="current-password" required></label>
      <label class="studio-form-field"><span>%s</span><input type="password" name="new_password" minlength="8" autocomplete="new-password"></label>
      <label class="studio-form-field"><span>%s</span><input type="password" name="confirm_password" minlength="8" autocomplete="new-password"></label>
      <button class="button" type="submit">%s</button>
    </form>
  </div>
  <div class="acc-card">
    <h3>🌐 %s</h3>
    <p>%s</p>
    %s
    <form method="post" action="/admin/account/locale" class="studio-form">
      <input type="hidden" name="_csrf" value="%s">
      <label class="studio-form-field"><span>%s</span><select name="locale"><option value="tr"%s>Türkçe (TR)</option><option value="en"%s>English (EN)</option></select></label>
      <button class="button" type="submit">%s</button>
    </form>
  </div>
  <div class="acc-card">
    <h3>🎨 %s</h3>
    <p>%s</p>
    <label class="studio-form-field"><span>%s</span><select id="ui-family-select" class="ui-family-select" aria-label="%s"><option value="hydrogo">HydroGo</option><option value="shadcn">shadcn</option><option value="nova">Nova</option><option value="aurora">Aurora</option></select></label>
  </div>
		</div>`, profileTitle, profileIntro, emailLabel, email, roleLabel, roleValue, sessionLabel, sessionValue, scopeLabel, scopeValue, securityTitle, securityIntro, passwordLabel, passwordHelp, accountSave, accountHelp, csrf, emailLabel, email, currentPasswordLabel, newPasswordLabel, confirmPasswordLabel, accountSave, i18n.TForTenant(data.TenantID, data.Locale, "admin.language"), i18n.TForTenant(data.TenantID, data.Locale, "admin.language"), languageSaved, csrf, i18n.TForTenant(data.TenantID, data.Locale, "admin.language"), selectedLocale(data.Locale, "tr"), selectedLocale(data.Locale, "en"), i18n.TForTenant(data.TenantID, data.Locale, "admin.language.save"), appearanceTitle, appearanceIntro, appearanceLabel, appearanceLabel)
		soundTitle, soundLabel, soundDescription := "Bildirim sesleri", "Yeni taleplerde sesli uyarı ver", "Bu tercih yalnızca bu tarayıcıda saklanır ve varsayılan olarak kapalıdır."
		if data.Locale == "en" {
			soundTitle, soundLabel, soundDescription = "Notification sounds", "Play a sound for new inquiries", "This preference is stored in this browser and is off by default."
		}
		if data.SoundAlertsEnabled {
			_ = designsystem.SoundAlertPreference(designsystem.SoundAlertPreferenceProps{Title: soundTitle, Label: soundLabel, Description: soundDescription}).Render(context.Background(), w)
		}

	default:
		_, _ = fmt.Fprintf(w, `<div class="studio-card"><p>Yönetim alanı yükleniyor...</p></div>`)
	}

	return AdminEnd(w)
}
