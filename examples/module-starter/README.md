# Module starter fixture

This directory is a deliberately small starting point for a new standard
collection. It demonstrates the safe authoring loop without copying Core
internals:

```powershell
hydro module create --spec .\collection.yaml --out .\generated
hydro validate --config .\config.yaml
hydro test --timeout 5m
```

The generated repository, service, transport, policy, migration, fixture,
admin manifest, and `module.yaml` are reviewable scaffolds. Add real
authorization, business rules, audit/outbox behavior, and migration metadata
before enabling the module in a customer edition.
