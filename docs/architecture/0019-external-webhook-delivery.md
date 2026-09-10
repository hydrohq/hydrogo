# ADR-0019: İmzalı dış webhook teslimi

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0017](0017-commerce-consistency-and-payments.md)

Dış sistemlere olay teslimi, sipariş transaction'ından sonra çalışan outbox tüketicisinin kullandığı ayrı bir dispatcher sözleşmesiyle yapılır. Dispatcher topic allowlist uygular, gövdeyi HMAC-SHA256 ile imzalar, `Idempotency-Key` olarak olay kimliğini gönderir ve HTTP timeout kullanır.

Teslim bounded retry/backoff ile sınırlıdır; dış endpoint hatası outbox olayını lease'sinden çıkarıp sonraki zamana planlayan worker politikasına bırakılır. Dispatcher kalıcı kayıt tutmaz ve outbox lease'inin yerine geçmez. Endpoint URL'si yalnızca HTTP/HTTPS olabilir; sırlar environment yapılandırmasından gelir ve response gövdesi sınırlı okunur. `hydro serve`, endpoint yapılandırılmışsa bu worker'ı başlatır; yapılandırılmamışsa kaynak tüketmeden devre dışı kalır.
