package ui

import "io"

// RenderComponentShowcase provides a small, dependency-free visual contract
// surface for theme providers. It is intentionally semantic: a provider may
// restyle these primitives without changing the markup contract.
func RenderComponentShowcase(w io.Writer, nonce string) error {
	if err := AdminStart(w, "Component Showcase", "Tasarım sistemi", nonce, "developer"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, renderDeveloperTabs("components")); err != nil {
		return err
	}
	content := `<section class="showcase-grid" aria-label="UI component showcase">
<article class="studio-card showcase-card"><h2>Button ve durumlar</h2><div class="showcase-row"><button class="button">Varsayılan</button><button class="button button-secondary">İkincil</button><button class="button" disabled aria-disabled="true">Devre dışı</button></div></article>
<article class="studio-card showcase-card"><h2>Form alanları</h2><div class="showcase-form"><div class="form-field"><label for="showcase-name">Ad</label><input id="showcase-name" name="name" placeholder="Bir değer girin"></div><div class="form-field"><label for="showcase-error">Hata örneği</label><input id="showcase-error" name="error" aria-invalid="true"><p class="form-error" role="alert">Bu alanı kontrol edin.</p></div></div></article>
<article class="studio-card showcase-card"><h2>Durumlar</h2><div class="showcase-row"><div class="ui-empty" role="status">Kayıt bulunamadı.</div><div class="ui-loading" role="status" aria-live="polite">Yükleniyor…</div><div class="ui-error" role="alert">İşlem tamamlanamadı.</div></div></article>
<article class="studio-card showcase-card"><h2>Erişilebilirlik sözleşmesi</h2><ul><li>Görünür klavye odağı</li><li>Semantic label ve heading</li><li>Reduced-motion desteği</li><li>RTL ve uzun metin uyumu</li></ul></article>
</section>`
	_, err := io.WriteString(w, content)
	if err != nil {
		return err
	}
	return AdminEnd(w)
}
