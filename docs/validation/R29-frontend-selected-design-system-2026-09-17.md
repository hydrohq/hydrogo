# R29 — Frontend starter selected design-system metadata

**Date:** 2026-09-17  
**Status:** Başarılı (yerel sözleşme)

## Amaç

`hydro project new --design-system ...` seçiminin generated frontend starter
manifestine taşınmasını doğrulamak. Önceki manifest yalnızca desteklenen
adapter listesini yazıyor, seçili adapterı tüketiciye bildirmiyordu.

## Değişiklik

`frontend/starter.json` artık iki ayrı alan taşır:

- `design_system`: bu projenin seçtiği varsayılan provider;
- `design_systems`: Client Edition'ın seçebileceği provider-agnostic adapter
  listesi (`native`, `shadcn`, `material`, `carbon`, `headless`, `vanilla`).

Core herhangi bir provider runtime'ı import etmez; alanlar yalnızca manifest
metadatasıdır. Mevcut tema manifestindeki `design_system` değeri ile aynı
seçim kullanılır.

## Doğrulama

Komut:

```powershell
go test ./internal/projectgen -count=1
```

Sonuç: **PASS**. Testler varsayılan `native` seçimini, adapter listesini ve
`DesignSystem: "shadcn"` ile üretilen manifestte seçimin korunmasını doğrular.

## Sınır

Bu kanıt provider CSS/runtime paketlerinin kurulumu veya gerçek browser görsel
uyumluluğu değildir. Bu bağımlılıklar Client Edition'a aittir ve ayrı browser
E2E/production gate kapsamında doğrulanır.
