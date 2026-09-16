# Headless starter istemci sertleştirme kanıtı (2026-09-16)

Generator tarafından üretilen `client/hydrogo.ts` artık generated collection
istemcileriyle aynı temel sınırları uygular:

- 100 ms–30 s aralığında, varsayılan 10 s `AbortController` timeout'u;
- `Content-Length` ve okunan gövde için 2 MiB response sınırı;
- bounded locale/slug girdileri (256 Unicode karakter);
- HTTP hata gövdesinde typed `HydroAPIError` ve güvenli fallback mesajı.

`go test ./internal/projectgen -count=1` generator çıktısında bu sözleşme
işaretlerini doğrular. Üretilen client dosyası `client/` altında tutulur ve Core
runtime'a bağımlılık eklemez.
