package ui

import (
	"fmt"
	"html/template"
	"io"
)

type AdminAreaData struct {
	Email string
	Nonce string
	CSRF  string
	Area  string
}

var adminAreaTemplate = template.Must(template.New("admin-area").Parse(`<section class="dashboard-grid"><article class="studio-card dashboard-card"><p class="eyebrow">{{.PrimaryEyebrow}}</p><h2>{{.PrimaryTitle}}</h2><p class="muted">{{.PrimaryText}}</p><div class="collection-grid">{{range .Links}}<a class="collection-tile" href="{{.Href}}" {{if .External}}target="_blank"{{end}}><strong>{{.Label}}</strong><small>{{.Description}}</small></a>{{end}}</div></article><article class="studio-card dashboard-card area-purpose"><p class="eyebrow">BU ALAN NE İÇİN?</p><h2>{{.PurposeTitle}}</h2><p class="muted">{{.PurposeText}}</p><ul>{{range .Checklist}}<li>{{.}}</li>{{end}}</ul></article></section>`))

type adminAreaLink struct {
	Label       string
	Description string
	Href        string
	External    bool
}

type adminAreaView struct {
	AdminAreaData
	AreaLabel      string
	Title          string
	Description    string
	PrimaryEyebrow string
	PrimaryTitle   string
	PrimaryText    string
	PurposeTitle   string
	PurposeText    string
	Checklist      []string
	Links          []adminAreaLink
}

func RenderAdminArea(w io.Writer, data AdminAreaData) error {
	view := adminAreaView{AdminAreaData: data}
	switch data.Area {
	case "site":
		view.AreaLabel, view.Title, view.Description = "SITE YÖNETİMİ", "Site ayarları ve yayın yüzeyi", "Navigation, public ayarlar, redirect ve form inbox yönetimi."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "YAPILANDIRMA", "Site kontrol paneli", "Site ayarı ve redirect kayıtları Yönetim araçları ekranındaki CSRF korumalı formlardan yapılabilir."
		view.Links = []adminAreaLink{{"Ayar ve redirect formları", "Kayıt işlemlerini aç", "/admin/control#site", false}, {"Public ayarlar", "JSON çıktısını incele", "/site/settings", true}, {"TR navigation", "Yayınlanan menü", "/site/navigation/tr", true}, {"Form inbox", "Contact kayıtları", "/admin/site/inbox/contact", true}}
		view.PurposeTitle, view.PurposeText = "Yayın ve yönlendirme yapılandırması", "Bu alan site verisinin API üzerinden nasıl sunulacağını yönetir; son kullanıcı arayüzü burada tasarlanmaz."
		view.Checklist = []string{"Site ayarı ve yayınlanan navigation verisini kontrol edin.", "Redirect veya form inbox işlemi gerektiğinde Yönetim Araçları'na geçin.", "Bağımsız frontend bu verileri API'den tüketir."}
	case "media":
		view.AreaLabel, view.Title, view.Description = "MEDYA", "Medya kütüphanesi", "Görsel yükleme, güvenli doğrulama ve pending → ready asset yaşam döngüsü."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "ASSET PIPELINE", "Medya işlemleri", "JPEG, PNG ve WebP dosyaları boyut, MIME ve görüntü imzası kontrollerinden geçer."
		view.Links = []adminAreaLink{{"Medya yükleme formu", "Yönetim araçlarında aç", "/admin/control#media", false}, {"Media API", "Protected upload endpoint", "/api/media", true}}
		view.PurposeTitle, view.PurposeText = "Dosyaların güvenli yaşam döngüsü", "Bu ekran dosya yükleme ve medya API'sine erişim içindir; içerik içindeki görsel seçimini Pages ve News editörleri yapar."
		view.Checklist = []string{"Yalnız desteklenen dosya türlerini yükleyin.", "Yüklemenin ready durumuna geçmesini bekleyin.", "Dönen asset kimliğini içerik alanında kullanın."}
	case "security":
		view.AreaLabel, view.Title, view.Description = "GÜVENLİK", "Kullanıcı ve erişim güvenliği", "Davet, MFA, oturum ve parola kurtarma akışları."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "ACCESS CONTROL", "Güvenlik merkezi", "Yönetici işlemleri default-deny, CSRF ve audit koruması üzerinden yürütülür."
		view.Links = []adminAreaLink{{"Editör daveti", "Davet formunu aç", "/admin/control#security", false}, {"Oturum bilgisi", "Mevcut hesabı incele", "/auth/me", true}, {"MFA ve hesap API", "Güvenli endpoint’ler", "/auth/me/mfa/enroll", true}}
		view.PurposeTitle, view.PurposeText = "Hesap riskini ve erişimi yönetme", "Buradan editör daveti başlatılır ve oturum/MFA durumları kontrol edilir. Rol politikasını değiştirmek için Güvenlik ve Yetkiler alanını kullanın."
		view.Checklist = []string{"Yeni bir editör için davet oluşturun.", "Mevcut hesabın oturum ve MFA durumunu kontrol edin.", "Rol kapsamını Güvenlik ve Yetkiler ekranında inceleyin."}
	case "account":
		view.AreaLabel, view.Title, view.Description = "HESAP AYARLARI", "Yönetici hesabı", "Oturum ve hesap güvenliği için mevcut kullanıcı bilgileri."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "CURRENT USER", "Hesap özeti", "Bu hesabın rolü ve aktif oturum bilgisi güvenli API üzerinden görüntülenebilir."
		view.Links = []adminAreaLink{{"Hesap bilgileri", data.Email, "/auth/me", true}, {"Güvenlik ayarları", "MFA ve erişim yönetimi", "/admin/security", false}, {"Oturumu kapat", "Güvenli çıkış", "/login?return_to=%2Fadmin", false}}
		view.PurposeTitle, view.PurposeText = "Kendi yönetici hesabınız", "Bu alan yalnızca oturum açmış kullanıcıya ait hesap bilgisini ve güvenlik ayarlarını yönlendirir."
		view.Checklist = []string{"Hesap bilgisini doğrulayın.", "MFA ve erişim işlemleri için Güvenlik'e gidin.", "Ortak bilgisayarda işiniz bittiğinde çıkış yapın."}
	case "permissions":
		view.AreaLabel, view.Title, view.Description = "GÜVENLİK VE YETKİLER", "Rol erişim matrisi", "Hangi rolün hangi çalışma alanında ne yapabildiğini bu matriste görün."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "ROLE POLICY", "Yetki matrisi", "Yetkilendirme yalnızca menü görünürlüğü değil, her application service mutasyonunda da uygulanır."
		view.Links = []adminAreaLink{{"Admin", "Tüm içerik, commerce, site ve operasyon yönetimi", "/admin", false}, {"Editor", "Haber ve sayfa taslağı oluşturma/düzenleme; yayın yetkisi politika kontrollüdür", "/admin/news", false}, {"Viewer", "Yayınlanmış içeriklerin salt okunur JSON projection'ı", "/api/news?locale=tr", true}, {"Güvenlik merkezi", "Davet ve MFA akışları", "/admin/security", false}}
		view.PurposeTitle, view.PurposeText = "Rolün gerçekte neye izin verdiği", "Bu bir rol ve erişim referansıdır. Yetki sadece menü görünürlüğü değildir; içerik, sipariş ve güvenlik işlemlerinde servis katmanında da doğrulanır."
		view.Checklist = []string{"İçerik yazacak kişiye Editor rolü verin.", "Tam yönetim gerektiren kişileri Admin olarak sınırlandırın.", "Yayınlanmış veriyi yalnız okuyan tüketiciler için Viewer erişimini kullanın."}
	case "developer":
		view.AreaLabel, view.Title, view.Description = "GELİŞTİRİCİ", "Geliştirici araçları", "API, headless tüketim ve sistem tanılarını ayrı bir çalışma alanında yönetin."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "DEVELOPER TOOLBOX", "Teknik çalışma alanı", "Bu ekran uygulama geliştiricileri içindir; günlük içerik ve commerce işlemleri dashboard’da sade tutulur."
		view.Links = []adminAreaLink{{"Commerce REST API", "Ürün ve katalog JSON uçlarını incele", "/api/v1/commerce/products", true}, {"GraphQL", "Headless sorgu yüzeyini aç", "/api/graphql", true}, {"Health check", "Servis canlılığını doğrula", "/healthz", true}, {"Readiness check", "Veritabanı ve migration hazır oluşu", "/readyz", true}, {"Metrikler", "Authenticated runtime metrikleri", "/metrics", true}, {"News API", "Yayınlanmış haber JSON uçları", "/api/news?locale=tr", true}}
		view.PurposeTitle, view.PurposeText = "Frontend ve entegrasyon geliştirme başlangıç noktası", "Bu ekran içerik girmek için değil; ayrı bir web, mobil uygulama veya entegrasyonun HydroGo API'lerine bağlanması için teknik referanstır."
		view.Checklist = []string{"Frontend'iniz için REST veya GraphQL sözleşmesini doğrulayın.", "Ürün, haber ve sayfa JSON'larını örnek veriyle inceleyin.", "Health ve readiness uçlarını deployment/monitoring sistemine bağlayın."}
	default:
		view.AreaLabel, view.Title, view.Description = "OPERASYON", "Sağlık ve operasyon merkezi", "Health, readiness, metrik, background job ve bakım akışları."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "OBSERVABILITY", "Platform operasyonu", "Sistem görünürlüğü ve bakım uçları tek operasyon ekranında toplanır."
		view.Links = []adminAreaLink{{"Raporlar", "Platform metrik özeti", "/admin/reports", false}, {"Metrikler", "Authenticated metrics endpoint", "/metrics", true}, {"Dead-letter işler", "CLI ile bounded inceleme", "/admin/control#operations", false}, {"Backup / restore", "hydro CLI operasyonları", "/admin/control#operations", false}}
		view.PurposeTitle, view.PurposeText = "Sistemi çalışır ve kurtarılabilir tutma", "Operasyon alanı günlük içerik yönetimi için değildir. Canlılık, hazır oluş, metrikler, başarısız arka plan işleri ve yedekleme kararları burada toplanır."
		view.Checklist = []string{"Sorun anında önce Raporlar, health ve readiness durumunu kontrol edin.", "Başarısız arka plan işleri için Dead-letter incelemesini kullanın.", "Yedekleme ve geri yükleme işlemlerini onaylı CLI akışıyla yapın."}
	}
	if err := AdminStart(w, view.Title, view.AreaLabel, data.Nonce, data.Area); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, `<style nonce="%s">.area-purpose ul{display:grid;gap:10px;margin:18px 0 0;padding:0;list-style:none}.area-purpose li{position:relative;padding-left:22px;color:var(--ink-secondary);font-size:.86rem;line-height:1.5}.area-purpose li:before{content:"";position:absolute;left:0;top:.48em;width:8px;height:8px;border:2px solid var(--brand);border-radius:50%%}</style>`, template.HTMLEscapeString(data.Nonce)); err != nil {
		return fmt.Errorf("render admin area styles: %w", err)
	}
	if err := adminAreaTemplate.Execute(w, view); err != nil {
		return fmt.Errorf("render admin area: %w", err)
	}
	return AdminEnd(w)
}
