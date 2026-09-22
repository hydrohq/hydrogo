# ADR-0222: Backup manifest Edition/retention/encryption sınırı

## Durum

Kabul edildi — 2026-09-22.

## Karar

Mevcut SQLite/PostgreSQL backup akışı korunur; manifest artık isteğe bağlı
`edition`, `retention_class` ve dış KMS/key-escrow referansı (`encryption_ref`)
alanlarını taşır. HydroGo anahtar veya şifreleme sırrı üretmez ve backup'ı
paralel bir sistemle yeniden yazmaz. Alanlar deployment environment'tan
bounded metadata olarak alınır; gerçek şifreleme ve saklama işlemi harici
operator/KMS katmanındadır.

## Operasyon

`HYDRO_BACKUP_RETENTION_CLASS` ile yasal saklama sınıfı, 
`HYDRO_BACKUP_ENCRYPTION_REF` ile KMS kayıt referansı sağlanır. Restore öncesi
manifest doğrulama ve mevcut staging/quarantine kontrolleri zorunludur.
