# HydroGo Contribution Rules

Bu depoda mimari veya ürün davranışı değiştiren çalışmaya başlamadan önce aşağıdaki belgeleri oku:

1. `docs/HYDROGO_PHILOSOPHY.md`
2. `docs/implementation_plan.md`
3. `docs/architecture/README.md`
4. Değişikliğin etkilediği ADR dosyaları

## Mimari disiplin

- Kabul edilmiş bir ADR ile çelişen değişikliği sessizce uygulama.
- Yeni bilgi eski kararı geçersiz kılıyorsa eski ADR'yi silme; yeni ADR oluştur, önceki kararı “Değiştirildi” olarak işaretle ve iki belgeyi bağla.
- Müşteri ihtiyacı için HydroCore fork etme. Genel yeteneği sürümlü Core/Module sözleşmesine, özel davranışı Client Edition'a yerleştir.
- Etkin olmayan modül route, migration, asset, worker veya kaynak yükü oluşturmamalıdır.
- Domain ve application service koduna SQL dialect veya cache sürücüsü ayrıntısı sızdırma.

## Değişiklik kontrolü

Yeni veya değişen bir modül için şunları değerlendir:

- SQLite/PostgreSQL contract davranışı ve gelecekteki adaptörlere etkisi
- Dialect migration'ları ve rollback/uyumluluk metadata'sı
- Cache key, TTL, invalidation, stampede ve arıza fallback'i
- Compact profilinde gereksiz servis ve sınırsız kaynak tüketimi oluşmaması
- Büyük profilde connection pool, worker ve backpressure davranışı
- Blue/green sırasında eski ve yeni sürüm uyumluluğu
- Session, medya, asset ve background job sürekliliği
- Güvenlik, audit ve veri yaşam döngüsü

## Tamamlanmış iş

Bir mimari değişiklik; ilgili ADR, uygulama, sertifikalı veritabanı davranış testleri, gerekli migration metadata'sı ve kapasite/deployment etkisi birlikte tamamlanmadan bitmiş sayılmaz.
