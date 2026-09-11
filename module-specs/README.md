# Module specs

The `v0/` directory carries versioned collection resources for Admin Studio and the generator. To compile a spec into artifacts for review purposes:

```powershell
go run ./cmd/hydro generate module --spec module-specs/v0/testimonial.yaml --out .tmp/module-output
```

This command does not execute runtime DDL, register HTTP routes, or activate the existing module. It only generates models, repositories, transports, migration drafts, and an Admin Studio manifest open for review under `generated/`. The source schema v0 works fail-closed: unknown fields, unsupported versions, secret-like field names, or destructive intents are rejected.
