# R5 deterministic generator başlangıç doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 / generator ve S08 başlangıcı  
**Durum:** Spec doğrulama, Pages/News collection fixture'ları ve derlenebilir CRUD repository/transport üretimi başarılı.

`internal/generator` collection spec'ten generated model, CRUD repository, transport route, migration ve contract skeleton'ı ile spec hash'li `generator.lock` üretir. Çıktı yalnız `generated/` altına yazılır; `custom/` dosyalarına dokunulmaz. Secret alan adları, eksik permission/dialect ve destructive migration spec'leri deny edilir. Runtime otomatik DDL veya yetki kararı üretilmez. Pages ve News fixture'ları iki kez üretimde aynı hash ve byte çıktısını verir, üretilen Go paketi derlenir; yayınlanmış migration dosyası elle değiştirilmişse `ErrPublishedMigration` ile reddedilir.

Çalıştırılan kontrol:

```text
go test ./internal/generator
```

Test ikinci üretimde generated dosyanın byte-for-byte aynı kaldığını, custom dosyanın hash'inin korunduğunu, Pages/News fixture hash'lerini, CRUD repository/transport yöntemlerini, derlenebilirliği ve migration değişmezliğini doğrular.
