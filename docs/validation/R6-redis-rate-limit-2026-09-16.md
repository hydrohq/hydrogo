# R6 Redis rate-limit entegrasyon kanıtı (2026-09-16)

## Kapsam

WSL Ubuntu üzerinde çalışan yerel Redis/Valkey uyumlu sunucusuna bağlanılarak
`internal/security` Redis limiter sözleşmesi doğrulandı. Test yalnızca
`HYDRO_TEST_REDIS_URL` açıkça verildiğinde çalışır; servis yoksa varsayılan test
akışı dış bağımlılık başlatmaz ve testi skip eder.

## Komut ve sonuç

```bash
redis-cli ping
# PONG
cd /mnt/c/dev/cms
HYDRO_TEST_REDIS_URL=redis://127.0.0.1:6379/15 \
  go test ./internal/security -run TestRedisRateLimiterAtomicLimitAndTTL -count=1
# ok .../internal/security
```

Test ilk isteğin kabul edildiğini, aynı anahtarın ikinci isteğinin limit nedeniyle
reddedildiğini ve Redis anahtarında 30 saniyeyi aşmayan pozitif TTL bulunduğunu
kontrol eder. Anahtar test sonunda silinir; gerçek kullanıcı verisi kullanılmaz.

Bu kanıt atomic `INCR` + TTL davranışını kapsar. Multi-instance failover,
partition, clock-skew, persistence ve kapasite/soak testleri gerçek dağıtım
kapısı olarak [açık kapılar](../OPEN_GATES_AND_OPERATOR_ACTIONS.md) belgesinde
kalır.
