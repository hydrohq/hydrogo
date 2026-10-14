# Generated Admin CRUD role and state validation — 2026-10-10

Expanded the generated Admin handler capability test to cover valid-CSRF
create, update, and delete attempts for Admin, Editor, Viewer, and unknown
roles, then exercises explicit Edition capability overrides. The test uses a
stateful Edition API fake and checks resulting record state: Admin can
create/update/delete; Editor can create/update but cannot delete; an Admin
delete-deny override removes only that capability; a read-only Editor override
removes create/update; and a Viewer create grant permits only create. Viewer
and unknown roles otherwise cannot mutate. Denied mutations leave existing
records unchanged.

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
all possible permission override combinations, browser menu visibility, or
PostgreSQL; those remain separate acceptance cells.
