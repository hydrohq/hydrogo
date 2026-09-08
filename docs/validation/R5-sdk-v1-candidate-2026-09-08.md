# R5 SDK v1 aday yüzeyi doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 SDK v1 adayı  
**Durum:** Aday yüzey belirlendi; resmi v1 sertifikasyonu ve deprecation takvimi açık.

Pages ve News modülleri ile iki harici Client Edition'da gerçekten kullanılan küçük yüzey `extension/v1` altında sabitlendi: `Theme`, `Module`, `ModuleRoute`, typed view data ve API sürüm sabiti. `extension` kökündeki deneysel v0 API korunur; v1 adayı internal paketlere bağımlı değildir.

```text
go test ./extension/v1 ./...
go vet ./...
```

Resmi v1 için generator fixture'ı, PostgreSQL/SQLite sertifikasyonu, release imzası ve deprecation/support takvimi R5 kapılarında tamamlanmalıdır.
