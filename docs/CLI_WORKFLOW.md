# HydroGo CLI workflow

The CLI help is intentionally task-oriented. Start with `hydro help` and use
the examples there as copyable, safe defaults.

## Create and validate a project

The SDK-oriented namespace keeps the legacy `hydro new` spelling available:

```text
hydro project new haber-merkezi --preset=news
cd haber-merkezi
hydro project check --config config.yaml
```

`project check` validates configuration, capacity profile and enabled-module
invariants without opening the database. It is safe to run before dependencies
are installed or started.

## Validate before runtime work

```text
hydro validate --config config.yaml
```

This is dependency-free and does not open the database. Fix the reported
configuration or module errors before running migrations or serving traffic.

## Develop and test a focused area

```text
hydro run --address 127.0.0.1:8090
hydro test --package ./modules/news/... --timeout 2m
```

`hydro run` canlı geliştirme için önerilen komuttur. Eski betiklerle uyumluluk
için `hydro dev` aynı seçenekleri kabul eden bir takma ad olarak tutulur.
Komut başladığında yerel Studio adresi varsayılan tarayıcıda açılır; bunu
`hydro run --open=false` ile kapatabilirsiniz.

Use a different address when a local process owns the default port. Narrowing
the package makes failures faster to reproduce; increase the timeout only when
the test is known to need it.

## Migration safety

Run migrations against a disposable database first. Applied migrations are
immutable: create a new migration for a correction and review both dialect
files and metadata before committing.

## Error hygiene

CLI errors should identify the failed action and a safe next step without
leaking DSNs, credentials, SQL, or cache/queue implementation details.
