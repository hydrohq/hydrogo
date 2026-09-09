# R3 medya upload kapasite doğrulaması

## Kapsam

`media.Uploader` artık upload başına geçici dosya yazımı, SHA-256 hesaplama ve
görüntü boyutu doğrulamasını bounded bir semaphore üzerinden yürütür. Runtime
bootstrap'ı bu sınırı seçilen capacity profilinin `WorkerConcurrency` bütçesine
bağlar. Semaphore bekleyen bir request'in context'i iptal edilirse upload
başlamadan durur.

## Kanıt

- Commit: `674df9b`
- Ortam: Windows geliştirici ortamı, Go toolchain repository CI sürümüyle uyumlu
- Komutlar:
  - `go test ./internal/media ./cmd/hydro`
  - `go vet ./internal/media ./cmd/hydro`
  - `go test ./... -p 1`
  - `go vet ./...`
  - `git diff --check`
- Sonuç: tüm testler ve statik kontroller başarılı.
- Contract: `TestUploaderSaveContextHonorsCancellationWhileBusy`, tek slot doluyken
  bekleyen ikinci upload'ın `context.Canceled` ile bounded biçimde çıkmasını doğrular.

## Sınırlar

Bu kanıt process-local upload backpressure davranışını gösterir. Gerçek müşteri
donanımı üzerindeki throughput, disk I/O doygunluğu, çoklu instance toplam
kapasitesi ve object-storage provider davranışı bu raporla sertifikalanmaz.
