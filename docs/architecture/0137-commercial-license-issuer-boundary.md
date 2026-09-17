# ADR-0137: Commercial license issuer boundary

## Durum

Accepted

## Bağlam

HydroGo müşteri runtime'ı offline lisans doğrulaması yapmalıdır; ancak lisans üretimi ve özel anahtar erişimi müşteri dağıtımında bulunmamalıdır. Aynı binary içinde `issue` ve `keygen` komutlarının bulunması, yanlışlıkla özel anahtarın müşteri ortamına taşınması ve operasyonel sınırların bulanıklaşması riskini artırır.

## Karar

- `cmd/hydro-issuer` vendor-only lisans üretici binary'sidir ve `keygen` ile `issue` komutlarını taşır.
- `cmd/hydro` müşteri runtime'ıdır; yalnızca `license verify` ve `license status` sunar. `issue` ve `keygen` açık bir vendor-only hatasıyla reddedilir.
- Runtime, lisanstaki `public_key` alanını güven kaynağı kabul etmez; resmi release'e gömülen trusted public key ile doğrulama yapar.
- Lisans claims içindeki signed `key_id`, beklenen trusted key slot'una karşı doğrulanabilir; rotasyon sırasında eski ve yeni slotların birlikte kabul süresi release metadata'sında açıkça belirtilir.
- Özel anahtar yalnızca issuer operasyonunda tutulur. CI/CD secret store, dosya izinleri ve key rotation kayıtları release sürecinin sorumluluğundadır.
- Eski lisans formatı geriye dönük doğrulanır; format değişikliği gerektiğinde yeni bir ADR ve geçiş penceresi gerekir.

## Sonuçlar

- Müşteri artifact'ı imzalama yeteneği taşımaz ve issuer ile runtime arasındaki sınır görünür hale gelir.
- Vendor release pipeline'ı iki artifact üretmelidir: müşteri runtime ve erişimi sınırlı issuer.
- Offline lisanslarda merkezi iptal yoktur; kısa expiry/grace değerleri ve key rotation operasyonu ile risk azaltılır.

## Doğrulama

- `go test ./cmd/hydro ./cmd/hydro-issuer ./internal/licensing`
- Runtime testleri vendor-only komutların reddedildiğini ve verify/status akışlarının çalıştığını doğrular.
- Üretim release'inde issuer binary'si müşteri paketine dahil edilmez.
