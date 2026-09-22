# Module specs

Module specs are versioned Workspace inputs, not runtime configuration. They
describe a reusable capability that HydroGo can generate and review for a
Client Edition. They never invent customer authorization, execute DDL, or
activate routes/workers for an inactive module.

The `v0/` directory carries versioned collection resources for Admin Studio and the generator. To compile a spec into artifacts for review purposes:

```powershell
go run ./cmd/hydro generate module --spec module-specs/v0/testimonial.yaml --out .tmp/module-output
```

This command does not execute runtime DDL, register HTTP routes, or activate the existing module. It only generates models, repositories, transports, migration drafts, and an Admin Studio manifest open for review under `generated/`. The source schema v0 works fail-closed: unknown fields, unsupported versions, secret-like field names, or destructive intents are rejected.
