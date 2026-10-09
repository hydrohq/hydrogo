# Native clean scaffold Blueprint validation — 2026-10-10

## Verified locally

Created a clean Edition with the CLI, generated an `article` resource with
SQLite and PostgreSQL proposals, then ran `hydro project check`:

```sh
GOCACHE=/tmp/hydro-cover-cache go run ./cmd/hydro new audit-edition \
  --root /tmp/hydro-native-admin-audit2 --no-demo
GOCACHE=/tmp/hydro-cover-cache go run ./cmd/hydro resource create \
  --name article --field title:text \
  --out /tmp/hydro-native-admin-audit2/audit-edition \
  --dialect sqlite --dialect postgres
GOCACHE=/tmp/hydro-cover-cache go run ./cmd/hydro project check \
  --config /tmp/hydro-native-admin-audit2/audit-edition/config.yaml
```

The resource command generated 44 files under `generated/`, including the
Blueprint spec, Admin registry, deterministic `RegisterGeneratedAdmin`
selection, runtime wiring checklist, generated tests, and migration metadata.
The checker passed and confirmed the generated artifact lock/aggregate state.
The proposal declares additive SQLite/PostgreSQL compatibility and
restore/forward-only recovery; no DDL was applied. The generated Admin
registration contains `article` without a second resource-ID edit.

## Acceptance boundary

The fresh scaffold reports `core=licensed-runtime`, has no `go.mod`, and its
README says HydroGo Core is distributed separately. Therefore this local run
proves static Blueprint generation and project validation only. It does not
prove that generated Go packages compile in the customer Edition, that
runtime API/Admin routes activate, that an administrator can log in, or that
the first CRUD works in a browser.

The generated `RUNTIME_WIRING.md` still requires Edition-owned runtime steps:
bind generated API resources to a SQL connection, policy and tenant resolver;
call the generated Admin registration with API/auth/CSRF/page-renderer
callbacks; bind resource-level affordances; review/apply the proposal through
the module migration registry; and add domain, readiness, and deployment
checks. This separation is intentional: Blueprint metadata cannot infer
authorization or silently activate migrations/routes.

The remaining native golden path requires a valid private runtime distribution
and license, then a real Edition composition and browser run. The plan keeps
that gate open; the static CLI check is not counted as CRUD acceptance. No
Payload speed/parity claim is made.
