# Article runtime wiring

This generated contract deliberately separates deterministic scaffolding from
application composition. Complete each item in the owning module before
enabling the resource; the generator never guesses authorization or runtime
dependencies.

## Registration checklist

- [ ] Register the repository with the module composition root and pass the
  configured SQL dialect/connection pool.
- [ ] Register the service and transport under the versioned route
  /api/v1/article, including authentication, authorization and request limits.
- [ ] Bind the generated form/table/admin manifest to the authenticated admin
  surface; add CSRF, validation, error and empty-state handling.
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
