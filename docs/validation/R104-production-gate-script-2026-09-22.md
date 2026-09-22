# R104 — Production gate runner compatibility

**Tarih:** 2026-09-22  
**Durum:** PASS (script parse/entry gate)

`scripts/verify-production-gates.ps1` Windows PowerShell 5 ve PowerShell 7
üzerinde parse edilebilir hale getirildi. Önceki Unicode em-dash karakterleri
Windows PowerShell 5'in kaynak kodu yorumlayıcısında yanlış string-terminator
hatasına yol açıyordu; mesajlar ASCII `-` ile normalize edildi.

Komut artık parser hatası vermeden beklenen fail-closed kontrolüne ulaşıyor:

```text
HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped.
```

Bu makinede PostgreSQL URL'si bu çalıştırma sırasında sağlanmadığı için
production gate suite çalıştırılmadı. Bu, kapının başarısızlığı değil,
zorunlu dış servis yapılandırmasının eksikliği olarak raporlanır.
