# R5 SDK atomic runtime install — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.InstallArtifact`, doğrulanmış runtime artifact’ını önce aynı
cache dizinindeki geçici dosyaya kopyalar, fsync ve ikinci checksum doğrulaması
sonrasında atomik rename yapar. Var olan aynı artifact yeniden kullanılır;
farklı dosya üzerine yazılmaz. Kaynak/destination symlink ve regular-file
kontrolleri uygulanır.

## Kanıt

- Windows: `go test ./internal/release ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/release ./internal/sdkdist` ve
  `go test ./...` başarılı.
- Testler geçerli install, idempotent tekrar, tampered source ve farklı
  destination overwrite reddini doğrular.

## Sınır

Bu işlem ağdan indirme veya broker yetkilendirmesi yapmaz. Kaynak dosyanın
manifest metadata’sı ile indirilmiş olması çağıranın sorumluluğundadır; burada
checksum ve atomik kurulum tekrar güvence altına alınır.
