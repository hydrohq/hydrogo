# ADR-0246: Generated many-valued relations use bounded normalized storage

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** Blueprint'teki `many` relation alanlarının generated migration ve repository sözleşmesi
- **İlgili:** ADR-0230, ADR-0235, ADR-0245

## Karar

`cardinality: many` alanı kaynak tablosunda scalar sütun oluşturmaz. Generator,
deterministik isimli ayrı bir relation tablosu ve tenant-scoped kaynaklarda
`tenant_id` içeren composite primary key üretir. Listeleme tenant/source
predicate'i, deterministik sıra ve `max_items + 1` overflow kontrolü kullanır.
Replace metotları kendi transaction'larında atomiktir; `Replace…Tx` varyantı
Edition'ın daha büyük kaynak güncelleme transaction'ına katılabilir. Blueprint
limitleri 1–100 aralığındadır; varsayılan 50'dir.

Bu, ilişki hedeflerinin varlığını veya erişilebilirliğini doğrulamaz. Edition,
hedefleri aynı transaction içinde yetkilendirmeli ve tenant sınırında
doğrulamalıdır. Otomatik foreign key, cascade, eager loading, silme semantiği
ve cache invalidation üretilmez; bunlar provider/Edition sözleşmesi olarak
kalır. İlişki tablosu migration önerisidir ve yalnızca operatör migration
registry üzerinden uygulanır; runtime DDL yoktur.

## Sonuçlar

- Eski sürümler additive yeni tabloyla birlikte çalışabilir; rollback uygulama
  sürümünü geri alır, veri tablosunu otomatik silmez.
- SQLite ve PostgreSQL için ortak TEXT/INTEGER/placeholder yaklaşımı
  kullanılır. Yeni adapter'lar transactional replace, ordered bounded list ve
  tenant predicate davranışını sertifikalandırmalıdır.
- İlişki ID'leri için cache üretilmez; Edition cache kullanırsa successful
  replace sonrasında kendi anahtarlarını invalidate etmelidir.
- İlişki tablosu/listesi kaynak başına sınırlı olsa da toplam kaynak/tenant
  sayısı sınırsızdır; operasyonel kapasite sınırları Edition profilinde
  tanımlanmalıdır.

## Doğrulama

Generator testi normalized DDL ve generated API'yı kontrol eder. Örnek Edition
SQLite testi migration'ı çalıştırır; aynı tenant'ta sıralı list, farklı tenant
izolasyonu, dış transaction'a katılım ve duplicate/limit reddini doğrular.
PostgreSQL'e karşı canlı provider sertifikasyonu bu ADR'nin yerel test
kapsamında değildir ve release gate olarak açık kalır.
