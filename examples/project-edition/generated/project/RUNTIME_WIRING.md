# Project runtime wiring

This generated contract deliberately separates deterministic scaffolding from
application composition. Complete each item in the owning module before
enabling the resource; the generator never guesses authorization or runtime
dependencies.

## Registration checklist

- [ ] Call RegisterProjectRoute from the module composition root with the
  configured SQL dialect/connection pool and the Edition-owned policy.
- Route contract: /api/v1/project

- [ ] For tenant-scoped resources, provide the Edition-owned tenant resolver;
  keep authentication, authorization and request limits in the outer stack.


- [ ] Call generated.RegisterGeneratedAdmin once from the Edition composition
  root with the composed API handler, authenticated-admin authorization, the
  Edition CSRF boundary and its typed page renderer. Its generated selection
  follows this Edition's Blueprint registry, so newly generated resources join
  the shared list/form surface without hand-editing a resource ID list.
  For a deployment that intentionally enables only some generated resources,
  call adminstudio.RegisterGenerated with that reviewed selection instead.
- [ ] Apply the migration through the module migration registry for every
  declared dialect: postgres, sqlite.

- [ ] Add cache policy, invalidation and background-job ownership only when the
  module contract requires them; keep Compact profile resources bounded.
- [ ] Extend contract/fixture tests with domain invariants and include this
  resource in the module health/readiness checks.

## Optional capability kits

Selected kits: 

For each selected kit, bind its module adapter and permission surface explicitly;
an omitted kit must not add routes, migrations, workers or assets.

## Required evidence

Record the owning ADR, migration/rollback result, policy test, deployment
compatibility result and capacity impact in REVIEW.md before release.
