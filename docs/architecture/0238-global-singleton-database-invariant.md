# ADR-0238: Enforce global singleton identity at the database boundary

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** Generated global resource migrations
- **İlgili:** ADR-0224 (versioned global resource contract)

## Karar

`type: global` kaynaklarının generated migration'ı `id = 'global'` CHECK
kısıtını taşır. Generated validation/transport katmanı aynı sabit kimliği
normalize etmeye devam eder; veritabanı kısıtı ise doğrudan SQL yazımı veya
yanlış bir adapter kullanımıyla ikinci singleton satırı oluşturulmasını
engeller.

Kısıt SQLite ve PostgreSQL tarafından desteklenen provider-neutral bir
invariant'tır. Migration hâlâ yalnızca reviewable bir artifact'tır; runtime
DDL çalıştırılmaz. Mevcut production tablosuna uygulanması expand/migrate
adımı ve önceden yapılacak duplicate-ID temizliği ile Edition/operator
sorumluluğundadır.

## Sonuç

- Global transport'un stable ID sözleşmesi storage katmanında da korunur.
- İki veritabanı adapter'ı arasında davranış farkı azalır.
- Eski, geçersiz ID'ler otomatik silinmez; migration güvenli biçimde fail
  eder ve operator'a düzeltme gereksinimini gösterir.
