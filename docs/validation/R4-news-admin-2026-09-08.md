# R4 News admin akışı doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R4 News inceleme/zamanlama/yayın/geri çekme  
**Durum:** Başarılı; domain, generation guarded worker, JSON/public ve protected admin HTML akışları hazır.

News admin ekranı draft düzenleme, schedule, publish ve retract işlemlerini aynı `news.Service` üzerinden çalıştırır. Her mutation formunda CSRF token ve optimistic `version` alanı vardır; editor/admin yetkisi router'ın protected sınırıyla birlikte kontrol edilir. Schedule worker job generation'ı payload'da taşır ve repository atomik generation kontrolüyle eski job'ı etkisizleştirir.

Çalıştırılan kontroller:

```text
go test ./modules/news/transport ./...
go vet ./...
```

İkinci dış edition ayrıca Core'a import etmeden `EditorialDraft` kuralını uygular: başlık ve desk zorunludur; zamanlanmış haber için editoryal onay gerekir. `examples/news-edition/editorial_policy_test.go` bu davranışı doğrular. Kalan R4 işleri ortak kalıcı job backend'i/hook contract kit'i ve uyumluluk/SDK matrisi kapılarıdır.
