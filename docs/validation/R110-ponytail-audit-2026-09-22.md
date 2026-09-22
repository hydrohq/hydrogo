# R110 — Core over-engineering audit

**Tarih:** 2026-09-22  
**Kapsam:** Whole repository, complexity/bloat only  
**Durum:** Lean boundary review; no safe deletion applied

Ponytail whole-repo taraması aşağıdaki adayları inceledi:

- `yagni` `extension.Module` eski sözleşmesi yeni `extension/v1.Module` ile
  örtüşüyor; ancak açık deprecation/compatibility kararı nedeniyle bugün
  silinemez. Replacement: migration window tamamlanınca tek v1 contract.
- `yagni` `observability.Exporter` tek built-in provider taşımıyor; bu bir
  provider-neutral boundary'dir, gereksiz abstraction değildir. Replacement:
  yok; external OTLP/exporter adapter'ları için korunur.
- `yagni` `ui.DesignSystem` registry'si birkaç class mapping'i içeriyor; bu
  Client Edition provider değişimini Core'dan ayıran bilinçli boundary'dir.
  Replacement: yok.
- `yagni` `database.Executor`/`QueryExecutor` küçük görünüyor; SQLite/Postgres
  adapter izolasyonu ve test doubles için gerekli olduğundan silinmemelidir.
  Replacement: yok.
- `stdlib`/`native`: bağımlılık manifestinde doğrudan stdlib ile güvenle
  değiştirilebilecek production dependency tespit edilmedi; templ, pgx,
  Redis, S3, OTEL, SQLite ve YAML gerçek capability sınırlarıdır.

**Sonuç:** Davranışı bozmadan güvenle silinebilecek satır: 0; güvenle
kaldırılabilecek dependency: 0. Deprecated `extension.Module` için ileride
ayrı removal ADR'si açılabilir; bugün kaldırılması Edition compatibility'yi
bozar.
