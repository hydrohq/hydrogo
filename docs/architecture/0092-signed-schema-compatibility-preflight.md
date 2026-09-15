# ADR-0092: Signed schema compatibility preflight

## Status

Proposed

## Context

SDK ile dağıtılan runtime'lar blue/green geçişte aynı migration geçmişini
paylaşır. Ham SQL'i veya veritabanı lehçesini public CLI'a açmak hem bilgi
hiyerarşisini hem de Core sınırını bozar.

## Decision

`hydro schema diff`, iki imzalı release manifestinin migration checksum
ledger'ını karşılaştırır. Trusted public key doğrulaması geçmeden karşılaştırma
yapılmaz. Yeni migration ID'leri additive olarak raporlanır; mevcut ID'nin
silinmesi veya checksum değişimi uyumsuzluk hatasıdır. Gerçek migration
uygulaması bu komutun sorumluluğu değildir.

## Consequences

Release/blue-green preflight deterministik ve SQL lehçesinden bağımsız olur.
Manifest üretimi ve imzalama CI/release ortamında yapılmalıdır; boş veya yanlış
ledger bilgisi bilerek fail-closed davranır.

## References

- [ADR-0091: SDK runtime artifact distribution](0091-sdk-runtime-artifact-distribution.md)
- [ADR-0060: Migration metadata lint](0060-migration-metadata-lint.md)
