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

var adminAreaTemplate = template.Must(template.New("admin-area").Parse(`<section class="dashboard-welcome"><div><p class="eyebrow">{{.AreaLabel}}</p><h2>{{.Title}}</h2><p class="dashboard-subtitle">{{.Description}}</p></div></section><section class="dashboard-grid"><article class="studio-card dashboard-card"><p class="eyebrow">{{.PrimaryEyebrow}}</p><h2>{{.PrimaryTitle}}</h2><p class="muted">{{.PrimaryText}}</p><div class="collection-grid">{{range .Links}}<a class="collection-tile" href="{{.Href}}" {{if .External}}target="_blank"{{end}}><strong>{{.Label}}</strong><small>{{.Description}}</small></a>{{end}}</div></article><article class="studio-card dashboard-card"><p class="eyebrow">DURUM</p><h2>Bağlantılar hazır</h2><p class="muted">Bu ekran, ilgili yönetim servislerinin durumunu ve güvenli işlem girişlerini tek yerde toplar.</p><div class="tool-grid"><a class="tool-tile" href="/healthz" target="_blank"><strong>Healthz ↗</strong><small>Proses canlılık kontrolü</small></a><a class="tool-tile" href="/readyz" target="_blank"><strong>Readyz ↗</strong><small>Veritabanı ve migration hazır oluşu</small></a></div></article></section><section class="studio-card dashboard-card"><div class="dashboard-card-head"><div><p class="eyebrow">SONRAKİ İŞLEM</p><h2>Gelişmiş işlem merkezi</h2></div><a class="text-link" href="/admin/control">Tüm yönetim araçları →</a></div><p class="muted">CSRF korumalı işlemler, mevcut servis katmanları ve audit akışları üzerinden çalışır.</p></section>`))

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
	Links          []adminAreaLink
}

func RenderAdminArea(w io.Writer, data AdminAreaData) error {
	view := adminAreaView{AdminAreaData: data}
	switch data.Area {
	case "site":
		view.AreaLabel, view.Title, view.Description = "SITE YÖNETİMİ", "Site ayarları ve yayın yüzeyi", "Navigation, public ayarlar, redirect ve form inbox yönetimi."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "YAPILANDIRMA", "Site kontrol paneli", "Site ayarı ve redirect kayıtları Yönetim araçları ekranındaki CSRF korumalı formlardan yapılabilir."
		view.Links = []adminAreaLink{{"Ayar ve redirect formları", "Kayıt işlemlerini aç", "/admin/control#site", false}, {"Public ayarlar", "JSON çıktısını incele", "/site/settings", true}, {"TR navigation", "Yayınlanan menü", "/site/navigation/tr", true}, {"Form inbox", "Contact kayıtları", "/admin/site/inbox/contact", true}}
	case "media":
		view.AreaLabel, view.Title, view.Description = "MEDYA", "Medya kütüphanesi", "Görsel yükleme, güvenli doğrulama ve pending → ready asset yaşam döngüsü."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "ASSET PIPELINE", "Medya işlemleri", "JPEG, PNG ve WebP dosyaları boyut, MIME ve görüntü imzası kontrollerinden geçer."
		view.Links = []adminAreaLink{{"Medya yükleme formu", "Yönetim araçlarında aç", "/admin/control#media", false}, {"Media API", "Protected upload endpoint", "/api/media", true}}
	case "security":
		view.AreaLabel, view.Title, view.Description = "GÜVENLİK", "Kullanıcı ve erişim güvenliği", "Davet, MFA, oturum ve parola kurtarma akışları."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "ACCESS CONTROL", "Güvenlik merkezi", "Yönetici işlemleri default-deny, CSRF ve audit koruması üzerinden yürütülür."
		view.Links = []adminAreaLink{{"Editör daveti", "Davet formunu aç", "/admin/control#security", false}, {"Oturum bilgisi", "Mevcut hesabı incele", "/auth/me", true}, {"MFA ve hesap API", "Güvenli endpoint’ler", "/auth/me/mfa/enroll", true}}
	default:
		view.AreaLabel, view.Title, view.Description = "OPERASYON", "Sağlık ve operasyon merkezi", "Health, readiness, metrik, background job ve bakım akışları."
		view.PrimaryEyebrow, view.PrimaryTitle, view.PrimaryText = "OBSERVABILITY", "Platform operasyonu", "Sistem görünürlüğü ve bakım uçları tek operasyon ekranında toplanır."
		view.Links = []adminAreaLink{{"Raporlar", "Platform metrik özeti", "/admin/reports", false}, {"Metrikler", "Authenticated metrics endpoint", "/metrics", true}, {"Dead-letter işler", "CLI ile bounded inceleme", "/admin/control#operations", false}, {"Backup / restore", "hydro CLI operasyonları", "/admin/control#operations", false}}
	}
	if err := AdminStart(w, view.Title, view.AreaLabel, data.Nonce, data.Area); err != nil {
		return err
	}
	if err := adminAreaTemplate.Execute(w, view); err != nil {
		return fmt.Errorf("render admin area: %w", err)
	}
	return AdminEnd(w)
}
