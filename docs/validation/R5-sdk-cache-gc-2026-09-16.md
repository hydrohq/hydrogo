# R5 — SDK runtime cache yaşam döngüsü

## Kapsam

`ListCache` yalnızca `root/<version>` altındaki normal artifact dosyalarını
listeler. `PruneCache` korunacak sürümleri alır; dry-run planı üretir veya
korunmayan dosyaları siler. Symlink ve normal dosya olmayan girdiler reddedilir.

## Doğrulama

```text
go test ./internal/sdkdist
ok github.com/ardaozyildirim/hydrogo/internal/sdkdist
```

Testler dry-run'ın dosyaya dokunmadığını, apply modunun eski sürümü sildiğini,
güncel sürümü koruduğunu ve symlink'i reddettiğini kanıtlar.

CLI yüzeyi de doğrulanmıştır:

```text
hydro runtime cache list --root PATH
hydro runtime cache prune --root PATH --keep v2.0.0 --dry-run
hydro runtime cache prune --root PATH --keep v2.0.0 --confirm
```
