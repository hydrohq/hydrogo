# Frontend nasıl geliştirilir (insan rehberi)

Bu projede aslında **iki ayrı "frontend" sistemi** var, birbirine
karıştırma:

| | Site görünümü (tema) | Admin paneli |
|---|---|---|
| Kimin görür | Site ziyaretçileri | Senin ekibin/editörler |
| Nerede | `themes/<tema-adı>/` | `modules/designsystem/`, `internal/ui/` |
| Yazım dili | Düz HTML | Templ (özel bir şablon dili) |
| Stil | Tailwind, tema başına derleniyor | Tailwind, panel paketi başına derleniyor |
| Değişiklik ne zaman görünür | Küçük bir derleme komutu gerekir | `hydro run` çalışırken otomatik |

## Site görünümünü değiştirmek (tema)

`themes/corporate/` gibi bir klasörün içinde:
```
layouts/base.html        ← sayfanın genel iskeleti (header/footer vb.)
blocks/hero.html          ← ana sayfa gibi yerlerde kullanılan "blok"lar
blocks/cta.html
blocks/faq.html
...
static/theme.source.css  ← stil DEĞİŞİKLİKLERİNİ buraya yap
static/theme.css         ← bu otomatik üretiliyor, elle dokunma
```

HTML dosyalarını (blocks/, layouts/) direkt düzenleyebilirsin, ekstra bir
adım gerekmiyor. **Renk/stil değişikliği** yapacaksan `theme.source.css`'i
düzenle, sonra bir ajana "tema CSS'ini derle" de — küçük bir script
çalıştırıp senin için derliyor.

**Önemli bir detay:** CSS'i her derledikten sonra, dosyanın "önbellek
kırma" numarasının (`theme.css?v=abc12345` gibi bir şey) üç ayrı yerde
güncellenmesi gerekiyor — bunu unutursan tarayıcılar eski CSS'i
önbellekten göstermeye devam edebilir. Bir ajana yaptırıyorsan bu adımı
atlamaması gerektiğini hatırlat, kendi script'i zaten sana söylüyor.

## Admin panelini değiştirmek

Admin panelinin görsel bileşenleri "Templ" denen özel bir dilde yazılıyor
(`.templ` uzantılı dosyalar). **`hydro run` çalışırken bu dosyaları
kaydettiğinde otomatik olarak derlenip panelde anında görünüyor** —
ekstra bir komut çalıştırman gerekmiyor, sadece dosyayı kaydet, tarayıcıyı
yenile.

`_templ.go` diye biten dosyalara **asla elle dokunma** — bunlar otomatik
üretiliyor, senin elle yaptığın değişiklik bir sonraki derlemede silinir.

## Değişikliği doğrulamak

En güvenilir yöntem: `hydro run`'ı çalıştır, değiştirdiğin sayfayı
tarayıcıda gerçekten aç ve gör. Otomatik kontroller (lint, derleme) sadece
"bozuk değil" der, "güzel görünüyor" demez — görsel değişiklikleri mutlaka
gözle kontrol et.
