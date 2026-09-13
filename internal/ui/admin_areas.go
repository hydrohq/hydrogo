package ui

import (
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/observability"
)

type AdminAreaData struct {
	Email string
	Nonce string
	CSRF  string
	Area  string
}

type AdminResourceUsageData struct {
	Email    string
	Nonce    string
	Snapshot observability.RuntimeSnapshot
}

var adminResourceUsageTemplate = template.Must(template.New("admin-resource-usage").Parse(`<section class="resource-page" data-testid="resource-page">
<div class="resource-banner"><div><span class="eyebrow">CANLI PROCESS GÖRÜNÜMÜ</span><h2>Sistem kaynakları</h2><p>Bu ekran yalnızca mevcut process'in ölçebildiği anlık değerleri gösterir. Tarihsel veri, host CPU yüzdesi ve disk telemetrisi bu uygulama tarafından uydurulmaz.</p></div><span class="resource-status {{if .Snapshot.ReadinessState}}is-ready{{else}}is-waiting{{end}}"><i></i>{{if .Snapshot.ReadinessState}}Hazır{{else}}Hazırlanıyor{{end}}</span></div>
<div class="resource-kpis"><div class="resource-kpi"><small>Heap kullanımı</small><strong>{{.HeapAlloc}}</strong><span>{{.HeapInUse}} kullanım alanı</span></div><div class="resource-kpi"><small>Goroutine</small><strong>{{.Snapshot.Goroutines}}</strong><span>{{.Snapshot.GOMAXPROCS}} paralel çalışma limiti</span></div><div class="resource-kpi"><small>İstek / gecikme</small><strong>{{.Snapshot.Requests}}</strong><span>Ortalama {{.Snapshot.AvgLatencyMS}} ms</span></div><div class="resource-kpi"><small>Uptime</small><strong>{{.Uptime}}</strong><span>Başlangıç: {{.StartedAt}}</span></div></div>
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
	if err := AdminStart(w, "Kaynak tüketimi", "GELİŞTİRİCİ", data.Nonce, "resources"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, `<style nonce="%s">.resource-page{display:grid;gap:18px}.resource-banner{display:flex;justify-content:space-between;gap:24px;align-items:flex-start;padding:24px;border:1px solid var(--line);border-radius:var(--radius-lg);background:linear-gradient(135deg,var(--surface),var(--bg-subtle))}.resource-banner h2{margin:5px 0 6px;font-size:1.35rem}.resource-banner p{max-width:700px;margin:0;color:var(--muted);font-size:.86rem;line-height:1.55}.resource-status{display:inline-flex;align-items:center;gap:8px;white-space:nowrap;padding:8px 12px;border-radius:999px;background:var(--bg-subtle);color:var(--muted);font-size:.78rem;font-weight:700}.resource-status i{width:8px;height:8px;border-radius:50%%;background:var(--muted)}.resource-status.is-ready{color:var(--green)}.resource-status.is-ready i{background:var(--green)}.resource-kpis{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px}.resource-kpi{padding:18px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}.resource-kpi small,.resource-kpi span{display:block;color:var(--muted);font-size:.76rem}.resource-kpi strong{display:block;margin:7px 0 3px;font:700 1.55rem var(--font-mono);color:var(--ink)}.resource-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.resource-card-title{display:flex;justify-content:space-between;gap:10px;align-items:center;margin-bottom:12px}.resource-card-title h3{margin:0;font-size:1rem}.resource-badge{padding:4px 8px;border-radius:999px;background:var(--bg-subtle);color:var(--muted);font:500 .68rem var(--font-mono)}.resource-list{display:grid;grid-template-columns:1fr auto;gap:10px;margin:0}.resource-list dt{color:var(--muted);font-size:.82rem}.resource-list dd{margin:0;color:var(--ink);font:600 .82rem var(--font-mono);text-align:right}.resource-note{margin:0;color:var(--muted);font-size:.76rem}@media(max-width:760px){.resource-banner{display:grid}.resource-kpis,.resource-grid{grid-template-columns:1fr 1fr}}@media(max-width:480px){.resource-kpis,.resource-grid{grid-template-columns:1fr}}</style>`, template.HTMLEscapeString(data.Nonce)); err != nil {
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
	case "account":
		title, areaLabel = "Yönetici Hesap Ayarları", "HESAP BİLGİLERİ"
	case "media":
		title, areaLabel = "Medya & Asset Pipeline", "MEDYA KÜTÜPHANESİ"
	case "site":
		title, areaLabel = "Site & Yayın Yüzeyi", "SİTE YAPILANDIRMASI"
	default:
		title, areaLabel = "Yönetim Alanı", "STUDIO"
	}

	if err := AdminStart(w, title, areaLabel, data.Nonce, data.Area); err != nil {
		return err
	}

	nonce := template.HTMLEscapeString(data.Nonce)
	csrf := template.HTMLEscapeString(data.CSRF)
	email := template.HTMLEscapeString(data.Email)

	switch data.Area {
	case "developer":
		_, _ = fmt.Fprintf(w, `<style nonce="%s">
.dev-console{display:grid;gap:24px}
.dev-banner{display:flex;justify-content:space-between;align-items:center;padding:22px 26px;border:1px solid var(--line);border-radius:var(--radius-lg);background:linear-gradient(135deg,var(--surface),var(--bg-subtle))}
.dev-banner h2{margin:0 0 6px;font-size:1.3rem}
.dev-banner p{margin:0;color:var(--muted);font-size:.88rem;max-width:680px;line-height:1.5}
.dev-endpoints-grid{display:grid;grid-template-columns:1fr;gap:14px}
.dev-endpoint-card{border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);padding:18px 22px;display:grid;gap:12px;transition:border-color .15s}
.dev-endpoint-card:hover{border-color:var(--brand)}
.dev-ep-header{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px}
.dev-ep-route{display:flex;align-items:center;gap:10px}
.dev-method{font:700 .75rem var(--font-mono);padding:4px 8px;border-radius:6px;letter-spacing:.05em}
.dev-method-get{background:#e0f2fe;color:#0369a1}
.dev-method-post{background:#dcfce7;color:#15803d}
.dev-path{font:600 .95rem var(--font-mono);color:var(--ink)}
.dev-ep-actions{display:flex;gap:8px}
.dev-btn-sm{padding:6px 12px;font-size:.8rem;border-radius:6px;cursor:pointer;border:1px solid var(--line);background:var(--surface);color:var(--ink);font-weight:600;display:inline-flex;align-items:center;gap:6px;text-decoration:none}
.dev-btn-sm:hover{background:var(--bg-subtle);border-color:var(--brand)}
.dev-btn-primary{background:var(--brand);color:#fff;border-color:var(--brand)}
.dev-btn-primary:hover{opacity:.9}
.dev-preview-box{display:none;margin-top:10px;padding:14px;background:#0f172a;color:#38bdf8;border-radius:8px;font:500 .82rem var(--font-mono);white-space:pre-wrap;overflow-x:auto;max-height:260px}
.dev-two-col{display:grid;grid-template-columns:1fr 1fr;gap:18px}
.dev-code-card{border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);padding:20px}
.dev-code-card h3{margin:0 0 10px;font-size:1.05rem}
.dev-code-card p{margin:0 0 14px;color:var(--muted);font-size:.85rem;line-height:1.5}
.dev-code-snippet{background:#0f172a;color:#f8fafc;padding:14px;border-radius:8px;font:500 .82rem var(--font-mono);overflow-x:auto}
</style>
<div class="dev-console" data-testid="developer-area">
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
          <button class="dev-btn-sm" type="button" data-test-endpoint="/api/v1/commerce/products">▶ Canlı Test Et</button>
          <a class="dev-btn-sm" href="/api/v1/commerce/products" target="_blank">JSON Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.85rem;">E-Ticaret ürün kataloğu, aktif fiyatlar, varyant stokları ve görsellerin genel JSON projeksiyonu.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-get">GET</span>
          <span class="dev-path">/api/news?locale=tr</span>
        </div>
        <div class="dev-ep-actions">
          <button class="dev-btn-sm" type="button" data-test-endpoint="/api/news?locale=tr">▶ Canlı Test Et</button>
          <a class="dev-btn-sm" href="/api/news?locale=tr" target="_blank">JSON Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.85rem;">Yayınlanmış editoryal haberler, öne çıkan manşetler, etiketler ve medya referansları listesi.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-post">POST</span>
          <span class="dev-path">/api/graphql</span>
        </div>
        <div class="dev-ep-actions">
          <button class="dev-btn-sm" type="button" data-test-graphql>▶ Canlı Query Test Et</button>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.85rem;">Next.js, React veya mobil uygulamalar için tek istekte birleşik içerik ve katalog sorgulama uç noktası.</div>
      <pre class="dev-preview-box"></pre>
    </div>

    <div class="dev-endpoint-card">
      <div class="dev-ep-header">
        <div class="dev-ep-route">
          <span class="dev-method dev-method-get">GET</span>
          <span class="dev-path">/readyz &amp; /healthz</span>
        </div>
        <div class="dev-ep-actions">
          <button class="dev-btn-sm" type="button" data-test-endpoint="/readyz">▶ Readiness Test Et</button>
          <a class="dev-btn-sm" href="/readyz" target="_blank">Status Aç ↗</a>
        </div>
      </div>
      <div style="color:var(--muted);font-size:.85rem;">Kubernetes, Docker ve yük dengeleyiciler (Reverse Proxy) için canlılık ve veritabanı hazır bulunuşluk sondaları.</div>
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
      box.textContent = 'Hata: ' + err.message;
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
      box.textContent = 'GraphQL Hatası: ' + err.message;
      box.style.display = 'block';
      btn.disabled = false;
      btn.textContent = 'Tekrar Dene';
    });
}
</script>`, nonce, nonce)

	case "operations":
		_, _ = fmt.Fprintf(w, `<style nonce="%s">
.ops-grid{display:grid;gap:20px}
.ops-kpis{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px}
.ops-kpi{padding:18px 20px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.ops-kpi small{display:block;color:var(--muted);font-size:.78rem;font-weight:700;letter-spacing:.05em}
.ops-kpi strong{display:block;margin:6px 0 2px;font:700 1.45rem var(--font-mono);color:var(--ink)}
.ops-kpi span{display:block;color:var(--muted);font-size:.76rem}
.ops-two-col{display:grid;grid-template-columns:1.2fr 1fr;gap:18px}
.ops-card{padding:22px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.ops-card h3{margin:0 0 10px;font-size:1.1rem}
.ops-card p{margin:0 0 16px;color:var(--muted);font-size:.88rem;line-height:1.5}
.ops-action-row{display:flex;gap:10px;margin-top:14px}
.ops-cli-box{background:#0f172a;color:#38bdf8;padding:14px;border-radius:8px;font:500 .82rem var(--font-mono);overflow-x:auto;margin-top:10px}
@media(max-width:768px){.ops-kpis{grid-template-columns:1fr 1fr}.ops-two-col{grid-template-columns:1fr}}
</style>
<div class="ops-grid" data-testid="operations-area">
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
      <div style="font-size:.85rem;color:var(--ink);font-weight:600;margin-bottom:6px;">CLI Yedekleme Komutu:</div>
      <div class="ops-cli-box">hydro backup create --config hydro.yaml</div>
      <div style="font-size:.85rem;color:var(--ink);font-weight:600;margin:12px 0 6px;">Sağlık Kontrolü (Doctor):</div>
      <div class="ops-cli-box">hydro doctor --config hydro.yaml</div>
      <div class="ops-action-row">
        <a href="/admin/control#operations" class="button">Operasyon Kontrol Paneli ↗</a>
        <a href="/admin/developer/resources" class="btn-secondary" style="padding:9px 16px;font-size:.85rem;text-decoration:none;">Canlı Kaynak Metrikleri</a>
        <a href="/admin/commerce/refunds/pending" class="btn-secondary" style="padding:9px 16px;font-size:.85rem;text-decoration:none;">Bekleyen İadeler</a>
      </div>
    </div>

    <div class="ops-card">
      <h3>🛡️ Blue/Green Dağıtım &amp; Güvenlik Fensi</h3>
      <p>Blue/green geçişi ve geri alma, sürüm uyumluluğu ile işletim prosedürlerine bağlıdır. Canlı ortamda uygulanmadan önce geçiş planını test edin.</p>
      <ul style="margin:0 0 16px;padding-left:20px;color:var(--muted);font-size:.85rem;line-height:1.6;">
        <li>Şema ve uygulama sürümlerinin uyumluluğunu yayın öncesi doğrulayın.</li>
        <li>Medya, oturum ve arka plan işlerinin sürekliliği için geri dönüş planı hazırlayın.</li>
        <li>Geri yüklemeyi önce izole bir ortamda deneyin.</li>
      </ul>
      <div style="font-size:.82rem;padding:10px 14px;background:var(--bg-subtle);border-radius:8px;border:1px solid var(--line);">
        Sistem kimliği ve sürümü için çalışma zamanı durum uç noktasını kullanın.
      </div>
    </div>
  </div>
</div>`, nonce)

	case "security":
		_, _ = fmt.Fprintf(w, `<style nonce="%s">
.sec-grid{display:grid;gap:20px}
.sec-kpis{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}
.sec-kpi{padding:18px 20px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.sec-kpi small{display:block;color:var(--muted);font-size:.78rem;font-weight:700}
.sec-kpi strong{display:block;margin:6px 0 2px;font:700 1.45rem var(--font-mono);color:#16a34a}
.sec-kpi span{display:block;color:var(--muted);font-size:.76rem}
.sec-two-col{display:grid;grid-template-columns:1fr 1fr;gap:18px}
.sec-card{padding:22px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.sec-card h3{margin:0 0 10px;font-size:1.1rem}
.sec-card p{margin:0 0 16px;color:var(--muted);font-size:.88rem;line-height:1.5}
.sec-form-row{display:grid;gap:10px;margin-top:12px}
.sec-form-row label{display:grid;gap:4px;font-size:.85rem;font-weight:600;color:var(--ink)}
.sec-form-row input,.sec-form-row select{padding:9px 12px;border:1px solid var(--line);border-radius:6px;font-size:.9rem}
@media(max-width:768px){.sec-kpis,.sec-two-col{grid-template-columns:1fr}}
</style>
<div class="sec-grid" data-testid="security-area">
  <div class="sec-kpis">
    <div class="sec-kpi"><small>KİMLİK KORUMASI</small><strong>Argon2id</strong><span>Endüstri standardı hash &amp; salt</span></div>
    <div class="sec-kpi"><small>OTURUM GÜVENLİĞİ</small><strong>HttpOnly &amp; Lax</strong><span>XSS korumalı oturum çerezi</span></div>
    <div class="sec-kpi"><small>İÇERİK GÜVENLİĞİ (CSP)</small><strong>Strict Nonce</strong><span>default-src 'self' aktif</span></div>
  </div>

  <div class="sec-two-col">
    <div class="sec-card">
      <h3>✉️ Yeni Yönetici / Editör Davet Et</h3>
      <p>Yeni bir ekip üyesine Studio erişimi vermek için güvenli davet bağlantısı oluşturun:</p>
      <form class="sec-form-row" method="post" action="/admin/control#security">
        <input type="hidden" name="_csrf" value="%s">
        <input type="hidden" name="_action" value="invite">
        <label>E-posta Adresi <input type="email" name="email" required placeholder="yeni-editor@sirket.com"></label>
        <label>Erişim Rolü
          <select name="role">
            <option value="editor">Editor (İçerik &amp; Taslak Düzenleme)</option>
            <option value="admin">Admin (Tam Yönetim Yetkisi)</option>
          </select>
        </label>
        <button class="button" type="submit" style="margin-top:6px;justify-content:center;">Davet Bağlantısı Oluştur ✉️</button>
      </form>
    </div>

    <div class="sec-card">
      <h3>🔐 İki Faktörlü Doğrulama (MFA / TOTP)</h3>
      <p>Yönetici hesabınızı Google Authenticator veya 1Password gibi TOTP uygulamaları ile ek koruma altına alın.</p>
      <div style="background:var(--bg-subtle);border:1px solid var(--line);border-radius:8px;padding:14px;margin-bottom:14px;">
        <div style="font-size:.88rem;color:var(--ink);font-weight:600;">Oturum Açan: %s</div>
        <div style="font-size:.78rem;color:var(--muted);margin-top:2px;">Rol: Sistem Yöneticisi (Admin)</div>
      </div>
      <a href="/admin/control#security" class="btn-secondary" style="display:inline-block;padding:9px 16px;font-size:.85rem;text-decoration:none;">MFA Kurulumunu Yönet 🔑</a>
    </div>
  </div>
</div>`, nonce, csrf, email)

	case "permissions":
		_, _ = fmt.Fprintf(w, `<style nonce="%s">
.perm-grid{display:grid;gap:20px}
.perm-table-card{border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);overflow:hidden}
.perm-table{width:100%%;border-collapse:collapse;text-align:left;font-size:.88rem}
.perm-table th{background:var(--bg-subtle);padding:14px 18px;font-weight:700;color:var(--ink);border-bottom:1px solid var(--line)}
.perm-table td{padding:14px 18px;border-bottom:1px solid var(--line);color:var(--ink)}
.perm-table tr:last-child td{border-bottom:none}
.badge-perm{display:inline-flex;align-items:center;gap:4px;padding:4px 8px;border-radius:6px;font-weight:700;font-size:.75rem}
.badge-full{background:#dcfce7;color:#15803d}
.badge-edit{background:#e0f2fe;color:#0369a1}
.badge-view{background:#fef3c7;color:#b45309}
.badge-none{background:#f1f5f9;color:#64748b}
.perm-rules-card{padding:22px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.perm-rules-card h3{margin:0 0 8px;font-size:1.1rem}
.perm-rules-card p{margin:0 0 14px;color:var(--muted);font-size:.88rem;line-height:1.5}
</style>
<div class="perm-grid" data-testid="permissions-area">
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
    <p>Yetkilendirme kuralları yalnızca arayüzdeki menüleri gizlemekle kalmaz; tüm servis ve veri tabanı mutasyonlarında <code>auth.RequireAdmin</code> ve <code>auth.RequireRole</code> katmanları tarafından zorunlu olarak doğrulanır.</p>
    <div style="display:flex;gap:10px;flex-wrap:wrap;">
      <a href="/admin/security" class="button">Güvenlik &amp; Davet Merkezi →</a>
      <a href="/admin/control#security" class="btn-secondary" style="padding:9px 16px;font-size:.85rem;text-decoration:none;">Kullanıcı Yetkilerini Düzenle</a>
    </div>
  </div>
</div>`, nonce)

	case "account":
		_, _ = fmt.Fprintf(w, `<style nonce="%s">
.acc-grid{display:grid;grid-template-columns:1fr 1fr;gap:20px}
.acc-card{padding:24px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface)}
.acc-card h3{margin:0 0 10px;font-size:1.15rem}
.acc-card p{margin:0 0 16px;color:var(--muted);font-size:.88rem;line-height:1.5}
.acc-info-row{display:grid;grid-template-columns:120px 1fr;gap:10px;padding:8px 0;border-bottom:1px solid var(--line);font-size:.88rem}
.acc-info-row:last-child{border-bottom:none}
.acc-info-label{color:var(--muted);font-weight:600}
.acc-info-val{color:var(--ink);font-weight:700}
@media(max-width:768px){.acc-grid{grid-template-columns:1fr}}
</style>
<div class="acc-grid">
  <div class="acc-card">
    <h3>👤 Yönetici Profil Bilgileri</h3>
    <p>Admin Studio paneline erişim sağlayan mevcut yetkili hesap:</p>
    <div style="margin:16px 0;">
      <div class="acc-info-row"><span class="acc-info-label">E-Posta:</span><span class="acc-info-val">%s</span></div>
      <div class="acc-info-row"><span class="acc-info-label">Rol:</span><span class="acc-info-val" style="color:var(--brand);">👑 Sistem Yöneticisi (Admin)</span></div>
      <div class="acc-info-row"><span class="acc-info-label">Oturum Tipi:</span><span class="acc-info-val">HttpOnly Güvenli Oturum</span></div>
      <div class="acc-info-row"><span class="acc-info-label">Erişim Alanı:</span><span class="acc-info-val">Tüm Modüller &amp; Operasyon</span></div>
    </div>
    <div style="margin-top:20px;display:flex;gap:10px;">
      <a href="/admin/security" class="button">Güvenlik &amp; MFA Ayarları</a>
      <a href="/login?return_to=%%2Fadmin" class="btn-secondary" style="padding:9px 16px;font-size:.85rem;text-decoration:none;">Oturumu Kapat</a>
    </div>
  </div>

  <div class="acc-card">
    <h3>🔒 Hesap Güvenliği &amp; Parola Yenileme</h3>
    <p>Yönetici parolanız <code>Argon2id</code> algoritması ile korunmaktadır. Güvenlik politikası gereği güçlü parolalar (en az 8 karakter) zorunludur.</p>
    <div style="background:var(--bg-subtle);border:1px solid var(--line);border-radius:8px;padding:16px;margin-top:14px;">
      <div style="font-weight:700;font-size:.9rem;color:var(--ink);margin-bottom:6px;">Parola Değiştirme:</div>
      <div style="font-size:.82rem;color:var(--muted);line-height:1.5;">Parolanızı yenilemek için Güvenlik Merkezinden şifre sıfırlama akışını başlatabilir veya CLI üzerinden <code>hydro user reset-password</code> komutunu çalıştırabilirsiniz.</div>
    </div>
  </div>
</div>`, nonce, email)

	default:
		_, _ = fmt.Fprintf(w, `<div class="studio-card"><p>Yönetim alanı yükleniyor...</p></div>`)
	}

	return AdminEnd(w)
}
