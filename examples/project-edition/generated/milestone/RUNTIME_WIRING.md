# milestone runtime wiring

This generated contract deliberately separates deterministic scaffolding from
application composition. Complete each item in the owning module before
enabling the resource; the generator never guesses authorization or runtime
dependencies.

## Registration checklist

- [ ] When this Edition's generated directory belongs to a Go module, call
  generated.RegisterGeneratedResources once with the configured SQL
  dialect/connection pool, an Edition-owned PolicyFor callback, and an
  Edition-owned ResolveTenant callback if any resource is tenant-scoped. The
  helper preflights every generated policy and required tenant resolver before
  registering any route. For partial resource activation or a non-module
  generated directory, call RegisterMilestoneRoute for each explicitly selected
  resource instead.
- Route contract: /api/v1/milestone

- [ ] Keep authentication, authorization and request limits in the outer
  stack; the aggregate helper does not install middleware or infer policy.


- [ ] Call generated.RegisterGeneratedAdmin once from the Edition composition
  root with the composed API handler, the Edition authorization callback, its
  CSRF boundary and typed page renderer. Its generated selection
  follows this Edition's Blueprint registry, so newly generated resources join
  the shared list/form surface without hand-editing a resource ID list.
  For a deployment that intentionally enables only some generated resources,
  call adminstudio.RegisterGenerated with that reviewed selection instead.
- [ ] Derive ResourceAccess from the same user, tenant and record rules as the
  resource API. It controls generated CRUD affordances and direct Admin routes;
  leaving it nil preserves legacy all-actions UI, while the API remains the
  authorization boundary for every caller.
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
