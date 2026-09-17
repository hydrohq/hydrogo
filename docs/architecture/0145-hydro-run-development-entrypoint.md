# ADR 0145: `hydro run` development entrypoint

- Status: Accepted
- Date: 2026-09-18

## Context

HydroGo'nun canlı geliştirme sunucusu daha önce yalnızca `hydro dev` adıyla
erişilebiliyordu; README'de ise doğrudan iç araç olan `go run ./tools/devserver`
komutu gösteriliyordu. Bu, kullanıcıların desteklenen CLI yüzeyi ile dahili
uygulama ayrıntısını karıştırmasına neden oluyor.

## Decision

`hydro run` canlı geliştirme için kanonik kullanıcı komutudur. Mevcut projeleri
kırmamak için `hydro dev` aynı seçeneklerle geriye dönük uyumluluk takma adı
olarak korunur. Her iki yol da aynı `devCommand` uygulamasını çağırır; dahili
`tools/devserver` yolu dokümantasyonun kullanıcıya dönük parçası değildir.

## Consequences

- README ve CLI workflow belgeleri yalnızca `hydro run` komutunu önerir.
- Eski betikler ve geliştirici alışkanlıkları `hydro dev` ile çalışmaya devam eder.
- Komut davranışı tek bir uygulama yolunda kaldığı için iki ayrı sunucu yaşam
  döngüsü veya konfigürasyon sözleşmesi oluşmaz.
