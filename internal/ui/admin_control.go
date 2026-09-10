package ui

import (
	"html/template"
	"io"
)

type AdminControlData struct {
	Email string
	Nonce string
}

var adminControlTemplate = template.Must(template.New("admin-control").Parse(`<section class="dashboard-welcome"><div><p class="eyebrow">YÖNETİM ARAÇLARI</p><h2>Platformun tamamını tek yerden yönetin.</h2><p class="dashboard-subtitle">Site, medya, güvenlik ve operasyon yetenekleri aynı Studio çalışma alanında.</p></div></section>
<section class="dashboard-grid"><article class="studio-card dashboard-card"><div class="control-card-icon control-blue">◉</div><p class="eyebrow">SITE</p><h2>Site yönetimi</h2><p class="muted">Ayarlar, navigation, redirect ve form inbox akışları.</p><div class="collection-grid"><a class="collection-tile" href="/site/settings" target="_blank"><strong>Public ayarlar ↗</strong></a><a class="collection-tile" href="/site/navigation/tr" target="_blank"><strong>TR navigation ↗</strong></a><a class="collection-tile" href="/admin/control#site"><strong>Yönetim akışları →</strong></a></div></article>
<article class="studio-card dashboard-card"><div class="control-card-icon control-purple">▧</div><p class="eyebrow">ASSETLER</p><h2>Medya kütüphanesi</h2><p class="muted">Güvenli görsel yükleme ve pending → ready yaşam döngüsü.</p><div class="collection-grid"><a class="collection-tile" href="/admin/control#media"><strong>Medya merkezi →</strong></a><a class="collection-tile" href="/admin/control#media-api"><strong>Media API ↗</strong></a></div></article>
<article class="studio-card dashboard-card"><div class="control-card-icon control-green">◇</div><p class="eyebrow">ERİŞİM</p><h2>Kullanıcı ve güvenlik</h2><p class="muted">Davetiye, MFA, oturum ve hesap güvenliği akışları.</p><div class="collection-grid"><a class="collection-tile" href="/admin/control#security"><strong>Güvenlik merkezi →</strong></a><a class="collection-tile" href="/auth/me" target="_blank"><strong>Oturum bilgisi ↗</strong></a></div></article>
<article class="studio-card dashboard-card"><div class="control-card-icon control-orange">◌</div><p class="eyebrow">OPERASYON</p><h2>Sağlık ve işler</h2><p class="muted">Health, readiness, metrik, audit ve background job uçları.</p><div class="collection-grid"><a class="collection-tile" href="/admin/control#operations"><strong>Operasyon merkezi →</strong></a><a class="collection-tile" href="/healthz" target="_blank"><strong>Healthz ↗</strong></a><a class="collection-tile" href="/metrics" target="_blank"><strong>Metrikler ↗</strong></a></div></article></section>
<section class="studio-card control-next"><div><p class="eyebrow">HIZLI İŞLEM</p><h2>Yönetim endpoint’leri hazır</h2><p class="muted">Formlar CSRF koruması ve mevcut application service katmanı üzerinden çalışır.</p></div><div class="studio-actions"><a class="button" href="/admin/news#new-news">Yeni haber</a><a class="button button-quiet" href="/admin/pages">Sayfaları aç</a></div></section>`))

func RenderAdminControl(w io.Writer, data AdminControlData) error {
	if err := AdminStart(w, "Yönetim araçları", "Platform yönetimi", data.Nonce, "dashboard"); err != nil {
		return err
	}
	if err := adminControlTemplate.Execute(w, data); err != nil {
		return err
	}
	return AdminEnd(w)
}
