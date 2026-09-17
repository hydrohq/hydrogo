# R61 — Commercial license issuer boundary (2026-09-17)

## Kapsam

Vendor-only lisans üretimi ile müşteri runtime doğrulamasının ayrıldığı doğrulandı. Bu kayıt teknik bir release gate'tir; hukuki lisans metninin hukuk danışmanı incelemesinin yerine geçmez.

## Çalıştırılan kontroller

```text
gofmt -w cmd/hydro/license_cmd.go cmd/hydro/main_test.go cmd/hydro-issuer/main.go
go test ./cmd/hydro ./cmd/hydro-issuer ./internal/licensing
```

Sonuç: başarılı.

## Kabul kriterleri

- `hydro license issue` ve `hydro license keygen` müşteri runtime'ında başarısız olur: sağlandı.
- `hydro-issuer issue` ve `hydro-issuer keygen` vendor-only binary'de bulunur: sağlandı.
- `hydro license verify/status` mevcut lisansları doğrulamaya devam eder: sağlandı.
- Signed `key_id` eşleşmesi ve yanlış slot reddi test edildi: sağlandı.
- Kısa rotasyon geçişinde eski/yeni trusted public key listesinin birlikte doğrulanması test edildi: sağlandı.
- Özel anahtar runtime kaynaklarında veya release dokümantasyonundaki müşteri komutlarında kullanılmaz: sağlandı.

## Açık operasyonel işler

- Release pipeline'ında issuer artifact'ının müşteri paketinden ayrılması.
- Resmi trusted public key'in release secret/ldflags akışında rotasyonu.
- Ticari lisans ve SLA metinlerinin yetkili hukuk incelemesi.
