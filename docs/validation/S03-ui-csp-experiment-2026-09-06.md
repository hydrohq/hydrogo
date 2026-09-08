# S03 UI Stack, CSP ve Medya Yükleme Doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 / S03  
**Durum:** Başarılı; yerel Quill/HTMX asset'leri ve CSP uyumlu editör fixture'ı doğrulandı.

## Doğrulanan Davranışlar

1. **Templ ve View Slot Sözleşmesi:**
   - `extension.Theme` sözleşmesi (`RenderPage`, `RenderHeader`, `RenderFooter`) tanımlandı.
   - `examples/client-edition` içinde `DefaultTheme` ve `MinimalTheme` olmak üzere iki farklı tema uygulandı.
   - Core paketlerine (`internal/`) hiçbir bağımlılık olmadan harici Go modülü olarak derlendi.

2. **Zengin Metin & HTML Sanitizer:**
   - `bluemonday` UGC politikası temel alınarak `HTMLSanitizer` oluşturuldu.
   - `<script>`, `onload`, `onerror`, `javascript:` URI ve gömülü SVG/XSS saldırı vektörlerinin temizlendiği; güvenli HTML etiketlerinin korunduğu birim testleriyle (`internal/ui/sanitizer_test.go`) kanıtlandı.

3. **Sıkı CSP & Dinamik Nonce:**
   - Her istek için dinamik rastgele base64 nonce üretildi ve şablonların inline `<style nonce="...">` ve `<script nonce="...">` bloklarına güvenle aktarılması sağlandı.

4. **Güvenli Medya Yükleme Deneyi:**
   - `internal/media.Uploader` ile güvenli yükleme boru hattı kuruldu:
      - İlk 512 byte üzerinden içerik koklama (`http.DetectContentType`) ile gerçek MIME tipi doğrulandı (JPEG, PNG, WebP).
     - PHP/çalıştırılabilir dosyalar ve izin verilmeyen MIME türleri reddedildi.
     - Standart Go `image.DecodeConfig` ile görsel bütünlüğü ve boyutları doğrulandı; bozuk görseller geçici dosya temizliğiyle reddedildi.
     - Yol geçişi (`path traversal`) saldırıları engellendi; içerik tabanlı SHA-256 dosya adlandırması uygulandı.
     - Maksimum dosya boyutu sınırları test edildi.

5. **Quill/HTMX yerel asset ve CSP fixture'ı:**
   - Quill 2.0.3 (`static/vendor/quill`) ve HTMX 2.0.10 (`static/vendor/htmx`) CDN'e ihtiyaç duymadan repoya sabitlendi.
   - `internal/ui.RenderEditor` nonce'lu inline başlatma ile Quill Delta JSON alanı, CSRF alanı ve HTMX form hedefini üretir.
   - `allowEval=false` ve `allowScriptTags=false` ayarları ile HTMX dinamik değerlendirmesi kapatıldı; testte CDN URL'leri ve inline event handler'lar reddedilir.

## Çalıştırılan Komutlar

```text
go test -v ./internal/ui/...
go test -v ./internal/media/...
cd examples/client-edition && go build ./...
go run ./tools/importguard -root examples/client-edition
```
Tüm adımlar hatasız geçti. Sürümlenen üçüncü taraf lisansları `static/vendor/*/LICENSE` altındadır.
