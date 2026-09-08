# R3 cache generation ve bounded miss doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R3 / S05-C  
**Durum:** Yerel cache kapısı başarılı; blue/green ve ölçümlü DB kapasite tatbikatı açık.

`internal/cache.Cache.GetOrLoad` aynı anahtarın eşzamanlı miss'lerini tek loader çağrısında birleştirir. `Clear` sırasında generation artar; devam eden eski loader sonucu çağırana dönebilse de cache'i yeniden dolduramaz. Pages public route lookup bu sözleşmeyi kullanır.

Çalıştırılan kontroller:

```text
go test ./internal/cache ./tests/contract
go test ./...
go vet ./...
```

Kalan R3 işi blue/green deploy sırasında bypass/revalidation ve desteklenen Compact profilinde DB yükünün gerçekçi veri setiyle ölçülmesidir. L2/Distributed davranışı bu kapının parçası değildir.
