# R5 — CLI `new` uyumluluk takma adları

`create` komutları geriye dönük korunurken Payload benzeri kısa yazım eklendi:

```text
hydro module new --spec SPEC --out OUTPUT
hydro resource new --spec SPEC --out OUTPUT
hydro resource check --spec SPEC --root OUTPUT
hydro api diff --baseline OPENAPI.json --current OPENAPI.json
hydro check --config config.yaml --json
```

Windows ve WSL Ubuntu'da `cmd/hydro` testleri geçti. `resource new --name
article --field title:text` smoke testi deterministik scaffold üretti; mevcut
`create` komutları için regresyon testleri de çalıştırıldı.
`resource check` aynı spec ve çıktı köküyle generator uyumluluğunu da doğruladı.
`api diff`, mevcut OpenAPI compatibility kontrolünü SDK workflow'una taşıyan
uyumluluk takma adıdır; geriye dönük `validate api` komutu korunur.
`check`, `project check` için üst seviye SDK kısayoludur.

WSL komutu:

```text
go test ./cmd/hydro ./internal/generator ./internal/sdkdist
PASS
```
