# R4 News schedule doğrulaması

**Tarih:** 2026-09-08  
**Durum:** İlk News domain/repository, JSON/public route ve bounded worker lease dilimi başarılı; tam admin UI ve kalıcı job backend kapısı açık.

`202609080012_news_items` migration'ı revision, optimistic version, yayınlanan revision, schedule generation ve zaman alanlarını ekler. `modules/news` service'i draft/update, immediate publish, retract ve generation-guarded scheduled publish akışlarını aynı transaction içinde audit/outbox writer'larına bağlanabilir şekilde sunar.

`internal/jobs.Queue` bounded/idempotent enqueue, lease expiry reclaim ve worker token doğrulamalı ack/nack davranışını sağlar. News worker stale generation işini ack'leyip yeniden yayınlamaz; geçici hatayı retry için nack'ler.

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

PostgreSQL contract testi `HYDRO_TEST_POSTGRES_URL` yoksa atlanır. JSON/public route'ları aynı service/repository projection'ını kullanır; worker lease davranışı in-process contract ile doğrulanır. Kalıcı job backend'i, tam admin UI ve ikinci Client Edition sonraki R4 kapılarıdır.

`examples/news-edition` ayrı Go module olarak yalnız `extension` yüzeyini import eder; `EditorialTheme` ile ikinci tema ve CI importguard/build kanıtı eklenmiştir. News modülünün internal SQL katmanı Client Edition'a açılmaz.
