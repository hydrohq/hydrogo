# R5 — Permission contract check

`hydro permissions check --spec SPEC [--json]` artık resource spec'inin izin
isimlerini offline doğrular. Boşluk, traversal benzeri adlar, büyük harf ve
duplicate izinler reddedilir; geçerli spec yalnızca koleksiyon ve izin sayısını
raporlar.

Windows:

```text
go test ./internal/generator ./cmd/hydro
PASS
```

WSL Ubuntu (`arda`): aynı hedef paketler exit code 0 ile geçti.
