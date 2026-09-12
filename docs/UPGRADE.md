# HydroGo Upgrade & Migration Runbook

Bu doküman, HydroGo Core ve Client Edition projelerinin sıfır kesintiyle güncellenmesi, migrasyon güvenliği ve geri alma (rollback) prosedürlerini açıklar.

---

## 1. Yükseltme Öncesi Kontrol (`hydro upgrade check`)

Yeni bir Core sürümüne geçmeden önce mevcut veritabanı şemasının ve konfigürasyonun yeni sürüme hazır olduğunu doğrulamak için `upgrade check` komutunu çalıştırın:

```powershell
hydro upgrade check --config config.yaml
```

Çıktı örneği:
```text
upgrade-ready profile=compact database=sqlite
```

### Yapılan Kontroller:
1. Veritabanı bağlantısı ve erişilebilirliği (`PingContext`).
2. Gömülü Core migrasyonlarının ledger uyumluluğu (`database.CheckCompatibilityForModules`).
3. Şema sürümünün geri alınamaz bir çatışma barındırmadığının tespiti.

---

## 2. Kilit Dosyası (`hydrogo.lock`) ve Core Uyumluluğu

Her yükseltme işlemi sırasında `hydrogo.lock` dosyası incelenir:
- **Core Version:** Müşteri projesinin test edildiği Core sürümünü belirtir.
- **Generator Version:** İskeletin üretildiği sürüm.
- **Features:** Lisanslanan ve aktif edilen modüller listesi.

Core güncellendiğinde `hydrogo.lock` içindeki sürüm bilgisi kontrollü biçimde güncellenir.

---

## 3. Katkısal (Additive) Migrasyon Prensibi

- HydroGo asla çalışma zamanında veya otomatik migrasyonlarda `DROP TABLE`, `DROP COLUMN` gibi yıkıcı (destructive) komutlar üretmez.
- Eski binary ile yeni binary'nin aynı veritabanı üzerinde kısa bir süre birlikte çalışabilmesi (Blue/Green uyumluluğu - ADR-0005) için tüm şema değişiklikleri katkısaldır.
- Yeni kolonlar `NULL` veya `DEFAULT` değerlerle eklenir.

---

## 4. Geri Alma (Rollback) Prosedürü

Eğer yeni sürümde bir sorun tespit edilirse:
1. Trafiği anında önceki binary'ye yönlendirin.
2. Katkısal migrasyonlar eski binary'nin çalışmasını engellemez.
3. Gerekirse son doğrulanmış yedekten geri yükleme adımlarını izleyin:
   ```powershell
   hydro restore verify --snapshot /backups/2026-09-12
   hydro restore stage --snapshot /backups/2026-09-12 --destination /data/staging --sessions-revoked --egress-quarantined --old-writes-blocked
   hydro restore activate-media --staging /data/staging/media --active /data/media --rollback /data/media_old
   ```
