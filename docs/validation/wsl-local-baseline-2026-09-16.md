# WSL yerel test tabanı — 2026-09-16

**Ortam:** Windows WSL2, Ubuntu, `Linux DESKTOP-RL4NSEN 6.18.33.2-microsoft-standard-WSL2`, `x86_64`  
**Kullanıcı:** `testuser`  
**Go:** `go1.27.0 linux/amd64`

## Komut

```bash
cd /path/to/project
go test ./...
```

## Sonuç

Test paketi başarıyla tamamlandı. `cmd/hydro`, extension, internal paketleri, modules, migrations, contract testleri ve tools paketlerinde başarısız test bulunmadı. Bazı paketler kaynak indirimi sonrasında cache’den çalıştı; bu rapor full production certification yerine WSL yerel baseline kanıtıdır.

## Sınırlar

Bu çalışma aşağıdakilerin yerine geçmez:

- Temiz PostgreSQL hedefinde restore ve kapasite kanıtı
- Gerçek reverse-proxy blue/green geçişi
- Harici S3/CDN, SMTP ve backup sağlayıcı tatbikatı
- Cross-platform signed release
- Production RPO/RTO ve müşteri donanımı ölçümü

Kullanılan parola bu belgeye veya komut çıktısına yazılmamıştır.
