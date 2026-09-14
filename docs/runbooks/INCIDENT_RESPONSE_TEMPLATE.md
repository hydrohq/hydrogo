# HydroGo incident response template

Bu şablon olay sırasında doldurulur; hassas değerler, token’lar ve kişisel
veriler buraya yazılmaz.

## Olay özeti

- Başlangıç (UTC):
- Bildiren kişi:
- Etkilenen edition/profile:
- Kullanıcı etkisi:
- İlk request ID / alarm:

## Güvenli ilk adımlar

1. Etki alanını `/healthz`, `/readyz`, metrics ve redacted log correlation ile
   doğrula.
2. Yeni deploy/migration/write işlemlerini durdur; kanıt toplamadan veri silme.
3. Session, media, queue ve backup durumunu ayrı ayrı kaydet.

## Degraded mode kararı

- Etkinleştirme koşulu ve onaylayan kişi:
- Kapatılan capability/module:
- Kullanıcıya gösterilecek sade mesaj:
- Geri dönüş koşulu:

## Rollback / restore

- Hedef binary/release manifest:
- Migration compatibility sonucu:
- Backup manifest ve verify sonucu:
- Session revoke / write fence / egress quarantine durumu:
- Geri dönüşü onaylayan kişi:

## Kapanış

- Bitiş (UTC):
- Doğrulanan SLO/SLI etkisi:
- Kalıcı düzeltme task’ı:
- ADR / validation raporu:
- Hassas veri sızıntısı incelemesi:
