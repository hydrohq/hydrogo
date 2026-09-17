# R65 — Local production drills and capacity baseline (2026-09-17)

## Çalıştırılanlar

```text
pwsh -NoProfile -File scripts/verify-local-production-drills.ps1
pwsh -NoProfile -File scripts/measure-capacity.ps1 -OutputPath .tmp/gates/capacity-local-20260917.json -WarmupRequests 10 -SampleCount 3 -SampleIntervalMs 100
```

## Sonuç

- İzole SQLite binary/readiness smoke: PASS (`/healthz` ve `/readyz` 200).
- Backup, release ve portable contract testleri: PASS.
- Blue-green cache rollback stability (20 tekrar): PASS.
- Compact/Standard/Distributed local SQLite process baseline: PASS.
- PostgreSQL contract/restore: PENDING; `HYDRO_TEST_POSTGRES_URL` bu ortamda yapılandırılmadı.

## Baseline (Windows/Go 1.27, 10 warmup, 3 sample)

| Profil | Peak working set | Peak private memory | Handles |
|---|---:|---:|---:|
| Compact | 87.95 MiB | 119.35 MiB | 198 |
| Standard | 90.40 MiB | 122.07 MiB | 211 |
| Distributed | 94.65 MiB | 127.47 MiB | 297 |

Bu ölçüm tek prosesli SQLite smoke testidir; müşteri SLA’sı, PostgreSQL/Redis/S3/CDN kapasitesi veya Distributed sertifikasyonu değildir. Ham JSON kanıtı `.tmp/gates/capacity-local-20260917.json` dosyasına yazıldı ve geçici olduğu için Git’e alınmadı.
