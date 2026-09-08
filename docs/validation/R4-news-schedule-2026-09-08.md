# R4 News schedule doğrulaması

**Tarih:** 2026-09-08  
**Durum:** İlk News domain/repository ve JSON/public route dilimi başarılı; worker lease/idempotency ve tam admin UI kapısı açık.

`202609080012_news_items` migration'ı revision, optimistic version, yayınlanan revision, schedule generation ve zaman alanlarını ekler. `modules/news` service'i draft/update, immediate publish, retract ve generation-guarded scheduled publish akışlarını aynı transaction içinde audit/outbox writer'larına bağlanabilir şekilde sunar.

Contract testi SQLite'da şunları kanıtlar:

- stale editor update reddedilir;
- schedule generation yanlışsa yayın reddedilir;
- zamanı gelen doğru generation yayınlanır;
- published projection route üzerinden okunur;
- retract sonrası public route bulunamaz.

```text
go test ./tests/contract -run NewsContract -count=1
go test ./...
```

PostgreSQL contract testi `HYDRO_TEST_POSTGRES_URL` yoksa atlanır. JSON/public route'ları aynı service/repository projection'ını kullanır; worker yeniden yetki kontrolü, lease/idempotency ve ikinci Client Edition sonraki R4 kapılarıdır.
