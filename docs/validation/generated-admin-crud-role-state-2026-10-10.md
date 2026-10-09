# Generated Admin CRUD role and state validation — 2026-10-10

Expanded the generated Admin handler capability test to cover valid-CSRF
create, update, and delete attempts for Admin, Editor, Viewer, and unknown
roles. The test now uses a stateful Edition API fake and checks the resulting
record state: Admin can create/update/delete; Editor can create/update but
cannot delete; Viewer and unknown roles cannot mutate. Denied mutations leave
existing records unchanged.

Validation passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/adminstudio \
  -run TestGeneratedBasicCRUDUsesEditionRoleCapabilities -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/adminstudio -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/adminstudio
git diff --check
```

This is handler authorization and state-transition evidence against a
stateful API fake. It does not certify DB persistence, real session policy,
permission override combinations, browser menu visibility, or PostgreSQL;
those remain separate acceptance cells.
