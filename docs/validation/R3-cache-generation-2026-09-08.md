# R3 cache generation ve bounded miss doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R3 / S05-C  
**Durum:** Yerel cache kapısı, in-flight generation bypass/revalidation ve stale refill reddi başarılı; gerçek blue/green trafik ve ölçümlü DB kapasite tatbikatı açık.

`internal/cache.Cache.GetOrLoad` aynı generation içindeki aynı anahtarın eşzamanlı miss'lerini tek loader çağrısında birleştirir. `Clear` sırasında generation artar; yeni istek eski in-flight loader'a katılmaz, eski sonuç çağırana dönebilse de cache'i yeniden dolduramaz. Pages public route lookup bu sözleşmeyi kullanır.

Pages service contract testi 64 eşzamanlı aynı route isteğini çalıştırır ve repository/DB lookup sayısının 1 olduğunu doğrular. Böylece bounded miss yükü yalnız cache primitive'inde değil, gerçek public lookup service sınırında da kanıtlanır.

Çalıştırılan kontroller:

```text
go test ./internal/cache ./tests/contract
go test ./...
go vet ./...
```

Kalan R3 işi gerçek blue/green deploy trafiği ve desteklenen Compact profilinde DB yükünün gerçekçi veri setiyle ölçülmesidir. L2/Distributed davranışı bu kapının parçası değildir.
