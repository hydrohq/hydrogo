# Module specs

`v0/` dizini, Admin Studio ve generator için sürümlü collection kaynaklarını taşır. Bir spec'i inceleme amaçlı artifact'lere derlemek için:

```powershell
go run ./cmd/hydro generate module --spec module-specs/v0/testimonial.yaml --out .tmp/module-output
```

Bu komut runtime DDL çalıştırmaz, HTTP route kaydetmez ve var olan modülü etkinleştirmez. `generated/` altında yalnızca incelemeye açık model, repository, transport, migration taslağı ve Admin Studio manifest'i üretir. Kaynak şema v0 fail-closed çalışır: bilinmeyen alan, desteklenmeyen sürüm, secret-benzeri alan adı veya destructive intent reddedilir.
