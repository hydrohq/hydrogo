# R5 — CLI `new` uyumluluk takma adları

`create` komutları geriye dönük korunurken Payload benzeri kısa yazım eklendi:

```text
hydro module new --spec SPEC --out OUTPUT
hydro resource new --spec SPEC --out OUTPUT
```

Windows ve WSL Ubuntu'da `cmd/hydro` testleri geçti. `resource new --name
article --field title:text` smoke testi deterministik scaffold üretti; mevcut
`create` komutları için regresyon testleri de çalıştırıldı.

WSL komutu:

```text
go test ./cmd/hydro ./internal/sdkdist
PASS
```
