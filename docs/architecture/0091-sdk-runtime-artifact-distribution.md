# ADR-0091: Public SDK ve private runtime artifact dağıtımı

## Status

Proposed

## Context

HydroGo’nun Payload benzeri kurulum deneyimi için Linux, Windows ve macOS’ta kurulabilen bir `hydro` CLI gerekir. Ancak HydroGo Core private kalmalı; müşteri projelerine Core kaynak kodu, kalıcı GitHub token’ı veya private repository clone’u dağıtılmamalıdır.

## Decision

Public SDK; CLI, generator, starter, contract, frontend adapter ve geliştirici araçlarını içerir. Core ve ticari modüller CI tarafından üretilmiş, imzalı runtime artifact’ları olarak dağıtılır. CLI, bir artifact/license broker üzerinden kısa ömürlü indirme yetkisi alır; GitHub App veya private registry credential’ları broker tarafında tutulur. `git ls-remote` yalnızca kurumsal teşhis komutu olabilir ve entitlement kontrolü yerine geçmez.

Projeler SDK/runtime/module sürümlerini ve hash’lerini `hydrogo.lock` ile kilitler. CLI checksum, imza, platform ve compatibility doğrulaması yapmadan runtime çalıştırmaz. Offline kullanım, imzalı lisans ve kontrollü artifact aktarımı ile ayrıca desteklenir.

Runtime cache yaşam döngüsü de dağıtım sözleşmesinin parçasıdır: artifact'lar
`root/<version>/<canonical-target>` altında tutulur. Cache temizliği yalnızca
explicit koruma listesiyle çalışır; önce dry-run planı üretilebilir, ardından
yalnızca normal dosyalar atomik olmayan ama sınırlı bir silme işleminde kaldırılır.
Symlink, traversal ve normal dosya olmayan girdiler reddedilir. Böylece eski
lock'ların ihtiyaç duyduğu runtime'lar yanlışlıkla temizlenmez.

## Consequences

Olumlu sonuçlar: Core kaynak kodu private kalır, kullanıcı deneyimi Node.js araçlarına yaklaşır, lisans/repository erişimi ayrıştırılır, air-gapped müşteriler desteklenebilir ve release bütünlüğü doğrulanabilir.

Maliyetler: Artifact broker, cross-platform signing, installer bakımı, revocation/rotation ve resmi destek operasyonu gerekir. Binary dağıtımı tersine mühendisliği imkânsız kılmaz; hukuki lisans ve ticari kontroller teknik imzayı tamamlamalıdır.

## References

- [ADR-0010: SDK lifecycle and Edition manifest](0010-sdk-lifecycle-and-edition-manifest.md)
- [ADR-0028: Core isolation and commercial licensing](0028-core-isolation-and-licensing.md)
- [Payload parity execution plan](../PAYLOAD_PARITY_EXECUTION_PLAN.md)
- [SDK/private runtime distribution plan](../HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md)
