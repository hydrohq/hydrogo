# R6 validated kit registry

Client Edition composition roots can use `NewResolvedKitRegistry` to validate
declared capability kits and expand only the allowlisted dependencies. Unknown
or duplicate kits fail closed; `commerce` resolves its declared `forms`
dependency. The legacy raw constructor remains for migration compatibility.

Validation:

```powershell
go test ./internal/adminstudio -run 'TestNewResolvedKitRegistryValidatesAndExpands'
```
