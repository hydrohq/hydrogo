# R5 deterministic generator başlangıç doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 / generator ve S08 başlangıcı  
**Durum:** Spec doğrulama ve deterministic iskelet başarılı; gerçek modül üretim fixture'ları ve published migration sertifikasyonu açık.

`internal/generator` collection spec'ten generated model/repository/transport/migration/contract iskeleti ve spec hash'li `generator.lock` üretir. Çıktı yalnız `generated/` altına yazılır; `custom/` dosyalarına dokunulmaz. Secret alan adları, eksik permission/dialect ve destructive migration spec'leri deny edilir. Runtime otomatik DDL veya yetki kararı üretilmez.

Çalıştırılan kontrol:

```text
go test ./internal/generator
```

Test ikinci üretimde generated dosyanın byte-for-byte aynı kaldığını ve custom dosyanın hash'inin korunduğunu doğrular. R5 S08 için Pages/News gerçek spec'leriyle iki kez üretim, migration değişmezliği ve diff raporu ayrıca gereklidir.
