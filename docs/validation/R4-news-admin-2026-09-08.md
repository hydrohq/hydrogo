# R4 News admin akışı doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R4 News inceleme/zamanlama/yayın/geri çekme  
**Durum:** Başarılı; domain, generation guarded worker, durable scheduler bootstrap, JSON/public ve protected admin HTML akışları hazır.

News admin ekranı draft düzenleme, schedule, publish ve retract işlemlerini aynı `news.Service` üzerinden çalıştırır. Her mutation formunda CSRF token ve optimistic `version` alanı vardır; editor/admin yetkisi router'ın protected sınırıyla birlikte kontrol edilir. Schedule worker job generation'ı payload'da taşır ve repository atomik generation kontrolüyle eski job'ı etkisizleştirir. `cmd/hydro serve` artık due scheduled kayıtlarını beş saniyelik bounded polling ile kalıcı `jobs` store'a aktarır ve lease worker'ı çalıştırır; enqueue tekrarları job kimliğiyle idempotent kalır.

Çalıştırılan kontroller:

```text
go test ./modules/news/transport ./...
go vet ./...
```

İkinci dış edition ayrıca Core'a import etmeden `EditorialDraft` kuralını uygular: başlık ve desk zorunludur; zamanlanmış haber için editoryal onay gerekir. `examples/news-edition/editorial_policy_test.go` bu davranışı doğrular. Bu kapının kalan işi, gerçek deployment'ta worker process ayrıştırması ve kapasite ölçümüdür.
