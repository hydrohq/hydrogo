# ADR-0239: Generated globals use a singleton admin surface

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** Generated Admin Studio templates for `type: global`
- **İlgili:** ADR-0224, ADR-0238

## Karar

Generated admin list templates for global resources must not render collection
tables, pagination, search, or bulk selection. They render a bounded singleton
notice; the generated form remains the editing surface and the transport keeps
the stable `global` ID contract.

Bu ayrım UI'nin collection route'u varmış gibi yanlış bir kullanım üretmesini
önler. Authorization, save policy, audit and publication semantics remain
Edition/module-owned; the Core only shapes the safe generated surface.
