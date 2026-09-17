# R35 — Headless starter contract smoke E2E akışına bağlandı

**Date:** 2026-09-17  
**Status:** Başarılı (script/CI wiring)

## Kapsam

`scripts/run-e2e.ps1` artık generated browser journey başlamadan önce
`examples/headless-starter/npm test` komutunu çalıştırır. Böylece TypeScript
client contract smoke testi yalnızca geliştiricinin elle çağırdığı bir kontrol
olmaktan çıkar; mevcut Windows/CI E2E kapısının zorunlu ön koşulu olur.

## Doğrulama

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — TypeScript build ve `PASS: headless client contract smoke`.

Tam browser akışı için `HYDRO_E2E_ADMIN_PASSWORD` ile
`scripts/run-e2e.ps1` çalıştırılmalıdır. Bu faz dış servis, ödeme sağlayıcısı
veya production proxy kapılarını kapatmaz.
