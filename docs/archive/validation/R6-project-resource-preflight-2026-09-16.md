# R6 project resource preflight validation

`hydro project check --config config.yaml` now discovers a sibling
`generated/` directory when present and validates every generated collection
from its embedded `spec.json`. A project with no generated directory keeps the
same dependency-free check path.

Validation:

```powershell
go test ./cmd/hydro -run 'TestProjectCheckValidatesGeneratedResourcesWhenPresent'
```

The test confirms a complete generated collection passes and that removing a
required generated artifact fails the project check before runtime startup.
