# R6 generated OpenAPI contract validation

`hydro resource new` now emits an OpenAPI document that is useful to a client
generator instead of only a nominal success response. The generated list
operation documents the bounded pagination/filter/sort inputs and the stable
JSON error shape used by the runtime.

Validation:

```powershell
go test ./internal/generator
```

The generator test asserts `page_size`, `HydroAPIError`, and the `429` response
are present in the emitted document. Runtime authorization and provider-specific
business semantics remain explicit module responsibilities.

The compatibility preflight also rejects removal of a published request
parameter or request body, and rejects changing an optional parameter to
required. This prevents generated clients from silently breaking when a
contract is tightened.
