# S01 Client Edition sınırı doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 / S01  
**Durum:** Kısmi başarı; modül build composition kanıtı bekliyor.

## Doğrulanan davranış

- `examples/client-edition` bağımsız bir Go module olarak Core kaynağını kopyalamadan `extension` paketini import edip derlendi.
- `tools/importguard`, Client Edition kaynaklarında HydroGo module yolundan yalnız ilan edilen `extension` prefix'ine izin verdi.
- Ayrı `tests/fixtures/internal-import-denied` module'ü `internal/server` import etmeye çalıştığında Go 1.27.0 derleyicisi `use of internal package ... not allowed` ile build'i reddetti.
- Kök test, vet ve importguard kontrolleri Windows üzerinde başarılı oldu. Aynı pozitif ve negatif fixture'lar CI workflow'una eklendi.

## Çalıştırılan komutlar

```text
go test ./...
go vet ./...
go run ./tools/importguard -root examples/client-edition
cd examples/client-edition && go build ./...
cd tests/fixtures/internal-import-denied && go build ./...  # başarısız olması beklendi
```

## Kalan kanıt

Pages modülünün route, asset, job ve migration contribution yüzeyi henüz uygulanmadı. Bu nedenle seçilmeyen modülün Edition build'ine hiçbir contribution veya byte eklemediği bölümü şimdi kanıtlanamaz. İlk gerçek modül kayıt yüzeyi oluşturulunca varsayılan ve `pages` seçili iki build karşılaştırılarak S01 kapatılacaktır.

`extension` yüzeyi yalnız `APIStatus` sabiti içerir ve deneysel v0 olarak etiketlidir. Genel bir Module interface'i bu deneyle erkenden dondurulmadı.
