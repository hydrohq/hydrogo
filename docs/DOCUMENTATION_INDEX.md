# HydroGo dokümantasyon haritası

Bu belge, mevcut kararların ve doğrulama kanıtlarının nerede tutulduğunu gösterir. Bir özelliğin “yapıldı” sayılması için yalnızca kod değil, ilgili karar, test/kanıt ve operasyon notu birlikte incelenmelidir.

## Nereden başlanır?

| İhtiyaç | Ana belge |
|---|---|
| Ürünün felsefesi ve sınırları | [HYDROGO_PHILOSOPHY.md](HYDROGO_PHILOSOPHY.md) |
| Kullanıcıya gösterilecek bilgi | [INFORMATION_HIERARCHY.md](INFORMATION_HIERARCHY.md) |
| Uygulama sırası | [implementation_plan.md](implementation_plan.md), [ROADMAP.md](ROADMAP.md) |
| Mevcut durum | [PROJECT_STATUS.md](PROJECT_STATUS.md) |
| Uygulama ve açık kapıların ayrıntılı durumu | [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) |
| Kalan gerçek-ortam işleri | [OPEN_GATES_AND_OPERATOR_ACTIONS.md](OPEN_GATES_AND_OPERATOR_ACTIONS.md), [IMPLEMENTATION_AUDIT_REMAINING.md](IMPLEMENTATION_AUDIT_REMAINING.md) |
| Mimari karar | [architecture/README.md](architecture/README.md) ve ilgili ADR |
| Yeni proje üretimi ve SDK CLI | [PROJECT_GENERATOR.md](PROJECT_GENERATOR.md), [CLI_WORKFLOW.md](CLI_WORKFLOW.md), [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md) |
| SDK ve extension | [SDK_CAPABILITY_MATRIX.md](SDK_CAPABILITY_MATRIX.md), [SDK_SUPPORT_POLICY.md](SDK_SUPPORT_POLICY.md) |
| Payload hız/parite planı | [PAYLOAD_PARITY_EXECUTION_PLAN.md](PAYLOAD_PARITY_EXECUTION_PLAN.md) |
| Core bloat ve Compact profil denetimi | [CORE_BLOAT_AUDIT.md](CORE_BLOAT_AUDIT.md) |
| Payload ölçüm yöntemi | [PAYLOAD_BENCHMARK_PROTOCOL.md](PAYLOAD_BENCHMARK_PROTOCOL.md) |
| HydroGo iterasyon baseline | [validation/R5-iteration-baseline-2026-09-16.md](validation/R5-iteration-baseline-2026-09-16.md) |
| Public SDK/private runtime | [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md), [ADR-0091](architecture/0091-sdk-runtime-artifact-distribution.md) |
| Binary-only customer package | [COMMERCIAL_LICENSING.md](COMMERCIAL_LICENSING.md), [R64 validation](validation/R64-binary-only-customer-package-2026-09-17.md), [ADR-0143](architecture/0143-binary-only-customer-distribution.md) |
| SDK artifact broker API | [HYDRO_SDK_BROKER_API.md](HYDRO_SDK_BROKER_API.md) |
| SDK broker client kanıtı | [validation/R5-sdk-broker-client-2026-09-16.md](validation/R5-sdk-broker-client-2026-09-16.md) |
| Deployment ve operasyon | [DEPLOYMENT.md](DEPLOYMENT.md), [OPERATIONS_RUNBOOK.md](OPERATIONS_RUNBOOK.md), [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md) |
| Güvenlik modeli | [security/THREAT_MODEL.md](security/THREAT_MODEL.md), [COMMERCIAL_LICENSING.md](COMMERCIAL_LICENSING.md) |
| Son güvenlik taraması | [validation/security-check-2026-09-16.md](validation/security-check-2026-09-16.md), [`security-report/`](../security-report/) |
| Local production/capacity kanıtı | [R65](validation/R65-local-production-and-capacity-2026-09-17.md) |
| Güvenlik skill entegrasyonu | [SECURITY_CHECK_INTEGRATION.md](SECURITY_CHECK_INTEGRATION.md) |
| Frontend ve Admin Studio | [FRONTEND_ARCHITECTURE.md](FRONTEND_ARCHITECTURE.md), [HEADLESS_ADMIN_STUDIO_PLAN.md](HEADLESS_ADMIN_STUDIO_PLAN.md), [UI_DESIGN_SYSTEM.md](UI_DESIGN_SYSTEM.md) |

## Belge türleri

- `docs/architecture/`: Neden bu kararın alındığı; kabul edilmeden önce Proposed olarak işaretlenir.
- `docs/validation/`: Gerçekten çalıştırılmış komut, ortam, beklenen/gerçek sonuç ve sınırlamalar.
- `docs/*_PLAN.md` veya backlog belgeleri: Yapılacak iş; başarı kanıtı değildir.
- `PROJECT_STATUS.md`: Özet durum; üretim garantisi olarak yorumlanmaz.

## Durum dili

- **Implemented/Closed:** Kod ve ilgili test/kanıt mevcut.
- **Planned:** Tasarım veya görev tanımlı; uygulama kanıtı yok.
- **Open gate:** Gerçek ortam, harici sağlayıcı, hukuki karar veya release operasyonu bekliyor.
- **Not supported:** Bilinçli olarak kapsam dışı; alternatif ve gerekçe yazılmalı.

## Her değişiklikte güncellenecekler

1. Etkilenen ADR veya yeni ADR.
2. Uygulama ve contract testleri.
3. Migration/compatibility metadata.
4. İlgili validation raporu.
5. Roadmap/status ve kapasite/deployment notu.
6. Kullanıcıya dönük metinler için bilgi hiyerarşisi kontrolü.

Bir validation raporu yoksa sonuç “başarılı” diye sunulmaz; planlanan iş açık bırakılır.
