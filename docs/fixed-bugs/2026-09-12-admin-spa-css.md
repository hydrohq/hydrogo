# Admin Studio SPA CSS İlk Açılış Düzeltmeleri

## Belirti

Admin Studio içinde `API & Entegrasyon`, `Operasyon`, `Güvenlik`, `Yetkiler` ve `Kaynak tüketimi` sayfaları sidebar üzerinden ilk kez açıldığında sayfaya özel CSS uygulanmıyor, içerik düz blok olarak görünüyor ve tarayıcı yenilendiğinde düzeliyordu.

## Kök neden

Admin sayfaları SPA router tarafından yeni bir HTTP yanıtından fragment olarak alınıyordu. Fragment içindeki stiller yeni yanıtın CSP nonce değerini taşıyordu; mevcut dokümanın CSP politikası ise ilk sayfanın nonce değerini kabul ediyordu. Ek olarak fragment'in `innerHTML` üzerinden yeniden serialize edilmesi, tarayıcının nonce attribute'unu gizleyerek normalize edilmiş nonce'u da düşürebiliyordu.

## Uygulanan düzeltmeler

- SPA router, fragment stillerini DOM'a eklenmeden önce mevcut dokümanın nonce değeriyle normalize ediyor.
- Fragment aktarımı `innerHTML` yerine `replaceChildren(...Array.from(newMain.childNodes))` ile gerçek DOM node'ları taşınarak yapılıyor; nonce değerleri serialization sırasında kaybolmuyor.
- Kaynak tüketimi, geliştirici alanları ve Yetkiler ekranının eksik stilleri global Admin Studio CSS paketine alındı.
- Yetkiler tablosu, rozetleri, kurallar kartı ve mobil taşma davranışı global CSS'e eklendi.
- E2E manifesti doğrudan URL yerine gerçek sidebar SPA geçişlerini de doğruluyor.
- `computed-style` kontrolleriyle ilgili sayfalarda grid/table stillerinin gerçekten uygulandığı test ediliyor.

## Kanıt

- `internal/ui/admin_shell.go`
- `internal/ui/admin_areas.go`
- `tools/e2egen/main.go`
- `e2e/manifest.json`
- `TASKS.md`

Doğrulama sonuçları:

- `go test ./internal/ui ./tools/e2egen -p 1 -count=1` başarılı.
- `go vet ./...` başarılı.
- `scripts/run-e2e.ps1` sonucu: Chromium üzerinde `14/14 PASS`.
- Admin alt sayfaları gerçek `/admin/developer` sidebar geçişiyle test edildi.

## Operatör notu

Kod değişikliğinin çalışan dev sunucusuna alınması için devserver yeniden başlatılmalıdır:

```powershell
go run ./tools/devserver
```

Tarayıcı cache'ini temizlemek için ardından `Ctrl + F5` kullanılabilir.
